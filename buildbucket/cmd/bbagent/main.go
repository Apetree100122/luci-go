// Copyright 2019 The LUCI Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Command bbagent is Buildbucket's agent running in swarming.
//
// This executable creates a luciexe 'host' environment, and runs the
// Buildbucket build's exe within this environment. Please see
// https://go.chromium.org/luci/luciexe for details about the 'luciexe'
// protocol.
//
// This command is an implementation detail of Buildbucket.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"sync/atomic"
	"time"

	"net/http"
	_ "net/http/pprof"

	"github.com/golang/protobuf/jsonpb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"go.chromium.org/luci/buildbucket"
	"go.chromium.org/luci/buildbucket/cmd/bbagent/bbinput"
	bbpb "go.chromium.org/luci/buildbucket/proto"
	"go.chromium.org/luci/common/clock"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/common/logging/gologger"
	"go.chromium.org/luci/common/sync/dispatcher"
	"go.chromium.org/luci/common/system/environ"
	"go.chromium.org/luci/grpc/grpcutil"
	"go.chromium.org/luci/lucictx"
	"go.chromium.org/luci/luciexe"
	"go.chromium.org/luci/luciexe/host"
	"go.chromium.org/luci/luciexe/invoke"
)

var maxReserveDuration = 2 * time.Minute

func main() {
	go func() {
		// serves "/debug" endpoints for pprof.
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()
	os.Exit(mainImpl())
}

func mainImpl() int {
	ctx := logging.SetLevel(gologger.StdConfig.Use(context.Background()), logging.Info)

	check := func(err error) {
		if err != nil {
			logging.Errorf(ctx, err.Error())
			os.Exit(1)
		}
	}

	outputFile := luciexe.AddOutputFlagToSet(flag.CommandLine)
	flag.Parse()
	args := flag.Args()

	if len(args) != 1 {
		check(errors.Reason("expected 1 argument: got %d", len(args)).Err())
	}

	input, err := bbinput.Parse(args[0])
	check(errors.Annotate(err, "could not unmarshal BBAgentArgs").Err())

	// We start with retries disabled because the dispatcher.Channel will handle
	// them during the execution of the user process; In particular we want
	// dispatcher.Channel to be able to move on to a newer version of the Build if
	// it encounters transient errors, rather than retrying a potentially stale
	// Build state.
	//
	// We enable them again after the user process has finished.
	bbclientRetriesEnabled := false
	bbclient, secrets, err := newBuildsClient(ctx, input.Build.Infra.Buildbucket, &bbclientRetriesEnabled)
	check(errors.Annotate(err, "could not connect to Buildbucket").Err())
	logdogOutput, err := mkLogdogOutput(ctx, input.Build.Infra.Logdog)
	check(errors.Annotate(err, "could not create logdog output").Err())

	var (
		cctx   context.Context
		cancel func()
	)
	if dl := lucictx.GetDeadline(ctx); dl.GetSoftDeadline() != 0 {
		softDeadline := dl.SoftDeadlineTime()
		gracePeriod := time.Duration(dl.GetGracePeriod() * float64(time.Second))
		cctx, cancel = context.WithDeadline(ctx, softDeadline.Add(gracePeriod))
	} else {
		cctx, cancel = context.WithCancel(ctx)
	}
	defer cancel()

	dispatcherOpts, dispatcherErrCh := channelOpts(cctx)
	buildsCh, err := dispatcher.NewChannel(cctx, dispatcherOpts, mkSendFn(cctx, secrets, bbclient))
	check(errors.Annotate(err, "could not create builds dispatcher channel").Err())
	defer buildsCh.CloseAndDrain(cctx)

	// from this point forward we want to try to report errors to buildbucket,
	// too.
	check = func(err error) {
		if err != nil {
			logging.Errorf(cctx, err.Error())
			buildsCh.C <- &bbpb.Build{
				Status:          bbpb.Status_INFRA_FAILURE,
				SummaryMarkdown: fmt.Sprintf("fatal error in startup: %s", err),
			}
			buildsCh.CloseAndDrain(cctx)
			os.Exit(1)
		}
	}

	cctx = setResultDBContext(cctx, input.Build, secrets)
	prepareInputBuild(cctx, input.Build)

	opts := &host.Options{
		BaseBuild:      input.Build,
		ButlerLogLevel: logging.Warning,
		ViewerURL: fmt.Sprintf("https://%s/build/%d",
			input.Build.Infra.Buildbucket.Hostname, input.Build.Id),
		LogdogOutput: logdogOutput,
		ExeAuth:      host.DefaultExeAuth("bbagent", input.KnownPublicGerritHosts),
	}
	cwd, err := os.Getwd()
	check(errors.Annotate(err, "getting cwd").Err())
	opts.BaseDir = filepath.Join(cwd, "x")

	exeArgs := append(([]string)(nil), input.Build.Exe.Cmd...)
	payloadPath := input.PayloadPath
	if len(exeArgs) == 0 {
		// TODO(iannucci): delete me with ExecutablePath.
		var exe string
		payloadPath, exe = path.Split(input.ExecutablePath)
		exeArgs = []string{exe}
	}
	exePath, err := filepath.Abs(filepath.Join(payloadPath, exeArgs[0]))
	check(errors.Annotate(err, "absoluting exe path %q", input.ExecutablePath).Err())
	if runtime.GOOS == "windows" {
		exePath, err = resolveExe(exePath)
		check(errors.Annotate(err, "resolving %q", input.ExecutablePath).Err())
	}
	exeArgs[0] = exePath

	initialJSONPB, err := (&jsonpb.Marshaler{
		OrigName: true, Indent: "  ",
	}).MarshalToString(input)
	check(errors.Annotate(err, "marshalling input args").Err())
	logging.Infof(ctx, "Input args:\n%s", initialJSONPB)

	shutdownCh := make(chan struct{})
	var statusDetails *bbpb.StatusDetails
	var subprocErr error
	builds, err := host.Run(cctx, opts, func(ctx context.Context, hostOpts host.Options) error {
		logging.Infof(ctx, "running luciexe: %q", exeArgs)
		logging.Infof(ctx, "  (cache dir): %q", input.CacheDir)
		invokeOpts := &invoke.Options{
			BaseDir:  hostOpts.BaseDir,
			CacheDir: input.CacheDir,
		}
		// Buildbucket assigns some grace period to the surrounding task which is
		// more than what the user requested in `input.Build.GracePeriod`. We
		// reserve the difference here so the user task only gets what they asked
		// for.
		deadline := lucictx.GetDeadline(ctx)
		toReserve := deadline.GracePeriodDuration() - input.Build.GracePeriod.AsDuration()
		logging.Infof(
			ctx, "Reserving %s out of %s of grace_period from LUCI_CONTEXT.",
			toReserve, lucictx.GetDeadline(ctx).GracePeriodDuration())
		dctx, shutdown := lucictx.TrackSoftDeadline(ctx, toReserve)
		go func() {
			select {
			case <-shutdownCh:
				shutdown()
			case <-dctx.Done():
			}
		}()
		subp, err := invoke.Start(dctx, exeArgs, input.Build, invokeOpts)
		if err != nil {
			return err
		}

		var build *bbpb.Build
		build, subprocErr = subp.Wait()
		statusDetails = build.StatusDetails
		return nil
	})
	if err != nil {
		check(errors.Annotate(err, "could not start luciexe host environment").Err())
	}

	var (
		finalBuild                *bbpb.Build = proto.Clone(input.Build).(*bbpb.Build)
		fatalUpdateBuildErrorSlot atomic.Value
	)

	go func() {
		// Monitors the `dispatcherErrCh` and checks for fatal error.
		//
		// Stops the build shuttling and shuts down the luciexe if a fatal error is
		// received.
		stopped := false
		for {
			select {
			case err := <-dispatcherErrCh:
				if !stopped && grpcutil.Code(err) == codes.InvalidArgument {
					close(shutdownCh)
					fatalUpdateBuildErrorSlot.Store(err)
					stopped = true
				}
			case <-cctx.Done():
				return
			}
		}
	}()

	// Now all we do is shuttle builds through to the buildbucket client channel
	// until there are no more builds to shuttle.
	for build := range builds {
		if fatalUpdateBuildErrorSlot.Load() == nil {
			buildsCh.C <- build
			finalBuild = build
		}
	}
	buildsCh.CloseAndDrain(cctx)

	// Now that the builds channel has been closed, update bb directly.
	updateMask := []string{
		"build.status",
		"build.status_details",
		"build.summary_markdown",
	}
	var retcode int

	fatalUpdateBuildErr, _ := fatalUpdateBuildErrorSlot.Load().(error)
	if finalizeBuild(ctx, finalBuild, fatalUpdateBuildErr, statusDetails, outputFile) {
		updateMask = append(updateMask, "build.steps", "build.output")
	} else {
		// finalizeBuild indicated that something is really wrong; Omit steps and
		// output from the final push to minimize potential issues.
		retcode = 1
	}
	bbclientRetriesEnabled = true
	_, bbErr := bbclient.UpdateBuild(
		metadata.NewOutgoingContext(cctx, metadata.Pairs(buildbucket.BuildTokenHeader, secrets.BuildToken)),
		&bbpb.UpdateBuildRequest{
			Build:      finalBuild,
			UpdateMask: &fieldmaskpb.FieldMask{Paths: updateMask},
		})
	if bbErr != nil {
		logging.Errorf(cctx, "Failed to report error %s to Buildbucket due to %s", err, bbErr)
		retcode = 2
	}

	if retcode == 0 && subprocErr != nil {
		errors.Walk(subprocErr, func(err error) bool {
			exit, ok := err.(*exec.ExitError)
			if ok {
				retcode = exit.ExitCode()
				logging.Infof(cctx, "Returning exit code from user subprocess: %d", retcode)
			}
			return !ok
		})
		if retcode == 0 {
			retcode = 3
			logging.Errorf(cctx, "Non retcode-containing error from user subprocess: %s", subprocErr)
		}
	}

	return retcode
}

// finalizeBuild returns true if fatalErr is nil and there's no additional
// errors finalizing the build.
func finalizeBuild(ctx context.Context, finalBuild *bbpb.Build, fatalErr error, statusDetails *bbpb.StatusDetails, outputFile *luciexe.OutputFlag) bool {
	if statusDetails != nil {
		if finalBuild.StatusDetails == nil {
			finalBuild.StatusDetails = &bbpb.StatusDetails{}
		}
		proto.Merge(finalBuild.StatusDetails, statusDetails)
	}

	// set final times
	now := timestamppb.New(clock.Now(ctx))
	finalBuild.UpdateTime = now
	finalBuild.EndTime = now

	var finalErrs errors.MultiError
	if fatalErr != nil {
		finalErrs = append(finalErrs, errors.Annotate(fatalErr, "fatal error in builbucket.UpdateBuild").Err())
	}
	if err := outputFile.Write(finalBuild); err != nil {
		finalErrs = append(finalErrs, errors.Annotate(err, "writing final build").Err())
	}

	if len(finalErrs) > 0 {
		errors.Log(ctx, finalErrs)

		// we had some really bad error, just downgrade status and add a message to
		// summary markdown.
		finalBuild.Status = bbpb.Status_INFRA_FAILURE
		originalSM := finalBuild.SummaryMarkdown
		finalBuild.SummaryMarkdown = fmt.Sprintf("FATAL: %s", finalErrs.Error())
		if originalSM != "" {
			finalBuild.SummaryMarkdown += "\n\n" + originalSM
		}
	}

	return len(finalErrs) == 0
}

func prepareInputBuild(ctx context.Context, build *bbpb.Build) {
	// mark started
	build.Status = bbpb.Status_STARTED
	now := timestamppb.New(clock.Now(ctx))
	build.StartTime, build.UpdateTime = now, now
	// TODO(iannucci): this is sketchy, but we preemptively add the log entries
	// for the top level user stdout/stderr streams.
	//
	// Really, `invoke.Start` is the one that knows how to arrange the
	// Output.Logs, but host.Run makes a copy of this build immediately. Find
	// a way to set these up nicely (maybe have opts.BaseBuild be a function
	// returning an immutable bbpb.Build?).
	build.Output = &bbpb.Build_Output{
		Logs: []*bbpb.Log{
			{Name: "stdout", Url: "stdout"},
			{Name: "stderr", Url: "stderr"},
		},
	}
	populateSwarmingInfoFromEnv(build, environ.System())
	return
}

// Returns min(1% of the remaining time towards current soft deadline,
// `maxReserveDuration`). Returns 0 If soft deadline doesn't exist or
// has already been exceeded.
func calcDeadlineReserve(ctx context.Context) time.Duration {
	curSoftDeadline := lucictx.GetDeadline(ctx).SoftDeadlineTime()
	if now := clock.Now(ctx).UTC(); !curSoftDeadline.IsZero() && now.Before(curSoftDeadline) {
		if reserve := curSoftDeadline.Sub(now) / 100; reserve < maxReserveDuration {
			return reserve
		}
		return maxReserveDuration
	}
	return 0
}

func resolveExe(path string) (string, error) {
	if filepath.Ext(path) != "" {
		return path, nil
	}

	lme := errors.NewLazyMultiError(2)
	for i, ext := range []string{".exe", ".bat"} {
		candidate := path + ext
		if _, err := os.Stat(candidate); !lme.Assign(i, err) {
			return candidate, nil
		}
	}

	me := lme.Get().(errors.MultiError)
	return path, errors.Reason("cannot find .exe (%q) or .bat (%q)", me[0], me[1]).Err()
}
