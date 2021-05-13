// Copyright 2015 The LUCI Authors.
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

package lib

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync/atomic"
	"time"

	"github.com/bazelbuild/remote-apis-sdks/go/pkg/client"
	"github.com/bazelbuild/remote-apis-sdks/go/pkg/command"
	"github.com/bazelbuild/remote-apis-sdks/go/pkg/digest"
	"github.com/bazelbuild/remote-apis-sdks/go/pkg/filemetadata"
	"github.com/bazelbuild/remote-apis-sdks/go/pkg/uploadinfo"
	"github.com/maruel/subcommands"
	"golang.org/x/sync/errgroup"

	"go.chromium.org/luci/auth"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/cipd/version"
	"go.chromium.org/luci/client/archiver/tarring"
	"go.chromium.org/luci/client/cas"
	"go.chromium.org/luci/client/internal/common"
	"go.chromium.org/luci/client/isolate"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/data/text/units"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/isolatedclient"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/common/runtime/profiling"
	"go.chromium.org/luci/common/system/filesystem"
)

type baseCommandRun struct {
	subcommands.CommandRunBase
	defaultFlags common.Flags
	logConfig    logging.Config // for -log-level, used by ModifyContext
	profiler     profiling.Profiler

	// Overriden in tests.
	casClientFactory func(ctx context.Context, instance string, opts auth.Options, readOnly bool) (*client.Client, error)
}

var _ cli.ContextModificator = (*baseCommandRun)(nil)

func (c *baseCommandRun) Init() {
	c.defaultFlags.Init(&c.Flags)
	c.logConfig.Level = logging.Warning
	c.logConfig.AddFlags(&c.Flags)
	c.profiler.AddFlags(&c.Flags)
}

func (c *baseCommandRun) Parse() error {
	if c.logConfig.Level == logging.Debug {
		// extract glog flag used in remote-apis-sdks
		logtostderr := flag.Lookup("logtostderr")
		if logtostderr == nil {
			return errors.Reason("logtostderr flag for glog not found").Err()
		}
		v := flag.Lookup("v")
		if v == nil {
			return errors.Reason("v flag for glog not found").Err()
		}
		logtostderr.Value.Set("true")
		v.Value.Set("9")
	}
	if err := c.profiler.Start(); err != nil {
		return err
	}
	return c.defaultFlags.Parse()
}

// ModifyContext implements cli.ContextModificator.
func (c *baseCommandRun) ModifyContext(ctx context.Context) context.Context {
	return c.logConfig.Set(ctx)
}

func (c *baseCommandRun) newCASClient(ctx context.Context, instance string, opts auth.Options, readOnly bool) (*client.Client, error) {
	factory := c.casClientFactory
	if factory == nil {
		factory = cas.NewClient
	}
	return factory(ctx, instance, opts, readOnly)
}

type commonServerFlags struct {
	baseCommandRun
	isolatedFlags isolatedclient.Flags
	authFlags     authcli.Flags

	parsedAuthOpts auth.Options
}

func (c *commonServerFlags) Init(authOpts auth.Options) {
	c.baseCommandRun.Init()
	c.isolatedFlags.Init(&c.Flags)
	c.authFlags.Register(&c.Flags, authOpts)
}

func (c *commonServerFlags) Parse() error {
	var err error
	if err = c.baseCommandRun.Parse(); err != nil {
		return err
	}
	if err = c.isolatedFlags.Parse(); err != nil {
		return err
	}
	c.parsedAuthOpts, err = c.authFlags.Options()
	return err
}

func (c *commonServerFlags) createAuthClient(ctx context.Context) (*http.Client, error) {
	// Don't enforce authentication by using OptionalLogin mode. This is needed
	// for IP-allowed bots: they have NO credentials to send.
	return auth.NewAuthenticator(ctx, auth.OptionalLogin, c.parsedAuthOpts).Client()
}

func (c *commonServerFlags) createIsolatedClient(authCl *http.Client) (*isolatedclient.Client, error) {
	userAgent := "isolate-go/" + IsolateVersion
	if ver, err := version.GetStartupVersion(); err == nil && ver.InstanceID != "" {
		userAgent += fmt.Sprintf(" (%s@%s)", ver.PackageName, ver.InstanceID)
	}
	return c.isolatedFlags.NewClient(isolatedclient.WithAuthClient(authCl), isolatedclient.WithUserAgent(userAgent))
}

type isolateFlags struct {
	// TODO(tandrii): move ArchiveOptions from isolate pkg to here.
	isolate.ArchiveOptions
}

func (c *isolateFlags) Init(f *flag.FlagSet) {
	c.ArchiveOptions.Init()
	f.StringVar(&c.Isolate, "isolate", "", ".isolate file to load the dependency data from")
	f.StringVar(&c.Isolate, "i", "", "Alias for -isolate")
	f.StringVar(&c.IgnoredPathFilterRe, "ignored-path-filter-re", "", "A regular expression for filtering away the paths to be ignored. Note that this regexp matches ANY part of the path. So if you want to match the beginning of a path, you need to explicitly prepend ^ (same for $). Please use the Go regexp syntax. I.e. use double backslack \\\\ if you need a backslash literal.")
	f.Var(&c.ConfigVariables, "config-variable", "Config variables are used to determine which conditions should be matched when loading a .isolate file, default: [].")
	f.Var(&c.PathVariables, "path-variable", "Path variables are used to replace file paths when loading a .isolate file, default: {}")
	f.BoolVar(&c.AllowMissingFileDir, "allow-missing-file-dir", false, "If this flag is true, invalid entries in the isolated file are only logged, but won't stop it from being processed.")
}

// RequiredIsolateFlags specifies which flags are required on the command line
// being parsed.
type RequiredIsolateFlags uint

const (
	// RequireIsolateFile means the -isolate flag is required.
	RequireIsolateFile RequiredIsolateFlags = 1 << iota
	// RequireIsolatedFile means the -isolated flag is required.
	RequireIsolatedFile
)

func (c *isolateFlags) Parse(cwd string, flags RequiredIsolateFlags) error {
	if !filepath.IsAbs(cwd) {
		return errors.Reason("cwd must be absolute path").Err()
	}
	for _, vars := range [](map[string]string){c.ConfigVariables, c.PathVariables} {
		for k := range vars {
			if !isolate.IsValidVariable(k) {
				return fmt.Errorf("invalid key %s", k)
			}
		}
	}
	// Account for EXECUTABLE_SUFFIX.
	if len(c.ConfigVariables) != 0 || len(c.PathVariables) > 1 {
		os.Stderr.WriteString(
			"WARNING: -config-variables and -path-variables\n" +
				"         will be unsupported soon. Please contact the LUCI team.\n" +
				"         https://crbug.com/907880\n")
	}

	if c.Isolate == "" {
		if flags&RequireIsolateFile != 0 {
			return errors.Reason("-isolate must be specified").Err()
		}
	} else {
		if !filepath.IsAbs(c.Isolate) {
			c.Isolate = filepath.Clean(filepath.Join(cwd, c.Isolate))
		}
	}

	if c.Isolated == "" {
		if flags&RequireIsolatedFile != 0 {
			return errors.Reason("-isolated must be specified").Err()
		}
	} else {
		if !filepath.IsAbs(c.Isolated) {
			c.Isolated = filepath.Clean(filepath.Join(cwd, c.Isolated))
		}
	}
	return nil
}

func elideNestedPaths(deps []string, pathSep string) []string {
	// For |deps| having a pattern like below:
	// "ab/"
	// "ab/cd/"
	// "ab/foo.txt"
	//
	// We need to elide the nested paths under "ab/" to make HardlinkRecursively
	// work. Without this step, all files have already been hard linked when
	// processing "ab/", so "ab/cd/" would lead to an error.
	sort.Strings(deps)
	prefixDir := ""
	var result []string
	for _, dep := range deps {
		if len(prefixDir) > 0 && strings.HasPrefix(dep, prefixDir) {
			continue
		}
		// |dep| can be either an unseen directory, or an individual file
		result = append(result, dep)
		prefixDir = ""
		if strings.HasSuffix(dep, pathSep) {
			// |dep| is a directory
			prefixDir = dep
		}
	}
	return result
}

func recreateTree(outDir string, rootDir string, deps []string) error {
	if err := filesystem.MakeDirs(outDir); err != nil {
		return errors.Annotate(err, "failed to create directory: %s", outDir).Err()
	}
	deps = elideNestedPaths(deps, string(os.PathSeparator))
	createdDirs := make(map[string]struct{})
	for _, dep := range deps {
		dst := filepath.Join(outDir, dep[len(rootDir):])
		dstDir := filepath.Dir(dst)
		if _, ok := createdDirs[dstDir]; !ok {
			if err := filesystem.MakeDirs(dstDir); err != nil {
				return errors.Annotate(err, "failed to call MakeDirs(%s)", dstDir).Err()
			}
			createdDirs[dstDir] = struct{}{}
		}

		err := filesystem.HardlinkRecursively(dep, dst)
		if err != nil {
			return errors.Annotate(err, "failed to call HardlinkRecursively(%s, %s)", dep, dst).Err()
		}
	}
	return nil
}

// archiveLogger reports stats to stderr.
type archiveLogger struct {
	start time.Time
	quiet bool
}

// LogSummary logs (to eventlog and stderr) a high-level summary of archive operations(s).
func (al *archiveLogger) LogSummary(ctx context.Context, hits, misses int, bytesHit, bytesPushed units.Size) {
	if !al.quiet {
		duration := time.Since(al.start)
		fmt.Fprintf(os.Stderr, "Hits    : %5d (%s)\n", hits, bytesHit)
		fmt.Fprintf(os.Stderr, "Misses  : %5d (%s)\n", misses, bytesPushed)
		fmt.Fprintf(os.Stderr, "Duration: %s\n", duration.Round(time.Millisecond))
	}
}

// Print acts like fmt.Printf, but may prepend a prefix to format, depending on the value of al.quiet.
func (al *archiveLogger) Printf(format string, a ...interface{}) (n int, err error) {
	return al.Fprintf(os.Stdout, format, a...)
}

// Print acts like fmt.fprintf, but may prepend a prefix to format, depending on the value of al.quiet.
func (al *archiveLogger) Fprintf(w io.Writer, format string, a ...interface{}) (n int, err error) {
	prefix := "\n"
	if al.quiet {
		prefix = ""
	}
	args := make([]interface{}, 1+len(a))
	args[0] = prefix
	copy(args[1:], a)
	return fmt.Printf("%s"+format, args...)
}

func (al *archiveLogger) printSummary(summary tarring.IsolatedSummary) {
	al.Printf("%s\t%s\n", summary.Digest, summary.Name)
}

func buildCASInputSpec(opts *isolate.ArchiveOptions) (string, *command.InputSpec, error) {
	inputPaths, execRoot, err := isolate.ProcessIsolateForCAS(opts)
	if err != nil {
		return "", nil, err
	}

	inputSpec := &command.InputSpec{
		Inputs: inputPaths,
	}
	if opts.IgnoredPathFilterRe != "" {
		inputSpec.InputExclusions = []*command.InputExclusion{
			{
				Regex: opts.IgnoredPathFilterRe,
				Type:  command.UnspecifiedInputType,
			},
		}
	}

	return execRoot, inputSpec, nil
}

func (r *baseCommandRun) uploadToCAS(ctx context.Context, dumpJSON string, authOpts auth.Options, fl *cas.Flags, al *archiveLogger, opts ...*isolate.ArchiveOptions) ([]digest.Digest, error) {
	// To cancel |uploadEg| when there is error in |digestEg|.
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	cl, err := r.newCASClient(ctx, fl.Instance, authOpts, false)
	if err != nil {
		return nil, err
	}
	defer cl.Close()

	start := time.Now()
	fmCache := filemetadata.NewSingleFlightCache()

	rootDgs := make([]digest.Digest, len(opts))
	type entries struct {
		isolate string
		entries []*uploadinfo.Entry
	}
	entriesC := make(chan entries)

	digestEg, _ := errgroup.WithContext(ctx)

	// limit the number of concurrent hash calculations and I/O operations.
	ch := make(chan struct{}, runtime.NumCPU())
	logger := logging.Get(ctx)

	for i, o := range opts {
		i, o := i, o
		digestEg.Go(func() error {
			ch <- struct{}{}
			defer func() { <-ch }()

			execRoot, is, err := buildCASInputSpec(o)
			if err != nil {
				return errors.Annotate(err, "failed to call buildCASInputSpec").Err()
			}

			start := time.Now()
			rootDg, entrs, stats, err := cl.ComputeMerkleTree(execRoot, is, fmCache)
			if err != nil {
				return errors.Annotate(err, "failed to call ComputeMerkleTree for %s", o.Isolate).Err()
			}
			logger.Infof("ComputeMerkleTree returns %d entries with total size %d for %s, took %s",
				len(entrs), stats.TotalInputBytes, o.Isolate, time.Since(start))
			rootDgs[i] = rootDg
			entriesC <- entries{o.Isolate, entrs}

			return nil
		})
	}

	var entryCount int
	var entrySize int64
	var uploadEntryCount int64
	var uploadEntrySize int64
	var uploadedBytes int64

	uploadEg, uctx := errgroup.WithContext(ctx)
	uploadEg.Go(func() error {
		uploaded := make(map[digest.Digest]struct{})
		for entrs := range entriesC {
			entrs := entrs
			entryCount += len(entrs.entries)
			toUpload := make([]*uploadinfo.Entry, 0, len(entrs.entries))
			for _, e := range entrs.entries {
				entrySize += e.Digest.Size
				if _, ok := uploaded[e.Digest]; ok {
					continue
				}
				uploaded[e.Digest] = struct{}{}
				toUpload = append(toUpload, e)
			}

			uploadEg.Go(func() error {
				start := time.Now()
				uploaded, bytes, err := cl.UploadIfMissing(uctx, toUpload...)
				if err != nil {
					return errors.Annotate(err, "failed to upload: %s", entrs.isolate).Err()
				}

				logger.Infof("finished upload for %d entries (%d uploaded, %d bytes), took %s",
					len(toUpload), len(uploaded), bytes, time.Since(start))

				uploadSizeSum := int64(0)
				for _, d := range uploaded {
					uploadSizeSum += d.Size
				}
				atomic.AddInt64(&uploadEntryCount, int64(len(uploaded)))
				atomic.AddInt64(&uploadEntrySize, uploadSizeSum)
				atomic.AddInt64(&uploadedBytes, bytes)
				return nil
			})
		}
		return nil
	})

	if err := digestEg.Wait(); err != nil {
		close(entriesC)
		return nil, err
	}

	close(entriesC)
	if err := uploadEg.Wait(); err != nil {
		return nil, err
	}

	logger.Infof("finished upload for %d entries (%d uploaded, %d bytes), took %s",
		entryCount, uploadEntryCount, uploadedBytes, time.Since(start))

	if al != nil {
		al.LogSummary(ctx, entryCount-int(uploadEntryCount), int(uploadEntryCount), units.Size(entrySize-uploadEntrySize), units.Size(uploadEntrySize))
	}

	if dumpJSON == "" {
		return rootDgs, nil
	}

	m := make(map[string]string, len(opts))
	for i, o := range opts {
		m[filesystem.GetFilenameNoExt(o.Isolate)] = rootDgs[i].String()
	}
	f, err := os.Create(dumpJSON)
	if err != nil {
		return rootDgs, err
	}
	defer f.Close()
	return rootDgs, json.NewEncoder(f).Encode(m)
}
