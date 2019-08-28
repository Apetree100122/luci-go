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

package butler

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"testing"
	"time"

	"go.chromium.org/luci/common/clock/testclock"
	"go.chromium.org/luci/common/proto/google"
	. "go.chromium.org/luci/common/testing/assertions"
	"go.chromium.org/luci/logdog/api/logpb"
	"go.chromium.org/luci/logdog/client/butler/output"
	"go.chromium.org/luci/logdog/client/butlerlib/streamproto"
	"go.chromium.org/luci/logdog/common/types"

	. "github.com/smartystreets/goconvey/convey"
)

type testOutput struct {
	sync.Mutex

	err      error
	maxSize  int
	streams  map[string][]*logpb.LogEntry
	terminal map[string]struct{}
	closed   bool
}

func (to *testOutput) SendBundle(b *logpb.ButlerLogBundle) error {
	if to.err != nil {
		return to.err
	}

	to.Lock()
	defer to.Unlock()

	if to.streams == nil {
		to.streams = map[string][]*logpb.LogEntry{}
		to.terminal = map[string]struct{}{}
	}
	for _, be := range b.Entries {
		name := string(be.Desc.Name)

		to.streams[name] = append(to.streams[name], be.Logs...)
		if be.TerminalIndex >= 0 {
			to.terminal[name] = struct{}{}
		}
	}
	return nil
}

func (to *testOutput) MaxSize() int {
	return to.maxSize
}

func (to *testOutput) Stats() output.Stats {
	return &output.StatsBase{}
}

func (to *testOutput) Record() *output.EntryRecord { return nil }

func (to *testOutput) Close() {
	if to.closed {
		panic("double close")
	}
	to.closed = true
}

func (to *testOutput) logs(name string) []*logpb.LogEntry {
	to.Lock()
	defer to.Unlock()

	return to.streams[name]
}

func (to *testOutput) isTerminal(name string) bool {
	to.Lock()
	defer to.Unlock()

	_, isTerminal := to.terminal[name]
	return isTerminal
}

func shouldHaveTextLogs(actual interface{}, expected ...interface{}) string {
	exp := make([]string, len(expected))
	for i, e := range expected {
		exp[i] = e.(string)
	}

	var lines []string
	var prev string
	for _, le := range actual.([]*logpb.LogEntry) {
		if le.GetText() == nil {
			return fmt.Sprintf("non-text entry: %T %#v", le, le)
		}

		for _, l := range le.GetText().Lines {
			prev += string(l.Value)
			if l.Delimiter != "" {
				lines = append(lines, prev)
				prev = ""
			}
		}
	}

	// Add any trailing (non-delimited) data.
	if prev != "" {
		lines = append(lines, prev)
	}

	return ShouldResemble(lines, exp)
}

type testStreamData struct {
	data []byte
	err  error
}

type testStream struct {
	inC     chan *testStreamData
	closedC chan struct{}

	desc *logpb.LogStreamDescriptor
}

func (ts *testStream) data(d []byte, err error) {
	ts.inC <- &testStreamData{
		data: d,
		err:  err,
	}
	if err == io.EOF {
		// If EOF is hit, continue reporting EOF.
		close(ts.inC)
	}
}

func (ts *testStream) err(err error) {
	ts.data(nil, err)
}

func (ts *testStream) isClosed() bool {
	select {
	case <-ts.closedC:
		return true
	default:
		return false
	}
}

func (ts *testStream) Close() error {
	if ts.isClosed() {
		panic("double close")
	}

	close(ts.closedC)
	return nil
}

func (ts *testStream) Read(b []byte) (int, error) {
	select {
	case <-ts.closedC:
		return 0, io.EOF

	case d, ok := <-ts.inC:
		// We have data on "inC", but we want closed status to trump this.
		if !ok || ts.isClosed() {
			return 0, io.EOF
		}
		return copy(b, d.data), d.err
	}
}

type testStreamServer struct {
	err     error
	onNext  func()
	streamC chan *testStream
}

func newTestStreamServer() *testStreamServer {
	return &testStreamServer{
		streamC: make(chan *testStream),
	}
}

func (tss *testStreamServer) Address() string { return "test" }

func (tss *testStreamServer) Listen() error { return tss.err }

func (tss *testStreamServer) Next() (io.ReadCloser, *logpb.LogStreamDescriptor) {
	if tss.onNext != nil {
		tss.onNext()
	}

	ts, ok := <-tss.streamC
	if !ok {
		return nil, nil
	}
	return ts, ts.desc
}

func (tss *testStreamServer) Close() {
	close(tss.streamC)
}

func (tss *testStreamServer) enqueue(ts *testStream) {
	tss.streamC <- ts
}

func TestConfig(t *testing.T) {
	t.Parallel()

	Convey(`A Config instance`, t, func() {
		to := testOutput{
			maxSize: 1024,
		}
		conf := Config{
			Output: &to,
		}

		Convey(`Will validate.`, func() {
			So(conf.Validate(), ShouldBeNil)
		})

		Convey(`Will not validate with a nil Output.`, func() {
			conf.Output = nil
			So(conf.Validate(), ShouldErrLike, "an Output must be supplied")
		})
	})
}

// mkb is shorthand for "make Butler". It calls "new" and panics if there is
// an error.
func mkb(c context.Context, config Config) *Butler {
	b, err := New(c, config)
	if err != nil {
		panic(err)
	}
	return b
}

func TestButler(t *testing.T) {
	t.Parallel()

	Convey(`A testing Butler instance`, t, func() {
		c, _ := testclock.UseTime(context.Background(), testclock.TestTimeUTC)
		to := testOutput{
			maxSize: 1024,
		}

		conf := Config{
			Output:        &to,
			OutputWorkers: 1,
			BufferLogs:    false,
		}

		Convey(`Will error if an invalid Config is passed.`, func() {
			conf.Output = nil
			So(conf.Validate(), ShouldNotBeNil)

			_, err := New(c, conf)
			So(err, ShouldNotBeNil)
		})

		Convey(`Will Run until Activated, then shut down.`, func() {
			conf.BufferLogs = true // (Coverage)

			b := mkb(c, conf)
			b.Activate()
			So(b.Wait(), ShouldBeNil)
		})

		Convey(`Will forward a Shutdown error.`, func() {
			conf.BufferLogs = true // (Coverage)

			b := mkb(c, conf)
			b.shutdown(errors.New("shutdown error"))
			So(b.Wait(), ShouldErrLike, "shutdown error")
		})

		Convey(`Will retain the first error and ignore duplicate shutdowns.`, func() {
			b := mkb(c, conf)
			b.shutdown(errors.New("first error"))
			b.shutdown(errors.New("second error"))
			So(b.Wait(), ShouldErrLike, "first error")
		})

		Convey(`Using a generic stream Properties`, func() {
			newTestStream := func(setup func(d *logpb.LogStreamDescriptor)) *testStream {
				desc := &logpb.LogStreamDescriptor{
					Name:        "test",
					StreamType:  logpb.StreamType_TEXT,
					ContentType: string(types.ContentTypeText),
				}
				if setup != nil {
					setup(desc)
				}

				return &testStream{
					inC:     make(chan *testStreamData, 16),
					closedC: make(chan struct{}),
					desc:    desc,
				}
			}

			Convey(`Will not add a stream with an invalid configuration.`, func() {
				// No content type.
				s := newTestStream(func(d *logpb.LogStreamDescriptor) {
					d.ContentType = ""
				})
				b := mkb(c, conf)
				So(b.AddStream(s, s.desc), ShouldNotBeNil)
			})

			Convey(`Will not add a stream with a duplicate stream name.`, func() {
				b := mkb(c, conf)

				s0 := newTestStream(nil)
				So(b.AddStream(s0, s0.desc), ShouldBeNil)

				s1 := newTestStream(nil)
				So(b.AddStream(s1, s1.desc), ShouldErrLike, "a stream has already been registered")
			})

			Convey(`Can apply global tags.`, func() {
				conf.GlobalTags = streamproto.TagMap{
					"foo": "bar",
					"baz": "qux",
				}
				desc := &logpb.LogStreamDescriptor{
					Name:        "stdout",
					ContentType: "test/data",
					Timestamp:   google.NewTimestamp(time.Date(2016, 1, 1, 0, 0, 0, 0, time.UTC)),
				}

				closeStreams := make(chan *testStreamData)

				// newTestStream returns a stream that hangs in Read() until
				// 'closeStreams' is closed. We do this to ensure Bundler doesn't
				// "drain" and unregister the stream before GetStreamDescs() calls
				// below have a chance to notice it exists.
				newTestStream := func() io.ReadCloser {
					return &testStream{
						closedC: make(chan struct{}),
						inC:     closeStreams,
					}
				}

				b := mkb(c, conf)
				defer func() {
					close(closeStreams)
					b.Activate()
					b.Wait()
				}()

				Convey(`Applies global tags, but allows the stream to override.`, func() {
					desc.Tags = map[string]string{
						"baz": "override",
					}

					So(b.AddStream(newTestStream(), desc), ShouldBeNil)
					So(b.bundler.GetStreamDescs(), ShouldResemble, map[string]*logpb.LogStreamDescriptor{
						"stdout": {
							Name:        "stdout",
							ContentType: "test/data",
							Timestamp:   desc.Timestamp,
							Tags: map[string]string{
								"foo": "bar",
								"baz": "override",
							},
						},
					})
				})

				Convey(`Will apply global tags if the stream has none (nil).`, func() {
					So(b.AddStream(newTestStream(), desc), ShouldBeNil)
					So(b.bundler.GetStreamDescs(), ShouldResemble, map[string]*logpb.LogStreamDescriptor{
						"stdout": {
							Name:        "stdout",
							ContentType: "test/data",
							Timestamp:   desc.Timestamp,
							Tags: map[string]string{
								"foo": "bar",
								"baz": "qux",
							},
						},
					})
				})
			})

			Convey(`Run with 256 streams, stream{0..256} will deplete and finish.`, func() {
				b := mkb(c, conf)
				streams := make([]*testStream, 256)
				for i := range streams {
					streams[i] = newTestStream(func(d *logpb.LogStreamDescriptor) {
						d.Name = fmt.Sprintf("stream%d", i)
					})
				}

				for _, s := range streams {
					So(b.AddStream(s, s.desc), ShouldBeNil)
					s.data([]byte("stream data 0!\n"), nil)
					s.data([]byte("stream data 1!\n"), nil)
				}

				// Add data to the streams after shutdown.
				for _, s := range streams {
					s.data([]byte("stream data 2!\n"), io.EOF)
				}

				b.Activate()
				So(b.Wait(), ShouldBeNil)

				for _, s := range streams {
					name := string(s.desc.Name)

					So(to.logs(name), shouldHaveTextLogs, "stream data 0!", "stream data 1!", "stream data 2!")
					So(to.isTerminal(name), ShouldBeTrue)
				}
			})

			Convey(`Shutdown with 256 in-progress streams, stream{0..256} will terminate if they emitted logs.`, func() {
				b := mkb(c, conf)
				streams := make([]*testStream, 256)
				for i := range streams {
					streams[i] = newTestStream(func(d *logpb.LogStreamDescriptor) {
						d.Name = fmt.Sprintf("stream%d", i)
					})
				}

				for _, s := range streams {
					So(b.AddStream(s, s.desc), ShouldBeNil)
					s.data([]byte("stream data!\n"), nil)
				}

				b.shutdown(errors.New("test shutdown"))
				So(b.Wait(), ShouldErrLike, "test shutdown")

				for _, s := range streams {
					if len(to.logs(s.desc.Name)) > 0 {
						So(to.isTerminal(string(s.desc.Name)), ShouldBeTrue)
					} else {
						So(to.isTerminal(string(s.desc.Name)), ShouldBeFalse)
					}
				}
			})

			Convey(`Using ten test stream servers`, func() {
				servers := make([]*testStreamServer, 10)
				for i := range servers {
					servers[i] = newTestStreamServer()
				}
				streams := []*testStream(nil)

				Convey(`Can register both before Run and will retain streams.`, func() {
					b := mkb(c, conf)
					for i, tss := range servers {
						b.AddStreamServer(tss)

						s := newTestStream(func(d *logpb.LogStreamDescriptor) {
							d.Name = fmt.Sprintf("stream%d", i)
						})
						streams = append(streams, s)
						s.data([]byte("test data"), io.EOF)
						tss.enqueue(s)
					}

					b.Activate()
					So(b.Wait(), ShouldBeNil)

					for _, s := range streams {
						So(to.logs(s.desc.Name), shouldHaveTextLogs, "test data")
						So(to.isTerminal(s.desc.Name), ShouldBeTrue)
					}
				})

				Convey(`Can register both during Run and will retain streams.`, func() {
					b := mkb(c, conf)
					for i, tss := range servers {
						b.AddStreamServer(tss)

						s := newTestStream(func(d *logpb.LogStreamDescriptor) {
							d.Name = fmt.Sprintf("stream%d", i)
						})
						streams = append(streams, s)
						s.data([]byte("test data"), io.EOF)
						tss.enqueue(s)
					}

					b.Activate()
					So(b.Wait(), ShouldBeNil)

					for _, s := range streams {
						So(to.logs(s.desc.Name), shouldHaveTextLogs, "test data")
						So(to.isTerminal(s.desc.Name), ShouldBeTrue)
					}
				})
			})

			Convey(`Will ignore stream registration errors, allowing re-registration.`, func() {
				tss := newTestStreamServer()

				// Generate an invalid stream for "tss" to register.
				sGood := newTestStream(nil)
				sGood.data([]byte("good test data"), io.EOF)

				sBad := newTestStream(func(d *logpb.LogStreamDescriptor) {
					d.ContentType = ""
				})
				sBad.data([]byte("bad test data"), io.EOF)

				b := mkb(c, conf)
				b.AddStreamServer(tss)
				tss.enqueue(sBad)
				tss.enqueue(sGood)
				b.Activate()
				So(b.Wait(), ShouldBeNil)

				So(sBad.isClosed(), ShouldBeTrue)
				So(sGood.isClosed(), ShouldBeTrue)
				So(to.logs("test"), shouldHaveTextLogs, "good test data")
				So(to.isTerminal("test"), ShouldBeTrue)
			})
		})

		Convey(`Will terminate if the stream server panics.`, func() {
			tss := newTestStreamServer()
			tss.onNext = func() {
				panic("test panic")
			}

			b := mkb(c, conf)
			b.AddStreamServer(tss)
			So(b.Wait(), ShouldErrLike, "test panic")
		})
	})
}
