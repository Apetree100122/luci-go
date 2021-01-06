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
	"bytes"
	"context"
	"io"
	"strings"
	"sync"

	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/logdog/client/butlerlib/streamclient"
)

// Loggable is the common interface for build entities which have log data
// associated with them.
//
// Implemented by State and Step.
//
// Logs all have a name which is an arbitrary bit of text to identify the log to
// human users (it will appear as the link on the build UI page). In particular
// it does NOT need to conform to the LogDog stream name alphabet.
//
// The log name "log" is reserved, and will automatically capture all logging
// outputs generated with the "go.chromium.org/luci/common/logging" API.
type Loggable interface {
	// Log creates a new log stream (by default, line-oriented text) with the
	// given name.
	//
	// To uphold the requirements of the Build proto message, duplicate log names
	// will be deduplicated with the same algorithm used for deduplicating step
	// names.
	//
	// To create a binary stream, pass streamclient.Binary() as one of the
	// options.
	//
	// The stream will close when the associated object (step or build) is End'd.
	Log(name string, opts ...streamclient.Option) io.Writer

	// Log creates a new datagram-oriented log stream with the given name.
	//
	// To uphold the requirements of the Build proto message, duplicate log names
	// will be deduplicated with the same algorithm used for deduplicating step
	// names.
	//
	// The stream will close when the associated object (step or build) is End'd.
	LogDatagram(name string, opts ...streamclient.Option) streamclient.DatagramWriter
}

// LogFromFile is a convenience function which allows you to record a file from
// the filesystem as a log on a Loggable object.
//
// Example:
//
//    LogFromFile(step, "thingy.log", "/path/to/thingy.log")
func LogFromFile(l Loggable, name string, filepath string, opts ...streamclient.Option) error {
	panic("implement me")
}

type loggingWriter struct {
	mu   sync.Mutex
	buf  *bytes.Buffer
	logf func(string)
}

var _ io.WriteCloser = (*loggingWriter)(nil)

func makeLoggingWriter(ctx context.Context, name string) io.WriteCloser {
	ctx = logging.SetField(ctx, "build.logname", name)
	targetLevel := logging.Info
	if strings.HasPrefix(name, "$") {
		targetLevel = logging.Debug
	}
	if !logging.IsLogging(ctx, targetLevel) {
		return nopStream{}
	}

	rawLogFn := logging.Get(ctx).LogCall
	return &loggingWriter{
		buf: &bytes.Buffer{},
		logf: func(line string) {
			rawLogFn(targetLevel, 0, "%s", []interface{}{line})
		},
	}
}

func (l *loggingWriter) Write(bs []byte) (n int, err error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if n, err = l.buf.Write(bs); err != nil {
		return
	}
	l.drainLines()
	return
}

func (l *loggingWriter) drainLines() {
	maybeReadLine := func() (string, bool) {
		i := bytes.IndexByte(l.buf.Bytes(), '\n')
		if i < 0 {
			return "", false
		}
		line := make([]byte, i+1)
		l.buf.Read(line) // cannot panic
		return string(line), true
	}

	for {
		line, ok := maybeReadLine()
		if !ok {
			return
		}
		l.logf(line)
	}
}

func (l *loggingWriter) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.drainLines()
	if l.buf.Len() > 0 {
		l.logf(l.buf.String())
	}
	return nil
}

type nopStream struct{}

var _ io.WriteCloser = nopStream{}

func (n nopStream) Write(dg []byte) (int, error) { return len(dg), nil }
func (n nopStream) Close() error                 { return nil }

type nopDatagramStream struct{}

var _ streamclient.DatagramStream = nopDatagramStream{}

func (n nopDatagramStream) WriteDatagram(dg []byte) error { return nil }
func (n nopDatagramStream) Close() error                  { return nil }
