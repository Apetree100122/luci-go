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

package recorder

import (
	"context"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/golang/protobuf/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"

	"go.chromium.org/luci/common/clock"
	"go.chromium.org/luci/common/clock/testclock"
	"go.chromium.org/luci/common/testing/prpctest"
	"go.chromium.org/luci/grpc/appstatus"
	"go.chromium.org/luci/grpc/prpc"
	"go.chromium.org/luci/server/auth/authtest"

	"go.chromium.org/luci/resultdb/internal/span"
	"go.chromium.org/luci/resultdb/internal/testutil"
	"go.chromium.org/luci/resultdb/pbutil"
	pb "go.chromium.org/luci/resultdb/proto/rpc/v1"

	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"
)

func TestValidateInvocationDeadline(t *testing.T) {
	Convey(`ValidateInvocationDeadline`, t, func() {
		now := testclock.TestRecentTimeUTC

		Convey(`deadline in the past`, func() {
			deadline := pbutil.MustTimestampProto(now.Add(-time.Hour))
			err := validateInvocationDeadline(deadline, now)
			So(err, ShouldErrLike, `must be at least 10 seconds in the future`)
		})

		Convey(`deadline 5s in the future`, func() {
			deadline := pbutil.MustTimestampProto(now.Add(5 * time.Second))
			err := validateInvocationDeadline(deadline, now)
			So(err, ShouldErrLike, `must be at least 10 seconds in the future`)
		})

		Convey(`deadline in the future`, func() {
			deadline := pbutil.MustTimestampProto(now.Add(1e3 * time.Hour))
			err := validateInvocationDeadline(deadline, now)
			So(err, ShouldErrLike, `must be before 48h in the future`)
		})
	})
}

func TestValidateCreateInvocationRequest(t *testing.T) {
	t.Parallel()
	now := testclock.TestRecentTimeUTC
	Convey(`TestValidateCreateInvocationRequest`, t, func() {
		Convey(`empty`, func() {
			err := validateCreateInvocationRequest(&pb.CreateInvocationRequest{}, now, false)
			So(err, ShouldErrLike, `invocation_id: unspecified`)
		})

		Convey(`invalid id`, func() {
			err := validateCreateInvocationRequest(&pb.CreateInvocationRequest{
				InvocationId: "1",
			}, now, false)
			So(err, ShouldErrLike, `invocation_id: does not match`)
		})

		Convey(`reserved prefix`, func() {
			err := validateCreateInvocationRequest(&pb.CreateInvocationRequest{
				InvocationId: "build:8765432100",
			}, now, false)
			So(err, ShouldErrLike, `only invocations created by trusted systems may have id not starting with "u:"`)
		})

		Convey(`reserved prefix, allowed`, func() {
			err := validateCreateInvocationRequest(&pb.CreateInvocationRequest{
				InvocationId: "build:8765432100",
			}, now, true)
			So(err, ShouldBeNil)
		})

		Convey(`invalid request id`, func() {
			err := validateCreateInvocationRequest(&pb.CreateInvocationRequest{
				InvocationId: "u:a",
				RequestId:    "😃",
			}, now, false)
			So(err, ShouldErrLike, "request_id: does not match")
		})

		Convey(`invalid tags`, func() {
			err := validateCreateInvocationRequest(&pb.CreateInvocationRequest{
				InvocationId: "u:abc",
				Invocation: &pb.Invocation{
					Tags: pbutil.StringPairs("1", "a"),
				},
			}, now, false)
			So(err, ShouldErrLike, `invocation.tags: "1":"a": key: does not match`)
		})

		Convey(`invalid deadline`, func() {
			deadline := pbutil.MustTimestampProto(now.Add(-time.Hour))
			err := validateCreateInvocationRequest(&pb.CreateInvocationRequest{
				InvocationId: "u:abc",
				Invocation: &pb.Invocation{
					Deadline: deadline,
				},
			}, now, false)
			So(err, ShouldErrLike, `invocation: deadline: must be at least 10 seconds in the future`)
		})

		Convey(`invalid bigqueryExports`, func() {
			deadline := pbutil.MustTimestampProto(now.Add(time.Hour))
			err := validateCreateInvocationRequest(&pb.CreateInvocationRequest{
				InvocationId: "u:abc",
				Invocation: &pb.Invocation{
					Deadline: deadline,
					Tags:     pbutil.StringPairs("a", "b", "a", "c", "d", "e"),
					BigqueryExports: []*pb.BigQueryExport{
						{
							Project: "project",
						},
					},
				},
			}, now, false)
			So(err, ShouldErrLike, `bigquery_export[0]: dataset: unspecified`)
		})

		Convey(`producer_resource disallowed`, func() {
			err := validateCreateInvocationRequest(&pb.CreateInvocationRequest{
				InvocationId: "u:0",
				Invocation: &pb.Invocation{
					ProducerResource: "//builds.example.com/builds/1",
				},
			}, now, false)
			So(err, ShouldErrLike, `invocation: producer_resource: only trusted systems are allowed`)
		})

		Convey(`producer_resource allowed`, func() {
			err := validateCreateInvocationRequest(&pb.CreateInvocationRequest{
				InvocationId: "u:0",
				Invocation: &pb.Invocation{
					ProducerResource: "//builds.example.com/builds/1",
				},
			}, now, true)
			So(err, ShouldBeNil)
		})

		Convey(`valid`, func() {
			deadline := pbutil.MustTimestampProto(now.Add(time.Hour))
			err := validateCreateInvocationRequest(&pb.CreateInvocationRequest{
				InvocationId: "u:abc",
				Invocation: &pb.Invocation{
					Deadline: deadline,
					Tags:     pbutil.StringPairs("a", "b", "a", "c", "d", "e"),
				},
			}, now, false)
			So(err, ShouldBeNil)
		})
	})
}

func TestCreateInvocation(t *testing.T) {
	Convey(`TestCreateInvocation`, t, func() {
		ctx := testutil.SpannerTestContext(t)
		// Configure mock authentication to allow creation of custom invocation ids.
		ctx = authtest.MockAuthConfig(ctx)
		db := authtest.FakeDB{
			"anonymous:anonymous": []string{trustedInvocationCreators},
		}
		ctx = db.Use(ctx)

		start := clock.Now(ctx).UTC()

		// Setup a full HTTP server in order to retrieve response headers.
		server := &prpctest.Server{}
		server.UnaryServerInterceptor = func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
			res, err := handler(ctx, req)
			err = appstatus.GRPCifyAndLog(ctx, err)
			return res, err
		}
		pb.RegisterRecorderServer(server, newTestRecorderServer())
		server.Start(ctx)
		defer server.Close()
		client, err := server.NewClient()
		So(err, ShouldBeNil)
		recorder := pb.NewRecorderPRPCClient(client)

		Convey(`empty request`, func() {
			_, err := recorder.CreateInvocation(ctx, &pb.CreateInvocationRequest{})
			So(err, ShouldHaveGRPCStatus, codes.InvalidArgument, `bad request: invocation_id: unspecified`)
		})

		req := &pb.CreateInvocationRequest{
			InvocationId: "u:inv",
			Invocation:   &pb.Invocation{},
		}

		Convey(`already exists`, func() {
			_, err := span.Client(ctx).Apply(ctx, []*spanner.Mutation{
				testutil.InsertInvocation("u:inv", 1, nil),
			})
			So(err, ShouldBeNil)

			_, err = recorder.CreateInvocation(ctx, req)
			So(err, ShouldHaveGRPCStatus, codes.AlreadyExists)
		})

		Convey(`unsorted tags`, func() {
			req.Invocation.Tags = pbutil.StringPairs("b", "2", "a", "1")
			inv, err := recorder.CreateInvocation(ctx, req)
			So(err, ShouldBeNil)
			So(inv.Tags, ShouldResemble, pbutil.StringPairs("a", "1", "b", "2"))
		})

		Convey(`no invocation in request`, func() {
			inv, err := recorder.CreateInvocation(ctx, &pb.CreateInvocationRequest{InvocationId: "u:inv"})
			So(err, ShouldBeNil)
			So(inv.Name, ShouldEqual, "invocations/u:inv")
		})

		Convey(`idempotent`, func() {
			req := &pb.CreateInvocationRequest{
				InvocationId: "u:inv",
				Invocation:   &pb.Invocation{},
				RequestId:    "request id",
			}
			res, err := recorder.CreateInvocation(ctx, req)
			So(err, ShouldBeNil)

			res2, err := recorder.CreateInvocation(ctx, req)
			So(err, ShouldBeNil)
			So(res2, ShouldResembleProto, res)
		})

		Convey(`end to end`, func() {
			deadline := pbutil.MustTimestampProto(start.Add(time.Hour))
			headers := &metadata.MD{}
			bqExport := &pb.BigQueryExport{
				Project:     "project",
				Dataset:     "dataset",
				Table:       "table",
				TestResults: &pb.BigQueryExport_TestResults{},
			}
			req := &pb.CreateInvocationRequest{
				InvocationId: "u:inv",
				Invocation: &pb.Invocation{
					Deadline: deadline,
					Tags:     pbutil.StringPairs("a", "1", "b", "2"),
					BigqueryExports: []*pb.BigQueryExport{
						bqExport,
					},
					ProducerResource: "//builds.example.com/builds/1",
				},
			}
			inv, err := recorder.CreateInvocation(ctx, req, prpc.Header(headers))
			So(err, ShouldBeNil)

			expected := proto.Clone(req.Invocation).(*pb.Invocation)
			proto.Merge(expected, &pb.Invocation{
				Name:      "invocations/u:inv",
				State:     pb.Invocation_ACTIVE,
				CreatedBy: "anonymous:anonymous",

				// we use Spanner commit time, so skip the check
				CreateTime: inv.CreateTime,
			})
			So(inv, ShouldResembleProto, expected)

			So(headers.Get(UpdateTokenMetadataKey), ShouldHaveLength, 1)

			txn := span.Client(ctx).ReadOnlyTransaction()
			defer txn.Close()

			inv, err = span.ReadInvocationFull(ctx, txn, "u:inv")
			So(err, ShouldBeNil)
			So(inv, ShouldResembleProto, expected)

			// Check fields not present in the proto.
			var invExpirationTime, expectedResultsExpirationTime time.Time
			err = span.ReadInvocation(ctx, txn, "u:inv", map[string]interface{}{
				"InvocationExpirationTime":          &invExpirationTime,
				"ExpectedTestResultsExpirationTime": &expectedResultsExpirationTime,
			})
			So(err, ShouldBeNil)
			So(expectedResultsExpirationTime, ShouldHappenWithin, time.Second, start.Add(expectedResultExpiration))
			So(invExpirationTime, ShouldHappenWithin, time.Second, start.Add(invocationExpirationDuration))
		})
	})
}
