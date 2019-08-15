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

package streamclient

import (
	"bytes"
	"io"
	"testing"

	"go.chromium.org/luci/common/clock/clockflag"
	"go.chromium.org/luci/common/clock/testclock"
	"go.chromium.org/luci/common/data/recordio"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/logdog/client/butlerlib/streamproto"
	"go.chromium.org/luci/logdog/common/types"

	. "github.com/smartystreets/goconvey/convey"
)

type testStreamWriteCloser struct {
	bytes.Buffer
	addr string

	err    error
	closed bool
}

func (ts *testStreamWriteCloser) Write(d []byte) (int, error) {
	if ts.err != nil {
		return 0, ts.err
	}
	return ts.Buffer.Write(d)
}

func (ts *testStreamWriteCloser) Close() error {
	ts.closed = true
	return nil
}

// A list of all streams opened with the 'test' protocol.
var testError error
var testStreams = []*testStreamWriteCloser{}

func init() {
	protocolRegistry["test"] = func(addr string, ns types.StreamName) (Client, error) {
		return &clientImpl{
			factory: func() (io.WriteCloser, error) {
				tswc := &testStreamWriteCloser{addr: addr, err: testError}
				testStreams = append(testStreams, tswc)
				return tswc, nil
			},
			ns: ns,
		}, nil
	}
}

func TestClient(t *testing.T) {
	Convey(`A client registry with a test protocol`, t, func() {
		// clear out test globals
		testError = nil
		testStreams = nil

		flags := streamproto.Flags{
			Name:      "test",
			Timestamp: clockflag.Time(testclock.TestTimeUTC),
		}

		Convey(`Will fail to instantiate a Client with an invalid protocol.`, func() {
			_, err := New("fake:foo", "")
			So(err, ShouldNotBeNil)
		})

		Convey(`Can instantiate a new client.`, func() {
			client, err := New("test:foo", "namespace")
			So(err, ShouldBeNil)
			So(client, ShouldHaveSameTypeAs, &clientImpl{})

			Convey(`That can instantiate new Streams.`, func() {
				stream, err := client.NewStream(flags)
				So(err, ShouldBeNil)
				So(stream, ShouldHaveSameTypeAs, &BaseStream{})

				si := stream.(*BaseStream)
				So(si.WriteCloser, ShouldHaveSameTypeAs, &testStreamWriteCloser{})
				So(si.P.Name, ShouldEqual, "namespace/test")

				tswc := si.WriteCloser.(*testStreamWriteCloser)

				Convey(`The stream should have the stream header written to it.`, func() {
					So(tswc.Next(len(streamproto.ProtocolFrameHeaderMagic)), ShouldResemble,
						streamproto.ProtocolFrameHeaderMagic)

					r := recordio.NewReader(tswc, -1)
					f, err := r.ReadFrameAll()
					So(err, ShouldBeNil)
					So(string(f), ShouldResemble, `{"name":"namespace/test","timestamp":"0001-02-03T04:05:06.000000007Z"}`)
				})
			})

			Convey(`If the stream fails to write the handshake, it will be closed.`, func() {
				testError = errors.New("test error")
				_, err := client.NewStream(flags)
				So(err, ShouldNotBeNil)

				So(testStreams, ShouldHaveLength, 1)
				So(testStreams[0].closed, ShouldBeTrue)
			})
		})

		Convey(`Can instantiate a new client without namespace.`, func() {
			client, err := New("test:foo", "")
			stream, err := client.NewStream(flags)
			So(err, ShouldBeNil)
			So(stream, ShouldHaveSameTypeAs, &BaseStream{})

			si := stream.(*BaseStream)
			So(si.P.Name, ShouldEqual, "test")
		})

	})
}
