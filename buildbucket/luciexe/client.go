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

// Package luciexe implements LUCI Executable protocol, documented in
// message Executable in
// https://chromium.googlesource.com/infra/luci/luci-go/+/master/buildbucket/proto/common.proto
package luciexe

import (
	"io/ioutil"
	"os"
	"time"

	"github.com/golang/protobuf/proto"

	"go.chromium.org/luci/buildbucket/protoutil"
	"go.chromium.org/luci/common/clock/clockflag"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/logdog/api/logpb"
	"go.chromium.org/luci/logdog/client/butlerlib/bootstrap"
	"go.chromium.org/luci/logdog/client/butlerlib/streamclient"
	"go.chromium.org/luci/logdog/client/butlerlib/streamproto"

	pb "go.chromium.org/luci/buildbucket/proto"
)

// BuildStreamName is the name of the build stream, relative to $LOGDOG_STREAM_PREFIX.
// For more details, see Executable message in
// https://chromium.googlesource.com/infra/luci/luci-go/+/master/buildbucket/proto/common.proto
const BuildStreamName = "build.proto"

// Client can be used by Go programs to implement LUCI Executable protocol.
//
// Example program that does not check errors:
//
//   package main
//
//   import
//     "go.chromium.org/luci/buildbucket/luciexe"
//     "go.chromium.org/luci/buildbucket/proto"
//   )
//
//   func main() int {
//      var client luciexe.Client
//      client.Init()
//      client.WriteBuild(&buildbucketpb.Build{
//        Steps: []*buildbucketpb.Step{
//          {
//            Name: "checkout",
//            Status: buildbucketpb.SUCCESS,
//            // start time, end time
//          },
//        },
//      })
//   }
//
// TODO(nodir): add support for sub-builds.
type Client struct {
	// Timestamp for the build message stream.
	// If zero, time.Now is used.
	BuildTimestamp time.Time

	// InitBuild is the initial state of the build read from stdin.
	InitBuild *pb.Build

	// Logdog environment.
	// Logdog.Client can be used to create new LogDog streams.
	Logdog *bootstrap.Bootstrap

	buildStream streamclient.Stream
}

// Init initializes the client. Populates c.InitBuild and c.Logdog.
func (c *Client) Init() error {
	stdin, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		return errors.Annotate(err, "failed to read stdin").Err()
	}

	c.InitBuild = &pb.Build{}
	if err := proto.Unmarshal(stdin, c.InitBuild); err != nil {
		return errors.Annotate(err, "failed to parse buildbucket.v2.Build from stdin").Err()
	}

	if c.Logdog, err = bootstrap.Get(); err != nil {
		return err
	}

	buildTimestamp := c.BuildTimestamp
	if buildTimestamp.IsZero() {
		buildTimestamp = time.Now()
	}

	c.buildStream, err = c.Logdog.Client.NewStream(streamproto.Flags{
		Name:        streamproto.StreamNameFlag(BuildStreamName),
		Type:        streamproto.StreamType(logpb.StreamType_DATAGRAM),
		ContentType: protoutil.BuildMediaType,
		Timestamp:   clockflag.Time(buildTimestamp),
	})
	return err
}

// WriteBuild sends a new version of Build message to the other side
// of the protocol, that is the host of the LUCI executable.
func (c *Client) WriteBuild(build *pb.Build) error {
	c.assertInitialized()
	buf, err := proto.Marshal(build)
	if err != nil {
		return err
	}
	return c.buildStream.WriteDatagram(buf)
}

// Close flushes all builds and closes down the client connection.
func (c *Client) Close() error {
	return c.buildStream.Close()
}

// assertInitialized panics if c is not initialized.
func (c *Client) assertInitialized() {
	if c.buildStream == nil {
		panic(errors.Reason("client is not initialized").Err())
	}
}
