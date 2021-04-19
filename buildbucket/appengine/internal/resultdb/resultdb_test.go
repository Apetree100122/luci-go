// Copyright 2021 The LUCI Authors.
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

package resultdb

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	"go.chromium.org/luci/gae/impl/memory"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	grpcStatus "google.golang.org/grpc/status"

	"go.chromium.org/luci/common/proto"
	luciconfig "go.chromium.org/luci/config"
	"go.chromium.org/luci/config/cfgclient"
	rdbPb "go.chromium.org/luci/resultdb/proto/v1"

	"go.chromium.org/luci/buildbucket/appengine/model"
	pb "go.chromium.org/luci/buildbucket/proto"

	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"
)

type fakeCfgClient struct {
	luciconfig.Interface
}

func (*fakeCfgClient) GetConfig(ctx context.Context, configSet luciconfig.Set, path string, metaOnly bool) (*luciconfig.Config, error) {
	return &luciconfig.Config{Content: `resultdb {hostname: "rdbHost"}`}, nil
}

func TestCreateInvocations(t *testing.T) {
	t.Parallel()

	Convey("create invocations", t, func() {
		mockClient := rdbPb.NewMockRecorderClient(gomock.NewController(t))
		ctx := context.WithValue(context.Background(), &mockRecorderClientKey, mockClient)
		ctx = cfgclient.Use(ctx, &fakeCfgClient{})
		ctx = memory.UseInfo(ctx, "cr-buildbucket-dev")

		bqExports := []*rdbPb.BigQueryExport{}
		historyOptions := &rdbPb.HistoryOptions{UseInvocationTimestamp: true}
		cfgs := map[string]map[string]*pb.Builder{
			"proj1/bucket": {"builder": &pb.Builder{
				Resultdb: &pb.Builder_ResultDB{
					Enable:         true,
					HistoryOptions: historyOptions,
					BqExports:      bqExports,
				},
			}},
			"proj2/bucket": {"builder": &pb.Builder{
				Resultdb: &pb.Builder_ResultDB{
					Enable:         true,
					HistoryOptions: historyOptions,
					BqExports:      bqExports,
				},
			}},
		}

		Convey("builds without number", func() {
			builds := []*model.Build{
				{
					ID: 1,
					Proto: pb.Build{
						Id: 1,
						Builder: &pb.BuilderID{
							Project: "proj1",
							Bucket:  "bucket",
							Builder: "builder",
						},
						Infra: &pb.BuildInfra{Resultdb: &pb.BuildInfra_ResultDB{}},
					},
					Experiments: []string{"+luci.use_realms"},
				},
				{
					ID: 2,
					Proto: pb.Build{
						Id: 2,
						Builder: &pb.BuilderID{
							Project: "proj1",
							Bucket:  "bucket",
							Builder: "builder",
						},
						Infra: &pb.BuildInfra{Resultdb: &pb.BuildInfra_ResultDB{}},
					},
					Experiments: []string{"+luci.use_realms"},
				},
			}

			mockClient.EXPECT().CreateInvocation(gomock.Any(), proto.MatcherEqual(
				&rdbPb.CreateInvocationRequest{
					InvocationId: "build-1",
					Invocation: &rdbPb.Invocation{
						BigqueryExports:  bqExports,
						ProducerResource: "//cr-buildbucket-dev.appspot.com/builds/1",
						HistoryOptions:   historyOptions,
						Realm:            "proj1:bucket",
					},
					RequestId: "build-1",
				}), gomock.Any()).DoAndReturn(func(ctx context.Context, in *rdbPb.CreateInvocationRequest, opt grpc.CallOption) (*rdbPb.Invocation, error) {
				h, _ := opt.(grpc.HeaderCallOption)
				h.HeaderAddr.Set("update-token", "token for build-1")
				return &rdbPb.Invocation{}, nil
			})
			mockClient.EXPECT().CreateInvocation(gomock.Any(), proto.MatcherEqual(
				&rdbPb.CreateInvocationRequest{
					InvocationId: "build-2",
					Invocation: &rdbPb.Invocation{
						BigqueryExports:  bqExports,
						ProducerResource: "//cr-buildbucket-dev.appspot.com/builds/2",
						HistoryOptions:   historyOptions,
						Realm:            "proj1:bucket",
					},
					RequestId: "build-2",
				}), gomock.Any()).DoAndReturn(func(ctx context.Context, in *rdbPb.CreateInvocationRequest, opt grpc.CallOption) (*rdbPb.Invocation, error) {
				h, _ := opt.(grpc.HeaderCallOption)
				h.HeaderAddr.Set("update-token", "token for build-2")
				return &rdbPb.Invocation{}, nil
			})

			err := CreateInvocations(ctx, builds, cfgs, "host")
			So(err, ShouldBeNil)
			So(builds[0].ResultDBUpdateToken, ShouldEqual, "token for build-1")
			So(builds[0].Proto.Infra.Resultdb.Invocation, ShouldEqual, "invocations/build-1")
			So(builds[1].ResultDBUpdateToken, ShouldEqual, "token for build-2")
			So(builds[1].Proto.Infra.Resultdb.Invocation, ShouldEqual, "invocations/build-2")
		})

		Convey("build with number", func() {
			builds := []*model.Build{
				{
					ID: 1,
					Proto: pb.Build{
						Id:     1,
						Number: 123,
						Builder: &pb.BuilderID{
							Project: "proj1",
							Bucket:  "bucket",
							Builder: "builder",
						},
						Infra: &pb.BuildInfra{Resultdb: &pb.BuildInfra_ResultDB{}},
					},
					Experiments: []string{"+luci.use_realms"},
				},
			}
			bqExports := []*rdbPb.BigQueryExport{}
			historyOptions := &rdbPb.HistoryOptions{UseInvocationTimestamp: true}

			sha256Bldr := sha256.Sum256([]byte("proj1/bucket/builder"))
			mockClient.EXPECT().CreateInvocation(gomock.Any(), proto.MatcherEqual(
				&rdbPb.CreateInvocationRequest{
					InvocationId: "build-1",
					Invocation: &rdbPb.Invocation{
						BigqueryExports:  bqExports,
						ProducerResource: "//cr-buildbucket-dev.appspot.com/builds/1",
						HistoryOptions:   historyOptions,
						Realm:            "proj1:bucket",
					},
					RequestId: "build-1",
				}), gomock.Any()).DoAndReturn(func(ctx context.Context, in *rdbPb.CreateInvocationRequest, opt grpc.CallOption) (*rdbPb.Invocation, error) {
				h, _ := opt.(grpc.HeaderCallOption)
				h.HeaderAddr.Set("update-token", "token for build id 1")
				return &rdbPb.Invocation{}, nil
			})
			mockClient.EXPECT().CreateInvocation(gomock.Any(), proto.MatcherEqual(
				&rdbPb.CreateInvocationRequest{
					InvocationId: fmt.Sprintf("build-%s-123", hex.EncodeToString(sha256Bldr[:])),
					Invocation: &rdbPb.Invocation{
						IncludedInvocations: []string{"invocations/build-1"},
						ProducerResource:    "//cr-buildbucket-dev.appspot.com/builds/1",
						State:               rdbPb.Invocation_FINALIZING,
						Realm:               "proj1:bucket",
					},
					RequestId: "build-1-123",
				})).Return(&rdbPb.Invocation{}, nil)

			err := CreateInvocations(ctx, builds, cfgs, "host")
			So(err, ShouldBeNil)
			So(len(builds), ShouldEqual, 1)
			So(builds[0].ResultDBUpdateToken, ShouldEqual, "token for build id 1")
			So(builds[0].Proto.Infra.Resultdb.Invocation, ShouldEqual, "invocations/build-1")
		})

		Convey("already exists error", func() {
			builds := []*model.Build{
				{
					ID: 1,
					Proto: pb.Build{
						Id:     1,
						Number: 123,
						Builder: &pb.BuilderID{
							Project: "proj1",
							Bucket:  "bucket",
							Builder: "builder",
						},
						Infra: &pb.BuildInfra{Resultdb: &pb.BuildInfra_ResultDB{}},
					},
					Experiments: []string{"+luci.use_realms"},
				},
			}
			bqExports := []*rdbPb.BigQueryExport{}
			historyOptions := &rdbPb.HistoryOptions{UseInvocationTimestamp: true}

			sha256Bldr := sha256.Sum256([]byte("proj1/bucket/builder"))
			mockClient.EXPECT().CreateInvocation(gomock.Any(), proto.MatcherEqual(
				&rdbPb.CreateInvocationRequest{
					InvocationId: "build-1",
					Invocation: &rdbPb.Invocation{
						BigqueryExports:  bqExports,
						ProducerResource: "//cr-buildbucket-dev.appspot.com/builds/1",
						HistoryOptions:   historyOptions,
						Realm:            "proj1:bucket",
					},
					RequestId: "build-1",
				}), gomock.Any()).Return(nil, grpcStatus.Error(codes.AlreadyExists, "already exists"))
			mockClient.EXPECT().CreateInvocation(gomock.Any(), proto.MatcherEqual(
				&rdbPb.CreateInvocationRequest{
					InvocationId: fmt.Sprintf("build-%s-123", hex.EncodeToString(sha256Bldr[:])),
					Invocation: &rdbPb.Invocation{
						IncludedInvocations: []string{"invocations/build-1"},
						ProducerResource:    "//cr-buildbucket-dev.appspot.com/builds/1",
						State:               rdbPb.Invocation_FINALIZING,
						Realm:               "proj1:bucket",
					},
					RequestId: "build-1-123",
				}), gomock.Any()).DoAndReturn(func(ctx context.Context, in *rdbPb.CreateInvocationRequest, opt grpc.CallOption) (*rdbPb.Invocation, error) {
				h, _ := opt.(grpc.HeaderCallOption)
				h.HeaderAddr.Set("update-token", "token for build number 123")
				return &rdbPb.Invocation{}, nil
			})

			err := CreateInvocations(ctx, builds, cfgs, "host")
			So(err, ShouldErrLike, "failed to create the invocation for build id: 1: rpc error: code = AlreadyExists desc = already exists")
		})

		Convey("resultDB throws err", func() {
			builds := []*model.Build{
				{
					ID: 1,
					Proto: pb.Build{
						Id: 1,
						Builder: &pb.BuilderID{
							Project: "proj1",
							Bucket:  "bucket",
							Builder: "builder",
						},
						Infra: &pb.BuildInfra{Resultdb: &pb.BuildInfra_ResultDB{}},
					},
					Experiments: []string{"+luci.use_realms"},
				},
			}

			mockClient.EXPECT().CreateInvocation(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, grpcStatus.Error(codes.DeadlineExceeded, "timeout"))

			err := CreateInvocations(ctx, builds, cfgs, "host")
			So(err, ShouldErrLike, "failed to create the invocation for build id: 1: rpc error: code = DeadlineExceeded desc = timeout")
		})

		Convey("resultDB not enabled", func() {
			builds := []*model.Build{
				{
					ID: 1,
					Proto: pb.Build{
						Id: 1,
						Builder: &pb.BuilderID{
							Project: "proj1",
							Bucket:  "bucket",
							Builder: "builder",
						},
						Infra: &pb.BuildInfra{Resultdb: &pb.BuildInfra_ResultDB{}},
					},
				},
			}
			cfgs = map[string]map[string]*pb.Builder{
				"proj1/bucket": {"builder": &pb.Builder{
					Resultdb: &pb.Builder_ResultDB{
						Enable: false,
					},
				}},
			}

			err := CreateInvocations(ctx, builds, cfgs, "host")
			So(err, ShouldBeNil)
			So(builds[0].Proto.Infra.Resultdb.Invocation, ShouldEqual, "")
		})
	})
}
