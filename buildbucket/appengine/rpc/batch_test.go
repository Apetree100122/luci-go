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

package rpc

import (
	"context"
	"math/rand"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/golang/protobuf/jsonpb"
	spb "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc/codes"
	grpcStatus "google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"go.chromium.org/luci/common/clock/testclock"
	"go.chromium.org/luci/common/data/rand/mathrand"
	"go.chromium.org/luci/common/testing/mock"
	"go.chromium.org/luci/gae/filter/txndefer"
	"go.chromium.org/luci/gae/impl/memory"
	"go.chromium.org/luci/gae/service/datastore"
	"go.chromium.org/luci/server/auth"
	"go.chromium.org/luci/server/auth/authtest"
	"go.chromium.org/luci/server/tq"

	"go.chromium.org/luci/buildbucket/appengine/model"
	pb "go.chromium.org/luci/buildbucket/proto"

	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"
)

func TestBatch(t *testing.T) {
	t.Parallel()

	Convey("Batch", t, func() {
		mockPyBBClient := pb.NewMockBuildsClient(gomock.NewController(t))
		srv := &Builds{testPyBuildsClient: mockPyBBClient}
		ctx, _ := tq.TestingContext(txndefer.FilterRDS(memory.Use(context.Background())), nil)
		ctx = mathrand.Set(ctx, rand.New(rand.NewSource(0)))
		datastore.GetTestable(ctx).AutoIndex(true)
		datastore.GetTestable(ctx).Consistent(true)

		ctx = auth.WithState(ctx, &authtest.FakeState{
			Identity: "user:caller@example.com",
		})
		So(datastore.Put(
			ctx,
			&model.Bucket{
				ID:     "bucket",
				Parent: model.ProjectKey(ctx, "project"),
				Proto: pb.Bucket{
					Acls: []*pb.Acl{
						{
							Identity: "user:caller@example.com",
							Role:     pb.Acl_WRITER,
						},
					},
				},
			},
			&model.Build{
				Proto: pb.Build{
					Id: 1,
					Builder: &pb.BuilderID{
						Project: "project",
						Bucket:  "bucket",
						Builder: "builder1",
					},
				},
			},
			&model.Build{
				Proto: pb.Build{
					Id: 2,
					Builder: &pb.BuilderID{
						Project: "project",
						Bucket:  "bucket",
						Builder: "builder2",
					},
				},
			}), ShouldBeNil)

		Convey("empty", func() {
			req := &pb.BatchRequest{
				Requests: []*pb.BatchRequest_Request{},
			}
			res, err := srv.Batch(ctx, req)
			So(err, ShouldBeNil)
			So(res, ShouldResembleProto, &pb.BatchResponse{})

			req = &pb.BatchRequest{
				Requests: []*pb.BatchRequest_Request{{}},
			}
			mockPyBBClient.EXPECT().Batch(ctx, gomock.Any()).Times(0)
			res, err = srv.Batch(ctx, req)
			So(err, ShouldNotBeNil)
			So(res, ShouldBeNil)
			So(err, ShouldErrLike, "request includes an unsupported type")
		})

		Convey("error", func() {
			req := &pb.BatchRequest{
				Requests: []*pb.BatchRequest_Request{
					{Request: &pb.BatchRequest_Request_GetBuild{
						GetBuild: &pb.GetBuildRequest{BuildNumber: 1},
					}},
				},
			}
			res, err := srv.Batch(ctx, req)
			expectedRes := &pb.BatchResponse{
				Responses: []*pb.BatchResponse_Response{
					{Response: &pb.BatchResponse_Response_Error{
						Error: &spb.Status{
							Code:    3,
							Message: "bad request: one of id or (builder and build_number) is required",
						},
					}},
				},
			}
			So(err, ShouldBeNil)
			So(res, ShouldResembleProto, expectedRes)
		})

		Convey("getBuild req", func() {
			req := &pb.BatchRequest{
				Requests: []*pb.BatchRequest_Request{
					{Request: &pb.BatchRequest_Request_GetBuild{
						GetBuild: &pb.GetBuildRequest{Id: 1},
					}},
				},
			}
			res, err := srv.Batch(ctx, req)
			expectedRes := &pb.BatchResponse{
				Responses: []*pb.BatchResponse_Response{
					{Response: &pb.BatchResponse_Response_GetBuild{
						GetBuild: &pb.Build{
							Id: 1,
							Builder: &pb.BuilderID{
								Project: "project",
								Bucket:  "bucket",
								Builder: "builder1",
							},
							Input: &pb.Build_Input{},
						},
					}},
				},
			}
			So(err, ShouldBeNil)
			So(res, ShouldResembleProto, expectedRes)
		})

		Convey("searchBuilds req", func() {
			req := &pb.BatchRequest{
				Requests: []*pb.BatchRequest_Request{
					{Request: &pb.BatchRequest_Request_SearchBuilds{
						SearchBuilds: &pb.SearchBuildsRequest{},
					}},
				},
			}
			res, err := srv.Batch(ctx, req)
			expectedRes := &pb.BatchResponse{
				Responses: []*pb.BatchResponse_Response{
					{Response: &pb.BatchResponse_Response_SearchBuilds{
						SearchBuilds: &pb.SearchBuildsResponse{
							Builds: []*pb.Build{
								{Id: 1,
									Builder: &pb.BuilderID{
										Project: "project",
										Bucket:  "bucket",
										Builder: "builder1",
									},
									Input: &pb.Build_Input{},
								},
								{Id: 2,
									Builder: &pb.BuilderID{
										Project: "project",
										Bucket:  "bucket",
										Builder: "builder2",
									},
									Input: &pb.Build_Input{},
								},
							},
						},
					}},
				},
			}
			So(err, ShouldBeNil)
			So(res, ShouldResembleProto, expectedRes)
		})

		Convey("get and search reqs", func() {
			req := &pb.BatchRequest{
				Requests: []*pb.BatchRequest_Request{
					{Request: &pb.BatchRequest_Request_GetBuild{
						GetBuild: &pb.GetBuildRequest{Id: 1},
					}},
					{Request: &pb.BatchRequest_Request_SearchBuilds{
						SearchBuilds: &pb.SearchBuildsRequest{},
					}},
					{Request: &pb.BatchRequest_Request_GetBuild{
						GetBuild: &pb.GetBuildRequest{Id: 2},
					}},
				},
			}
			mockPyBBClient.EXPECT().Batch(ctx, gomock.Any()).Times(0)
			res, err := srv.Batch(ctx, req)
			build1 := &pb.Build{
				Id: 1,
				Builder: &pb.BuilderID{
					Project: "project",
					Bucket:  "bucket",
					Builder: "builder1",
				},
				Input: &pb.Build_Input{},
			}
			build2 := &pb.Build{
				Id: 2,
				Builder: &pb.BuilderID{
					Project: "project",
					Bucket:  "bucket",
					Builder: "builder2",
				},
				Input: &pb.Build_Input{},
			}
			expectedRes := &pb.BatchResponse{
				Responses: []*pb.BatchResponse_Response{
					{Response: &pb.BatchResponse_Response_GetBuild{
						GetBuild: build1,
					}},
					{Response: &pb.BatchResponse_Response_SearchBuilds{
						SearchBuilds: &pb.SearchBuildsResponse{
							Builds: []*pb.Build{build1, build2},
						},
					}},
					{Response: &pb.BatchResponse_Response_GetBuild{
						GetBuild: build2,
					}},
				},
			}
			So(err, ShouldBeNil)
			So(res, ShouldResembleProto, expectedRes)
		})

		Convey("schedule req py", func() {
			ctx = WithTrafficSplit(ctx, 0)
			req := &pb.BatchRequest{}
			err := jsonpb.UnmarshalString(`{
				"requests": [
					{"scheduleBuild": {}}
				]
			}`, req)
			So(err, ShouldBeNil)
			mockRes := &pb.BatchResponse{
				Responses: []*pb.BatchResponse_Response{
					{Response: &pb.BatchResponse_Response_ScheduleBuild{
						ScheduleBuild: &pb.Build{Id: 1},
					}},
				},
			}
			mockPyBBClient.EXPECT().Batch(ctx, mock.EqProto(req)).Return(mockRes, nil)
			actualRes, err := srv.Batch(ctx, req)
			So(err, ShouldBeNil)
			So(actualRes, ShouldResembleProto, mockRes)
		})

		Convey("schedule req", func() {
			ctx = WithTrafficSplit(ctx, 100)
			req := &pb.BatchRequest{
				Requests: []*pb.BatchRequest_Request{
					{Request: &pb.BatchRequest_Request_ScheduleBuild{
						ScheduleBuild: &pb.ScheduleBuildRequest{},
					}},
				},
			}
			res, err := srv.Batch(ctx, req)
			expectedRes := &pb.BatchResponse{
				Responses: []*pb.BatchResponse_Response{
					{Response: &pb.BatchResponse_Response_Error{
						Error: &spb.Status{
							Code:    3,
							Message: "bad request: builder or template_build_id is required",
						},
					}},
				},
			}
			So(err, ShouldBeNil)
			So(res, ShouldResembleProto, expectedRes)
		})

		Convey("cancel req", func() {
			now := testclock.TestRecentTimeLocal
			ctx, _ = testclock.UseTime(ctx, now)
			req := &pb.BatchRequest{
				Requests: []*pb.BatchRequest_Request{
					{Request: &pb.BatchRequest_Request_CancelBuild{
						CancelBuild: &pb.CancelBuildRequest{
							Id:              1,
							SummaryMarkdown: "summary",
						},
					}},
				},
			}
			res, err := srv.Batch(ctx, req)
			expectedRes := &pb.BatchResponse{
				Responses: []*pb.BatchResponse_Response{
					{Response: &pb.BatchResponse_Response_CancelBuild{
						CancelBuild: &pb.Build{
							Id: 1,
							Builder: &pb.BuilderID{
								Project: "project",
								Bucket:  "bucket",
								Builder: "builder1",
							},
							EndTime: timestamppb.New(now),
							Input:   &pb.Build_Input{},
							Status:  pb.Status_CANCELED,
						},
					}},
				},
			}
			So(err, ShouldBeNil)
			So(res, ShouldResembleProto, expectedRes)
		})

		Convey("get, schedule, search and cancel in req", func() {
			req := &pb.BatchRequest{}
			err := jsonpb.UnmarshalString(`{
				"requests": [
					{"getBuild": {"id": "1"}},
					{"scheduleBuild": {}},
					{"searchBuilds": {}},
					{"cancelBuild": {}}
				]}`, req)
			So(err, ShouldBeNil)
			expectedPyReq := &pb.BatchRequest{}
			err = jsonpb.UnmarshalString(`{
				"requests": [
					{"scheduleBuild": {}}
				]}`, expectedPyReq)
			So(err, ShouldBeNil)
			mockRes := &pb.BatchResponse{
				Responses: []*pb.BatchResponse_Response{
					{Response: &pb.BatchResponse_Response_ScheduleBuild{
						ScheduleBuild: &pb.Build{Id: 1},
					}},
				},
			}
			mockPyBBClient.EXPECT().Batch(ctx, mock.EqProto(expectedPyReq)).Return(mockRes, nil)
			actualRes, err := srv.Batch(ctx, req)
			build1 := &pb.Build{
				Id: 1,
				Builder: &pb.BuilderID{
					Project: "project",
					Bucket:  "bucket",
					Builder: "builder1",
				},
				Input: &pb.Build_Input{},
			}
			build2 := &pb.Build{
				Id: 2,
				Builder: &pb.BuilderID{
					Project: "project",
					Bucket:  "bucket",
					Builder: "builder2",
				},
				Input: &pb.Build_Input{},
			}
			expectedRes := &pb.BatchResponse{
				Responses: []*pb.BatchResponse_Response{
					{Response: &pb.BatchResponse_Response_GetBuild{
						GetBuild: build1,
					}},
					mockRes.Responses[0],
					{Response: &pb.BatchResponse_Response_SearchBuilds{
						SearchBuilds: &pb.SearchBuildsResponse{
							Builds: []*pb.Build{build1, build2},
						},
					}},
					{Response: &pb.BatchResponse_Response_Error{
						Error: &spb.Status{
							Code:    3,
							Message: "bad request: id is required",
						},
					}},
				},
			}
			So(err, ShouldBeNil)
			So(actualRes, ShouldResembleProto, expectedRes)
		})

		Convey("py service error", func() {
			req := &pb.BatchRequest{}
			err := jsonpb.UnmarshalString(`{
				"requests": [
					{"scheduleBuild": {}}
				]
			}`, req)
			So(err, ShouldBeNil)
			mockPyBBClient.EXPECT().Batch(ctx, mock.EqProto(req)).Return(nil, grpcStatus.Error(codes.Unavailable, "unavailable"))
			actualRes, err := srv.Batch(ctx, req)
			So(actualRes, ShouldBeNil)
			So(err, ShouldErrLike, "rpc error: code = Unavailable desc = unavailable")
		})

		Convey("py timeout error", func() {
			req := &pb.BatchRequest{}
			err := jsonpb.UnmarshalString(`{
				"requests": [
					{"scheduleBuild": {}}
				]
			}`, req)
			So(err, ShouldBeNil)
			mockPyBBClient.EXPECT().Batch(ctx, mock.EqProto(req)).Return(nil, grpcStatus.Error(codes.DeadlineExceeded, "timeout"))
			actualRes, err := srv.Batch(ctx, req)
			So(actualRes, ShouldBeNil)
			So(err, ShouldErrLike, "rpc error: code = Internal desc = timeout")
		})

		Convey("transport error", func() {
			ctx := memory.Use(context.Background())
			ctx = context.WithValue(ctx, &testFakeTransportError, grpcStatus.Error(codes.Internal, "failed to get Py BB RPC transport"))
			srv := &Builds{}
			req := &pb.BatchRequest{}
			err := jsonpb.UnmarshalString(`{
				"requests": [
					{"scheduleBuild": {}}
				]
			}`, req)
			So(err, ShouldBeNil)
			actualRes, err := srv.Batch(ctx, req)
			So(actualRes, ShouldBeNil)
			So(err, ShouldErrLike, "code = Internal desc = failed to get Py BB RPC transport")
		})

		Convey("exceed max read reqs amount", func() {
			req := &pb.BatchRequest{}
			for i := 0; i < readReqsSizeLimit+1; i++ {
				req.Requests = append(req.Requests, &pb.BatchRequest_Request{Request: &pb.BatchRequest_Request_GetBuild{}})
			}
			_, err := srv.Batch(ctx, req)
			So(err, ShouldErrLike, "the maximum allowed read request count in Batch is 1000.")
		})

		Convey("exceed max write reqs amount", func() {
			req := &pb.BatchRequest{}
			for i := 0; i < writeReqsSizeLimit+1; i++ {
				req.Requests = append(req.Requests, &pb.BatchRequest_Request{Request: &pb.BatchRequest_Request_ScheduleBuild{}})
			}
			_, err := srv.Batch(ctx, req)
			So(err, ShouldErrLike, "the maximum allowed write request count in Batch is 200.")
		})
	})
}
