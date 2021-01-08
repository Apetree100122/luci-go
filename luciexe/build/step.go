// Copyright 2020 The LUCI Authors.
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

package build

import (
	"context"
	"fmt"
	"io"
	"strings"
	"sync"

	"google.golang.org/protobuf/types/known/timestamppb"

	bbpb "go.chromium.org/luci/buildbucket/proto"
	"go.chromium.org/luci/buildbucket/protoutil"
	"go.chromium.org/luci/common/clock"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/common/logging/gologger"
	"go.chromium.org/luci/common/system/environ"
	"go.chromium.org/luci/logdog/api/logpb"
	"go.chromium.org/luci/logdog/client/butlerlib/streamclient"
	"go.chromium.org/luci/logdog/client/butlerlib/streamproto"
	ldTypes "go.chromium.org/luci/logdog/common/types"
	"go.chromium.org/luci/luciexe"
)

// Step represents the state of a single step.
//
// This is properly initialized by the StartStep and ScheduleStep functions.
type Step struct {
	ctx       context.Context
	ctxCloser func()
	state     *State

	// duplicated from stepPb.Name at construction time to avoid need for locks.
	// Read-only.
	name string

	stepPbMu sync.Mutex
	stepPb   *bbpb.Step

	logPrefix     string
	relLogPrefix  string
	logNames      nameTracker
	logClosers    map[string]func() error
	loggingStream io.Closer
}

var _ Loggable = (*Step)(nil)

// StartStep adds a new step to the build.
//
// The step will have a "STARTED" status with a StartTime.
//
// The returned context is updated so that calling StartStep/ScheduleStep on it
// will create sub-steps.
//
// If `name` contains `|` this function will panic, since this is a reserved
// character for delimiting hierarchy in steps.
//
// Duplicate step names will be disambiguated by appending " (N)" for the 2nd,
// 3rd, etc. duplicate.
//
// The returned context will have `name` embedded in it; Calling StartStep or
// ScheduleStep with this context will generate a sub-step.
//
// The returned context also has an updated `environ.FromCtx` containing
// a unique $LOGDOG_NAMESPACE value. If you launch a subprocess, you should use
// this environment to correctly namespace any logdog log streams your
// subprocess attempts to open. Using `go.chromium.org/luci/luciexe/build/exec`
// does this automatically.
//
// You MUST call Step.End. To automatically map errors and panics to their
// correct visual representation, End the Step like:
//
//    var err error
//    step, ctx := build.StartStep(ctx, "Step name")
//    defer func() { step.End(err) }()
//
//    err = opThatErrsOrPanics(ctx)
//
// NOTE: A panic will still crash the program as usual. This does NOT
// `recover()` the panic. Please use conventional Go error handling and control
// flow mechanisms.
func StartStep(ctx context.Context, name string) (*Step, context.Context) {
	ret, ctx := ScheduleStep(ctx, name)
	ret.Start()
	return ret, ctx
}

// ScheduleStep is like StartStep, except that it leaves the new step in the
// SCHEDULED status, and does not set a StartTime.
//
// The step will move to STARTED when calling any other methods on
// the Step, when creating a sub-Step, or if you explicitly call
// Step.Start().
func ScheduleStep(ctx context.Context, name string) (*Step, context.Context) {
	if strings.Contains(name, "|") {
		panic(errors.Reason("step name %q contains reserved character `|`", name).Err())
	}

	cstate := getState(ctx)
	ctx, ctxCloser := context.WithCancel(ctx)

	if cstate.step != nil {
		cstate.step.ensureStarted(nil)
	}

	ret := &Step{
		ctx:       ctx,
		ctxCloser: ctxCloser,
		state:     cstate.state,

		logClosers: map[string]func() error{},
	}
	ret.stepPb, ret.relLogPrefix, ret.logPrefix = cstate.state.registerStep(&bbpb.Step{
		Name:   cstate.stepNamePrefix() + name,
		Status: bbpb.Status_SCHEDULED,
	})
	ret.name = ret.stepPb.Name

	if ls := ret.logsink(); ls == nil {
		ctx = logging.SetField(ctx, "build.step", ret.stepPb.Name)
		logging.Infof(ctx, "set status: %s", ret.stepPb.Status)
	} else {
		ret.addLog("log", func(name string, relLdName ldTypes.StreamName) io.Closer {
			var err error
			var stream io.WriteCloser
			stream, err = ls.NewStream(ret.ctx, relLdName)
			if err != nil {
				panic(err)
			}

			// TODO(iannucci): figure out how to preserve log format from context?
			ctx = (&gologger.LoggerConfig{Out: stream}).Use(ctx)
			// we track this in ret.loggingStream so don't have addLog track it.
			ret.loggingStream = stream

			return nil
		})

		// Each step gets its own logdog namespace "step/X/u". Any subprocesses
		// running within this ctx SHOULD use environ.FromCtx to pick this up.
		env := environ.FromCtx(ctx)
		env.Set(luciexe.LogdogNamespaceEnv, ret.logPrefix+"/u")
		ctx = env.SetInCtx(ctx)
	}
	ret.ctx = ctx

	cstate.step = ret
	return ret, setState(ctx, cstate)
}

// End sets the step's final status, according to `err` (See ExtractStatus).
//
// End will also be able to set INFRA_FAILURE status and log additional
// information if the program is panic'ing.
//
// End'ing a Step will Cancel the context associated with this step (returned
// from StartStep or ScheduleStep).
//
// End must be invoked like:
//
//    var err error
//    step, ctx := build.StartStep(ctx, ...)  // or build.ScheduleStep
//    defer func() { step.End(err) }()
//
//    err = opThatErrsOrPanics()
//
// NOTE: A panic will still crash the program as usual. This does NOT
// `recover()` the panic. Please use conventional Go error handling and control
// flow mechanisms.
func (s *Step) End(err error) {
	var message string
	s.mutate(func() bool {
		s.stepPb.Status, message = computePanicStatus(err)
		s.stepPb.EndTime = timestamppb.New(clock.Now(s.ctx))
		if s.stepPb.StartTime == nil {
			// In case the user scheduled the step, but never Start'd it.
			s.stepPb.StartTime = s.stepPb.EndTime
		}

		for logName, closer := range s.logClosers {
			if err := closer(); err != nil {
				logging.Warningf(s.ctx, "error closing log %q: %s", logName, err)
			}
		}
		s.logClosers = nil

		return true
	})
	// stepPb is immutable after mutate ends, so we should be fine to access it
	// outside the locks.

	logStatus(s.ctx, s.stepPb.Status, message, s.stepPb.SummaryMarkdown)

	if s.loggingStream != nil {
		s.loggingStream.Close()
	}

	s.ctxCloser()
}

// addLog adds a new Log entry to this Step.
//
// `name` is the user-provided name for the log.
//
// `openStream` is a callback which takes
//   * `dedupedName` - the deduplicated version of `name`
//   * `relLdName` - The logdog stream name, relative to this process'
//     LOGDOG_NAMESPACE, suitable for use with s.state.logsink.
func (s *Step) addLog(name string, openStream func(dedupedName string, relLdName ldTypes.StreamName) io.Closer) {
	relLdName := ""
	s.mutate(func() bool {
		name = s.logNames.resolveName(name)
		relLdName = fmt.Sprintf("%s/log/%d", s.relLogPrefix, len(s.stepPb.Logs))
		s.stepPb.Logs = append(s.stepPb.Logs, &bbpb.Log{
			Name: name,
			Url:  relLdName,
		})
		if closer := openStream(name, ldTypes.StreamName(relLdName)); closer != nil {
			s.logClosers[relLdName] = closer.Close
		}
		return true
	})
}

// Log creates a new step-level line-oriented text log stream with the given name.
//
// The stream will close when the step is End'd.
func (s *Step) Log(name string, opts ...streamclient.Option) io.Writer {
	ls := s.logsink()
	var ret io.WriteCloser
	var openStream func(string, ldTypes.StreamName) io.Closer

	if ls == nil {
		openStream = func(name string, relLdName ldTypes.StreamName) io.Closer {
			if desc, _ := streamclient.RenderOptions(opts...); desc.Type != streamproto.StreamType(logpb.StreamType_TEXT) {
				// logpb.StreamType cast is necessary or .String() doesn't work.
				typ := logpb.StreamType(desc.Type)
				logging.Warningf(s.ctx, "dropping %s log %q", typ, name)
				ret = nopStream{}
			} else {
				ret = makeLoggingWriter(s.ctx, name)
			}
			return ret
		}
	} else {
		openStream = func(name string, relLdName ldTypes.StreamName) io.Closer {
			var err error
			ret, err = ls.NewStream(s.ctx, relLdName, opts...)
			if err != nil {
				panic(err)
			}
			return ret
		}
	}

	s.addLog(name, openStream)
	return ret
}

// LogDatagram creates a new step-level datagram log stream with the given name.
// Each call to WriteDatagram will produce a single datagram message in the
// stream.
//
// The stream will close when the step is End'd.
func (s *Step) LogDatagram(name string, opts ...streamclient.Option) streamclient.DatagramWriter {
	ls := s.logsink()
	var ret streamclient.DatagramStream
	var openStream func(string, ldTypes.StreamName) io.Closer
	if ls == nil {
		openStream = func(name string, relLdName ldTypes.StreamName) io.Closer {
			logging.Warningf(s.ctx, "dropping DATAGRAM log %q", name)
			ret = nopDatagramStream{}
			return ret
		}
	} else {
		openStream = func(name string, relLdName ldTypes.StreamName) io.Closer {
			var err error
			ret, err = ls.NewDatagramStream(s.ctx, relLdName, opts...)
			if err != nil {
				panic(err)
			}
			return ret
		}
	}

	s.addLog(name, openStream)
	return ret
}

func (s *Step) logsink() *streamclient.Client {
	if s.state == nil {
		return nil
	}
	return s.state.logsink
}

// mutate gives exclusive access to read+write stepPb
//
// Will panic if stepPb is in a terminal (ended) state.
func (s *Step) mutate(cb func() bool) {
	s.state.mutate(func() bool {
		s.stepPbMu.Lock()
		defer s.stepPbMu.Unlock()
		if protoutil.IsEnded(s.stepPb.Status) {
			panic(errors.Reason("cannot mutate ended step %q", s.stepPb.Name).Err())
		}
		return cb()
	})
}
