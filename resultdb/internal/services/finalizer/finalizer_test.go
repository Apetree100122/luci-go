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

package finalizer

import (
	"testing"
	"time"

	"google.golang.org/protobuf/proto"

	"go.chromium.org/luci/server/tq"

	"go.chromium.org/luci/resultdb/internal/invocations"
	"go.chromium.org/luci/resultdb/internal/tasks/taskspb"
	"go.chromium.org/luci/resultdb/internal/testutil"
	"go.chromium.org/luci/resultdb/internal/testutil/insert"
	pb "go.chromium.org/luci/resultdb/proto/v1"

	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"
)

func TestShouldFinalize(t *testing.T) {
	Convey(`ShouldFinalize`, t, func() {
		ctx := testutil.SpannerTestContext(t)

		assertReady := func(invID invocations.ID, expected bool) {
			should, err := readyToFinalize(ctx, invID)
			So(err, ShouldBeNil)
			So(should, ShouldEqual, expected)
		}

		Convey(`Includes two ACTIVE`, func() {
			testutil.MustApply(ctx, testutil.CombineMutations(
				insert.InvocationWithInclusions("a", pb.Invocation_FINALIZING, nil, "b", "c"),
				insert.InvocationWithInclusions("b", pb.Invocation_ACTIVE, nil),
				insert.InvocationWithInclusions("c", pb.Invocation_ACTIVE, nil),
			)...)

			assertReady("a", false)
		})

		Convey(`Includes ACTIVE and FINALIZED`, func() {
			testutil.MustApply(ctx, testutil.CombineMutations(
				insert.InvocationWithInclusions("a", pb.Invocation_FINALIZING, nil, "b", "c"),
				insert.InvocationWithInclusions("b", pb.Invocation_ACTIVE, nil),
				insert.InvocationWithInclusions("c", pb.Invocation_FINALIZED, nil),
			)...)

			assertReady("a", false)
		})

		Convey(`INCLUDES ACTIVE and FINALIZING`, func() {
			testutil.MustApply(ctx, testutil.CombineMutations(
				insert.InvocationWithInclusions("a", pb.Invocation_FINALIZING, nil, "b", "c"),
				insert.InvocationWithInclusions("b", pb.Invocation_ACTIVE, nil),
				insert.InvocationWithInclusions("c", pb.Invocation_FINALIZING, nil),
			)...)

			assertReady("a", false)
		})

		Convey(`INCLUDES FINALIZING which includes ACTIVE`, func() {
			testutil.MustApply(ctx, testutil.CombineMutations(
				insert.InvocationWithInclusions("a", pb.Invocation_FINALIZING, nil, "b"),
				insert.InvocationWithInclusions("b", pb.Invocation_FINALIZING, nil, "c"),
				insert.InvocationWithInclusions("c", pb.Invocation_ACTIVE, nil),
			)...)

			assertReady("a", false)
		})

		Convey(`Cycle with one node`, func() {
			testutil.MustApply(ctx, testutil.CombineMutations(
				insert.InvocationWithInclusions("a", pb.Invocation_FINALIZING, nil, "a"),
			)...)

			assertReady("a", true)
		})

		Convey(`Cycle with two nodes`, func() {
			testutil.MustApply(ctx, testutil.CombineMutations(
				insert.InvocationWithInclusions("a", pb.Invocation_FINALIZING, nil, "b"),
				insert.InvocationWithInclusions("b", pb.Invocation_FINALIZING, nil, "a"),
			)...)

			assertReady("a", true)
		})
	})
}

func TestFinalizeInvocation(t *testing.T) {
	Convey(`FinalizeInvocation`, t, func() {
		ctx := testutil.SpannerTestContext(t)
		ctx, sched := tq.TestingContext(ctx, nil)

		// This is flaky https://crbug.com/1042602#c19
		SkipConvey(`Changes the state and finalization time`, func() {
			testutil.MustApply(ctx, testutil.CombineMutations(
				insert.InvocationWithInclusions("x", pb.Invocation_FINALIZING, nil),
			)...)

			err := finalizeInvocation(ctx, "x")
			So(err, ShouldBeNil)

			var state pb.Invocation_State
			var finalizeTime time.Time
			testutil.MustReadRow(ctx, "Invocations", invocations.ID("x").Key(), map[string]interface{}{
				"State":        &state,
				"FinalizeTime": &finalizeTime,
			})
			So(state, ShouldEqual, pb.Invocation_FINALIZED)
			So(finalizeTime, ShouldNotResemble, time.Time{})
		})

		// This is flaky https://crbug.com/1042602#c21
		SkipConvey(`Enqueues more finalizing tasks`, func() {
			testutil.MustApply(ctx, testutil.CombineMutations(
				insert.InvocationWithInclusions("active", pb.Invocation_ACTIVE, nil, "x"),
				insert.InvocationWithInclusions("finalizing1", pb.Invocation_FINALIZING, nil, "x"),
				insert.InvocationWithInclusions("finalizing2", pb.Invocation_FINALIZING, nil, "x"),
				insert.InvocationWithInclusions("x", pb.Invocation_FINALIZING, nil),
			)...)

			err := finalizeInvocation(ctx, "x")
			So(err, ShouldBeNil)

			// Enqueued TQ tasks.
			So(sched.Tasks().Payloads(), ShouldResembleProto, []*taskspb.TryFinalizeInvocation{
				{InvocationId: "finalizing1"},
				{InvocationId: "finalizing2"},
			})
		})

		// This is flaky https://crbug.com/1042602#c17
		SkipConvey(`Enqueues more bq_export tasks`, func() {
			bq1, _ := proto.Marshal(&pb.BigQueryExport{
				Dataset: "dataset",
				Project: "project",
				Table:   "table",
			})
			bq2, _ := proto.Marshal(&pb.BigQueryExport{
				Dataset: "dataset",
				Project: "project2",
				Table:   "table1",
			})
			testutil.MustApply(ctx,
				insert.Invocation("x", pb.Invocation_FINALIZING, map[string]interface{}{
					"BigQueryExports": [][]byte{bq1, bq2},
				}),
			)

			err := finalizeInvocation(ctx, "x")
			So(err, ShouldBeNil)
			// Enqueued TQ tasks.
			So(sched.Tasks().Payloads(), ShouldResembleProto, []*taskspb.ExportInvocationToBQ{
				{InvocationId: "x", BqExport: &pb.BigQueryExport{Dataset: "dataset", Project: "project2", Table: "table1"}},
				{InvocationId: "x", BqExport: &pb.BigQueryExport{Dataset: "dataset", Project: "project", Table: "table"}},
			})
		})
	})
}
