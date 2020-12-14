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

package resultdb

import (
	"fmt"
	"testing"

	"google.golang.org/grpc/codes"

	"go.chromium.org/luci/server/auth"
	"go.chromium.org/luci/server/auth/authtest"

	"go.chromium.org/luci/resultdb/internal/pagination"
	uipb "go.chromium.org/luci/resultdb/internal/proto/ui"
	"go.chromium.org/luci/resultdb/internal/testutil"
	"go.chromium.org/luci/resultdb/internal/testutil/insert"
	"go.chromium.org/luci/resultdb/pbutil"
	pb "go.chromium.org/luci/resultdb/proto/v1"

	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"
)

func TestQueryTestVariants(t *testing.T) {
	Convey(`QueryTestVariants`, t, func() {
		ctx := auth.WithState(testutil.SpannerTestContext(t), &authtest.FakeState{
			Identity: "user:someone@example.com",
			IdentityPermissions: []authtest.RealmPermission{
				{Realm: "testproject:testrealm", Permission: permListTestResults},
			},
		})

		testutil.MustApply(ctx, insert.Invocation("inv0", pb.Invocation_ACTIVE, map[string]interface{}{"Realm": "testproject:testrealm"}))
		testutil.MustApply(ctx, insert.Invocation("inv1", pb.Invocation_ACTIVE, map[string]interface{}{"Realm": "testproject:testrealm"}))
		testutil.MustApply(ctx, testutil.CombineMutations(
			insert.TestResults("inv0", "T1", nil, pb.TestStatus_FAIL),
			insert.TestResults("inv0", "T2", nil, pb.TestStatus_FAIL),
			insert.TestResults("inv1", "T3", nil, pb.TestStatus_PASS),
			insert.TestResults("inv1", "T1", pbutil.Variant("a", "b"), pb.TestStatus_FAIL, pb.TestStatus_PASS),
			insert.TestExonerations("inv0", "T1", nil, 1),
		)...)

		srv := &uiServer{}

		Convey(`Permission denied`, func() {
			testutil.MustApply(ctx, insert.Invocation("invx", pb.Invocation_ACTIVE, map[string]interface{}{"Realm": "randomproject:testrealm"}))
			_, err := srv.QueryTestVariants(ctx, &uipb.QueryTestVariantsRequest{
				Invocations: []string{"invocations/invx"},
			})
			So(err, ShouldHaveAppStatus, codes.PermissionDenied)
		})

		Convey(`Valid`, func() {
			res, err := srv.QueryTestVariants(ctx, &uipb.QueryTestVariantsRequest{
				Invocations: []string{"invocations/inv0", "invocations/inv1"},
			})
			So(err, ShouldBeNil)
			So(res.NextPageToken, ShouldEqual, pagination.Token("EXPECTED", "", ""))

			So(len(res.TestVariants), ShouldEqual, 3)
			getTVStrings := func(tvs []*uipb.TestVariant) []string {
				tvStrings := make([]string, len(tvs))
				for i, tv := range tvs {
					tvStrings[i] = fmt.Sprintf("%d/%s/%s", int32(tv.Status), tv.TestId, tv.VariantHash)
				}
				return tvStrings
			}
			So(getTVStrings(res.TestVariants), ShouldResemble, []string{
				"1/T2/e3b0c44298fc1c14",
				"2/T1/c467ccce5a16dc72",
				"3/T1/e3b0c44298fc1c14",
			})
		})
	})
}
