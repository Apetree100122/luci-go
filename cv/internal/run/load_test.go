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

package run

import (
	"context"
	"fmt"
	"testing"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/timestamppb"

	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/gae/service/datastore"
	"go.chromium.org/luci/grpc/appstatus"

	"go.chromium.org/luci/cv/internal/common"
	"go.chromium.org/luci/cv/internal/cvtesting"

	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"
)

func TestLoadChildRuns(t *testing.T) {
	t.Parallel()

	Convey("LoadChildRuns works", t, func() {
		ct := cvtesting.Test{}
		ctx, cancel := ct.SetUp(t)
		defer cancel()

		put := func(runID common.RunID, depRuns common.RunIDs) {
			So(datastore.Put(ctx, &Run{
				ID:      runID,
				DepRuns: depRuns,
			}), ShouldBeNil)
		}

		const parentRun1 = common.RunID("parent/1-cow")

		const orphanRun = common.RunID("orphan/1-chicken")
		put(orphanRun, common.RunIDs{})
		out1, err := LoadChildRuns(ctx, parentRun1)
		So(err, ShouldBeNil)
		So(out1, ShouldHaveLength, 0)

		const pendingRun = common.RunID("child/1-pending")
		put(pendingRun, common.RunIDs{parentRun1})
		const runningRun = common.RunID("child/1-running")
		put(runningRun, common.RunIDs{parentRun1})

		out2, err := LoadChildRuns(ctx, parentRun1)
		So(err, ShouldBeNil)
		So(out2, ShouldHaveLength, 2)
	})
}

func TestLoadRunLogEntries(t *testing.T) {
	t.Parallel()

	Convey("LoadRunLogEntries works", t, func() {
		ct := cvtesting.Test{}
		ctx, cancel := ct.SetUp(t)
		defer cancel()

		ev := int64(1)
		put := func(runID common.RunID, entries ...*LogEntry) {
			So(datastore.Put(ctx, &RunLog{
				Run:     datastore.MakeKey(ctx, common.RunKind, string(runID)),
				ID:      ev,
				Entries: &LogEntries{Entries: entries},
			}), ShouldBeNil)
			ev += 1
		}

		const run1 = common.RunID("rust/123-1-beef")
		const run2 = common.RunID("dart/789-2-cafe")

		put(
			run1,
			&LogEntry{
				Time: timestamppb.New(ct.Clock.Now()),
				Kind: &LogEntry_Created_{Created: &LogEntry_Created{
					ConfigGroupId: "fi/rst",
				}},
			},
		)
		ct.Clock.Add(time.Minute)
		put(
			run1,
			&LogEntry{
				Time: timestamppb.New(ct.Clock.Now()),
				Kind: &LogEntry_ConfigChanged_{ConfigChanged: &LogEntry_ConfigChanged{
					ConfigGroupId: "se/cond",
				}},
			},
			&LogEntry{
				Time: timestamppb.New(ct.Clock.Now()),
				Kind: &LogEntry_TryjobsRequirementUpdated_{TryjobsRequirementUpdated: &LogEntry_TryjobsRequirementUpdated{}},
			},
		)

		ct.Clock.Add(time.Minute)
		put(
			run2,
			&LogEntry{
				Time: timestamppb.New(ct.Clock.Now()),
				Kind: &LogEntry_Created_{Created: &LogEntry_Created{
					ConfigGroupId: "fi/rst-but-run2",
				}},
			},
		)

		out1, err := LoadRunLogEntries(ctx, run1)
		So(err, ShouldBeNil)
		So(out1, ShouldHaveLength, 3)
		So(out1[0].GetCreated().GetConfigGroupId(), ShouldResemble, "fi/rst")
		So(out1[1].GetConfigChanged().GetConfigGroupId(), ShouldResemble, "se/cond")
		So(out1[2].GetTryjobsRequirementUpdated(), ShouldNotBeNil)

		out2, err := LoadRunLogEntries(ctx, run2)
		So(err, ShouldBeNil)
		So(out2, ShouldHaveLength, 1)
		So(out2[0].GetCreated().GetConfigGroupId(), ShouldResemble, "fi/rst-but-run2")
	})
}

func TestLoadRunsBuilder(t *testing.T) {
	t.Parallel()

	Convey("LoadRunsBuilder works", t, func() {
		ct := cvtesting.Test{}
		ctx, cancel := ct.SetUp(t)
		defer cancel()

		const lProject = "proj"
		// Run statuses are used in this test to ensure Runs were actually loaded.
		makeRun := func(id int, s Status) *Run {
			r := &Run{ID: common.RunID(fmt.Sprintf("%s/%03d", lProject, id)), Status: s}
			So(datastore.Put(ctx, r), ShouldBeNil)
			return r
		}

		r1 := makeRun(1, Status_RUNNING)
		r2 := makeRun(2, Status_CANCELLED)
		r3 := makeRun(3, Status_PENDING)
		r4 := makeRun(4, Status_SUCCEEDED)
		r201 := makeRun(201, Status_FAILED)
		r202 := makeRun(202, Status_FAILED)
		r404 := makeRun(404, Status_PENDING)
		r405 := makeRun(405, Status_PENDING)
		So(datastore.Delete(ctx, r404, r405), ShouldBeNil)

		Convey("Without checker", func() {
			Convey("Every Run exists", func() {
				verify := func(b LoadRunsBuilder) {
					runsA, errs := b.Do(ctx)
					So(errs, ShouldResemble, make(errors.MultiError, 2))
					So(runsA, ShouldResemble, []*Run{r201, r202})

					runsB, err := b.DoIgnoreNotFound(ctx)
					So(err, ShouldBeNil)
					So(runsB, ShouldResemble, runsA)
				}
				Convey("IDs", func() {
					verify(LoadRunsFromIDs(r201.ID, r202.ID))
				})
				Convey("keys", func() {
					verify(LoadRunsFromKeys(
						datastore.MakeKey(ctx, common.RunKind, string(r201.ID)),
						datastore.MakeKey(ctx, common.RunKind, string(r202.ID)),
					))
				})
			})

			Convey("A missing Run", func() {
				b := LoadRunsFromIDs(r404.ID)

				runsA, errs := b.Do(ctx)
				So(errs, ShouldResemble, errors.MultiError{datastore.ErrNoSuchEntity})
				So(runsA, ShouldResemble, []*Run{{ID: r404.ID}})

				runsB, err := b.DoIgnoreNotFound(ctx)
				So(err, ShouldBeNil)
				So(runsB, ShouldBeNil)
			})
			Convey("Mix of existing and missing", func() {
				b := LoadRunsFromIDs(r201.ID, r404.ID, r202.ID, r405.ID, r4.ID)

				runsA, errs := b.Do(ctx)
				So(errs, ShouldResemble, errors.MultiError{nil, datastore.ErrNoSuchEntity, nil, datastore.ErrNoSuchEntity, nil})
				So(runsA, ShouldResemble, []*Run{
					r201,
					{ID: r404.ID},
					r202,
					{ID: r405.ID},
					r4,
				})

				runsB, err := b.DoIgnoreNotFound(ctx)
				So(err, ShouldBeNil)
				So(runsB, ShouldResemble, []*Run{r201, r202, r4})
			})
		})

		Convey("With checker", func() {
			checker := fakeRunChecker{
				afterOnNotFound: appstatus.Error(codes.NotFound, "not-found-ds"),
			}

			Convey("No errors of any kind", func() {
				b := LoadRunsFromIDs(r201.ID, r202.ID, r4.ID).Checker(checker)

				runsA, errs := b.Do(ctx)
				So(errs, ShouldResemble, make(errors.MultiError, 3))
				So(runsA, ShouldResemble, []*Run{r201, r202, r4})

				runsB, err := b.DoIgnoreNotFound(ctx)
				So(err, ShouldBeNil)
				So(runsB, ShouldResemble, runsA)
			})

			Convey("Missing in datastore", func() {
				b := LoadRunsFromIDs(r404.ID).Checker(checker)

				runsA, errs := b.Do(ctx)
				So(errs[0], ShouldHaveAppStatus, codes.NotFound)
				So(runsA, ShouldResemble, []*Run{{ID: r404.ID}})

				runsB, err := b.DoIgnoreNotFound(ctx)
				So(err, ShouldBeNil)
				So(runsB, ShouldBeNil)
			})

			Convey("Mix", func() {
				checker.before = map[common.RunID]error{
					r1.ID: appstatus.Error(codes.NotFound, "not-found-before"),
					r2.ID: errors.New("before-oops"),
				}
				checker.after = map[common.RunID]error{
					r3.ID: appstatus.Error(codes.NotFound, "not-found-after"),
					r4.ID: errors.New("after-oops"),
				}
				Convey("only found and not found", func() {
					b := LoadRunsFromIDs(r201.ID, r1.ID, r202.ID, r3.ID, r404.ID).Checker(checker)

					runsA, errs := b.Do(ctx)
					So(errs[0], ShouldBeNil) // r201
					So(errs[1], ShouldErrLike, "not-found-before")
					So(errs[2], ShouldBeNil) // r202
					So(errs[3], ShouldErrLike, "not-found-after")
					So(errs[4], ShouldErrLike, "not-found-ds")
					So(runsA, ShouldResemble, []*Run{
						r201,
						{ID: r1.ID},
						r202,
						r3, // loaded & returned, despite errors
						{ID: r404.ID},
					})

					runsB, err := b.DoIgnoreNotFound(ctx)
					So(err, ShouldBeNil)
					So(runsB, ShouldResemble, []*Run{r201, r202})
				})
				Convey("of everything", func() {
					b := LoadRunsFromIDs(r201.ID, r1.ID, r2.ID, r3.ID, r4.ID, r404.ID).Checker(checker)

					runsA, errs := b.Do(ctx)
					So(errs[0], ShouldBeNil) // r201
					So(errs[1], ShouldErrLike, "not-found-before")
					So(errs[2], ShouldErrLike, "before-oops")
					So(errs[3], ShouldErrLike, "not-found-after")
					So(errs[4], ShouldErrLike, "after-oops")
					So(errs[5], ShouldErrLike, "not-found-ds")
					So(runsA, ShouldResemble, []*Run{
						r201,
						{ID: r1.ID},
						{ID: r2.ID},
						r3, // loaded & returned, despite errors
						r4, // loaded & returned, despite errors
						{ID: r404.ID},
					})

					runsB, err := b.DoIgnoreNotFound(ctx)
					So(err, ShouldErrLike, "before-oops")
					So(runsB, ShouldBeNil)
				})
			})
		})
	})
}

type fakeRunChecker struct {
	before          map[common.RunID]error
	beforeFunc      func(common.RunID) error // applied only if Run is not in `before`
	after           map[common.RunID]error
	afterOnNotFound error
}

func (f fakeRunChecker) Before(ctx context.Context, id common.RunID) error {
	err := f.before[id]
	if err == nil && f.beforeFunc != nil {
		err = f.beforeFunc(id)
	}
	return err
}

func (f fakeRunChecker) After(ctx context.Context, runIfFound *Run) error {
	if runIfFound == nil {
		return f.afterOnNotFound
	}
	return f.after[runIfFound.ID]
}
