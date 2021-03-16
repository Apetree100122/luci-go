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

package handler

import (
	"fmt"
	"testing"
	"time"

	"go.chromium.org/luci/common/clock"
	"go.chromium.org/luci/gae/service/datastore"

	"go.chromium.org/luci/cv/internal/changelist"
	"go.chromium.org/luci/cv/internal/common"
	"go.chromium.org/luci/cv/internal/cvtesting"
	"go.chromium.org/luci/cv/internal/prjmanager/pmtest"
	"go.chromium.org/luci/cv/internal/run"
	"go.chromium.org/luci/cv/internal/run/impl/state"

	. "github.com/smartystreets/goconvey/convey"
)

func TestCancel(t *testing.T) {
	t.Parallel()

	Convey("Cancel", t, func() {
		ct := cvtesting.Test{}
		ctx, close := ct.SetUp()
		defer close()
		var runID common.RunID = "chromium/111-1-deadbeef"
		var clid common.CLID = 11
		rs := &state.RunState{
			Run: run.Run{
				ID:         runID,
				CreateTime: clock.Now(ctx).UTC().Add(-2 * time.Minute),
				CLs:        []common.CLID{clid},
			},
		}
		So(datastore.Put(ctx, &changelist.CL{
			ID:             clid,
			IncompleteRuns: common.RunIDs{runID, "chromium/222-1-cafecafe"},
		}), ShouldBeNil)
		h := &Impl{}

		latestCL := func() *changelist.CL {
			cl := &changelist.CL{ID: clid}
			So(datastore.Get(ctx, cl), ShouldBeNil)
			return cl
		}

		Convey("Cancels PENDING Run", func() {
			rs.Run.Status = run.Status_PENDING
			res, err := h.Cancel(ctx, rs)
			So(err, ShouldBeNil)
			So(res.State.Run.Status, ShouldEqual, run.Status_CANCELLED)
			now := clock.Now(ctx).UTC()
			So(res.State.Run.StartTime, ShouldResemble, now)
			So(res.State.Run.EndTime, ShouldResemble, now)
			So(res.SideEffectFn, ShouldNotBeNil)
			So(datastore.RunInTransaction(ctx, res.SideEffectFn, nil), ShouldBeNil)
			So(res.PreserveEvents, ShouldBeFalse)
			pmtest.AssertReceivedRunFinished(ctx, rs.Run.ID)
			So(latestCL().IncompleteRuns.ContainsSorted(runID), ShouldBeFalse)
		})

		Convey("Cancels RUNNING Run", func() {
			rs.Run.Status = run.Status_RUNNING
			rs.Run.StartTime = clock.Now(ctx).UTC().Add(-1 * time.Minute)
			res, err := h.Cancel(ctx, rs)
			So(err, ShouldBeNil)
			So(res.State.Run.Status, ShouldEqual, run.Status_CANCELLED)
			now := clock.Now(ctx).UTC()
			So(res.State.Run.StartTime, ShouldResemble, now.Add(-1*time.Minute))
			So(res.State.Run.EndTime, ShouldResemble, now)
			So(res.SideEffectFn, ShouldNotBeNil)
			So(datastore.RunInTransaction(ctx, res.SideEffectFn, nil), ShouldBeNil)
			So(res.PreserveEvents, ShouldBeFalse)
			pmtest.AssertReceivedRunFinished(ctx, rs.Run.ID)
			So(latestCL().IncompleteRuns.ContainsSorted(runID), ShouldBeFalse)
		})

		Convey("Cancels SUBMITTING Run", func() {
			rs.Run.Status = run.Status_SUBMITTING
			res, err := h.Cancel(ctx, rs)
			So(err, ShouldBeNil)
			So(res.State, ShouldEqual, rs)
			So(res.SideEffectFn, ShouldBeNil)
			So(res.PreserveEvents, ShouldBeTrue)
		})

		statuses := []run.Status{
			run.Status_SUCCEEDED,
			run.Status_FAILED,
			run.Status_CANCELLED,
		}
		for _, status := range statuses {
			Convey(fmt.Sprintf("Noop when Run is %s", status), func() {
				rs.Run.Status = status
				rs.Run.StartTime = clock.Now(ctx).UTC().Add(-1 * time.Minute)
				rs.Run.EndTime = clock.Now(ctx).UTC().Add(-30 * time.Second)
				res, err := h.Cancel(ctx, rs)
				So(err, ShouldBeNil)
				So(res.State, ShouldEqual, rs)
				So(res.SideEffectFn, ShouldBeNil)
				So(res.PreserveEvents, ShouldBeFalse)
			})
		}
	})
}
