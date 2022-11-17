// Copyright 2022 The LUCI Authors.
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
	"testing"

	"github.com/golang/mock/gomock"
	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"
	rdbpb "go.chromium.org/luci/resultdb/proto/v1"
)

func TestResultDB(t *testing.T) {
	t.Parallel()
	Convey(`resultdb`, t, func() {
		ctl := gomock.NewController(t)
		defer ctl.Finish()
		mc := NewMockedClient(context.Background(), ctl)
		rc, err := NewClient(mc.Ctx, "rdbhost", "project")
		So(err, ShouldBeNil)

		inv := "invocations/build-87654321"

		Convey(`GetInvocation`, func() {
			realm := "realm"
			req := &rdbpb.GetInvocationRequest{
				Name: inv,
			}
			res := &rdbpb.Invocation{
				Name:  inv,
				Realm: realm,
			}
			mc.GetInvocation(req, res)

			invProto, err := rc.GetInvocation(mc.Ctx, inv)
			So(err, ShouldBeNil)
			So(invProto, ShouldResembleProto, res)
		})

		Convey(`BatchGetTestVariants`, func() {
			req := &rdbpb.BatchGetTestVariantsRequest{
				Invocation: inv,
				TestVariants: []*rdbpb.BatchGetTestVariantsRequest_TestVariantIdentifier{
					{
						TestId:      "ninja://test1",
						VariantHash: "hash1",
					},
					{
						TestId:      "ninja://test2",
						VariantHash: "hash2",
					},
				},
			}

			res := &rdbpb.BatchGetTestVariantsResponse{
				TestVariants: []*rdbpb.TestVariant{
					{
						TestId:      "ninja://test1",
						VariantHash: "hash1",
						Status:      rdbpb.TestVariantStatus_UNEXPECTED,
					},
					{
						TestId:      "ninja://test2",
						VariantHash: "hash2",
						Status:      rdbpb.TestVariantStatus_FLAKY,
					},
				},
			}
			mc.BatchGetTestVariants(req, res)
			tvs, err := rc.BatchGetTestVariants(mc.Ctx, req)
			So(err, ShouldBeNil)
			So(len(tvs), ShouldEqual, 2)
		})
	})
}
