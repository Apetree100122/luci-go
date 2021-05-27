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

package e2e

import (
	"fmt"
	"sort"
	"strings"
	"testing"
	"time"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	gerritpb "go.chromium.org/luci/common/proto/gerrit"
	"go.chromium.org/luci/gae/service/datastore"

	cvbqpb "go.chromium.org/luci/cv/api/bigquery/v1"
	cfgpb "go.chromium.org/luci/cv/api/config/v2"
	migrationpb "go.chromium.org/luci/cv/api/migration"
	"go.chromium.org/luci/cv/internal/common"
	gf "go.chromium.org/luci/cv/internal/gerrit/gerritfake"
	"go.chromium.org/luci/cv/internal/gerrit/trigger"
	"go.chromium.org/luci/cv/internal/migration"
	"go.chromium.org/luci/cv/internal/migration/cqdfake"
	"go.chromium.org/luci/cv/internal/run"

	. "github.com/smartystreets/goconvey/convey"
)

func TestCreatesSingularRun(t *testing.T) {
	t.Parallel()

	Convey("CV creates 1 CL Run, which gets canceled by the user", t, func() {
		/////////////////////////    Setup   ////////////////////////////////
		ct := Test{}
		ctx, cancel := ct.SetUp()
		defer cancel()

		const lProject = "infra"
		const gHost = "g-review"
		const gRepo = "re/po"
		const gRef = "refs/heads/main"
		const gChange = 33

		// TODO(tandrii): remove this once Run creation is not conditional on CV
		// managing Runs for a project.
		ct.EnableCVRunManagement(ctx, lProject)
		cfg := MakeCfgSingular("cg0", gHost, gRepo, gRef)
		ct.Cfg.Create(ctx, lProject, cfg)

		tStart := ct.Clock.Now()

		ct.GFake.AddFrom(gf.WithCIs(gHost, gf.ACLRestricted(lProject), gf.CI(
			gChange, gf.Project(gRepo), gf.Ref(gRef),
			gf.Owner("user-1"),
			gf.Updated(tStart), gf.CQ(+2, tStart, gf.U("user-2")),
		)))

		/////////////////////////    Run CV   ////////////////////////////////
		ct.LogPhase(ctx, "CV notices CL and starts the Run")
		So(ct.PMNotifier.UpdateConfig(ctx, lProject), ShouldBeNil)
		var r *run.Run
		ct.RunUntil(ctx, func() bool {
			r = ct.EarliestCreatedRunOf(ctx, lProject)
			return r != nil && r.Status == run.Status_RUNNING
		})
		So(ct.LoadGerritCL(ctx, gHost, gChange).IncompleteRuns.ContainsSorted(r.ID), ShouldBeTrue)

		// Under normal conditions, this would be done by the same TQ handler that
		// creates the Run, but with flaky Datastore this may come late.
		ct.LogPhase(ctx, "PM incorporates Run into component")
		ct.RunUntil(ctx, func() bool {
			cs := ct.LoadProject(ctx, lProject).State.GetComponents()
			return len(cs) == 1 && len(cs[0].GetPruns()) == 1
		})
		So(ct.LoadProject(ctx, lProject).State.GetComponents()[0].GetPruns()[0].GetId(), ShouldResemble, string(r.ID))

		ct.LogPhase(ctx, "User cancels the Run")
		ct.GFake.MutateChange(gHost, gChange, func(c *gf.Change) {
			gf.CQ(0, ct.Clock.Now(), "user-2")(c.Info)
			gf.Updated(ct.Clock.Now())(c.Info)
		})

		ct.LogPhase(ctx, "CV cancels the Run")
		ct.RunUntil(ctx, func() bool {
			r2 := ct.LoadRun(ctx, r.ID)
			p := ct.LoadProject(ctx, lProject)
			return r2.Status == run.Status_CANCELLED && len(p.State.GetComponents()) == 0
		})

		/////////////////////////    Verify    ////////////////////////////////
		So(ct.LoadGerritCL(ctx, gHost, gChange).IncompleteRuns.ContainsSorted(r.ID), ShouldBeFalse)
		So(ct.LoadProject(ctx, lProject).State.GetPcls(), ShouldBeEmpty)
		So(ct.LoadProject(ctx, lProject).State.GetComponents(), ShouldBeEmpty)
	})
}

func TestCreatesSingularRunCQDinCharge(t *testing.T) {
	t.Parallel()

	Convey("CV creates and finalizes a 1 CL Run while CQD is in charge", t, func() {
		/////////////////////////    Setup   ////////////////////////////////
		ct := Test{}
		ctx, cancel := ct.SetUp()
		defer cancel()

		const lProject = "e2e-always-create-runs"
		const gHost = "g-review"
		const gRepo = "re/po"
		const gRef = "refs/heads/main"
		const gChange = 33

		ct.DisableCVRunManagement(ctx)
		cfg := MakeCfgSingular("cg0", gHost, gRepo, gRef)
		ct.Cfg.Create(ctx, lProject, cfg)

		tStart := ct.Clock.Now()

		ct.GFake.AddFrom(gf.WithCIs(gHost, gf.ACLRestricted(lProject), gf.CI(
			gChange, gf.Project(gRepo), gf.Ref(gRef),
			gf.Owner("user-1"),
			gf.Updated(tStart), gf.CQ(+1, tStart, gf.U("user-2")),
		)))
		const expectedAttemptKey = "0a6686c7f19a33a3"

		/////////////////////////    Run CV   ////////////////////////////////
		ct.LogPhase(ctx, "CV and CQD notice CL and start a Run")

		So(ct.PMNotifier.UpdateConfig(ctx, lProject), ShouldBeNil)
		ct.MustCQD(ctx, lProject).SetCandidatesClbk(func() []*migrationpb.ReportedRun {
			if ct.MaxCQVote(ctx, gHost, gChange) == 0 {
				return nil
			}
			return []*migrationpb.ReportedRun{
				{
					Id: "", // CQD isn't aware of CV Runs while computing its own candidates.
					Attempt: &cvbqpb.Attempt{
						Key:           expectedAttemptKey,
						GerritChanges: []*cvbqpb.GerritChange{{Host: gHost, Change: gChange, Mode: cvbqpb.Mode_DRY_RUN}},
						Status:        cvbqpb.AttemptStatus_STARTED,
					},
				},
			}
		})

		var r *run.Run
		ct.RunUntil(ctx, func() bool {
			r = ct.EarliestCreatedRunOf(ctx, lProject)
			as := ct.MustCQD(ctx, lProject).ActiveAttemptKeys()
			return r != nil && r.Status == run.Status_RUNNING && len(as) == 1 && as[0] == expectedAttemptKey
		})
		So(r.ID.AttemptKey(), ShouldResemble, expectedAttemptKey)

		ct.LogPhase(ctx, "Dry Run passes in CQD and removes CQ+1 vote")
		ct.MustCQD(ctx, lProject).SetVerifyClbk(func(r *migrationpb.ReportedRun, _ bool) *migrationpb.ReportedRun {
			if r.GetAttempt().GetEndTime() != nil {
				return r // already finalized
			}
			return &migrationpb.ReportedRun{
				Id: "",
				Attempt: &cvbqpb.Attempt{
					Key:                  expectedAttemptKey,
					ClGroupKey:           "beef",
					EquivalentClGroupKey: "beef",
					GerritChanges:        []*cvbqpb.GerritChange{{Host: gHost, Change: gChange, Mode: cvbqpb.Mode_DRY_RUN}},
					Status:               cvbqpb.AttemptStatus_SUCCESS,
					Builds: []*cvbqpb.Build{
						{Critical: true, Host: "bb-host", Id: 22233, Origin: cvbqpb.Build_NOT_REUSABLE},
					},
					ConfigGroup: cfg.GetConfigGroups()[0].GetName(),
					LuciProject: lProject,
					StartTime:   timestamppb.New(tStart),
					EndTime:     timestamppb.New(ct.Clock.Now()),
				},
			}
		})
		ct.RunUntil(ctx, func() bool {
			return ct.MaxCQVote(ctx, gHost, gChange) == 0
		})

		// TODO(tandrii): implemenet handling in Run Manager and finish the test.
		// ct.LogPhase(ctx, "CV finalizes the Run")
		// ct.RunUntil(ctx, func() bool {
		// 	return ct.LoadRun(ctx, r.ID).Status == run.Status_SUCCEEDED
		// })
		// So(ct.LoadGerritCL(ctx, gHost, gChange).IncompleteRuns.ContainsSorted(r.ID), ShouldBeFalse)
		// ct.RunUntil(ctx, func() bool {
		// 	return len(ct.LoadProject(ctx, lProject).State.GetPcls()) == 0
		// })
		// TODO(qyearsley): assert no BQ row was sent.
	})
}

func TestCreatesSingularQuickDryRunSuccess(t *testing.T) {
	t.Parallel()

	Convey("CV creates 1 CL Quick Dry Run, which succeeds", t, func() {
		/////////////////////////    Setup   ////////////////////////////////
		ct := Test{}
		ctx, cancel := ct.SetUp()
		defer cancel()

		const lProject = "infra"
		const gHost = "g-review"
		const gRepo = "re/po"
		const gRef = "refs/heads/main"
		const gChange = 33
		const quickLabel = "Quick-Label"

		// TODO(tandrii): remove this once Run creation is not conditional on CV
		// managing Runs for a project.
		ct.EnableCVRunManagement(ctx, lProject)
		// Start CQDaemon.
		ct.MustCQD(ctx, lProject)

		cfg := MakeCfgSingular("cg0", gHost, gRepo, gRef)
		cfg.GetConfigGroups()[0].AdditionalModes = []*cfgpb.Mode{{
			Name:            string(run.QuickDryRun),
			CqLabelValue:    1,
			TriggeringValue: 1,
			TriggeringLabel: quickLabel,
		}}
		ct.Cfg.Create(ctx, lProject, cfg)

		tStart := ct.Clock.Now()

		ct.GFake.AddFrom(gf.WithCIs(gHost, gf.ACLRestricted(lProject), gf.CI(
			gChange, gf.Project(gRepo), gf.Ref(gRef),
			gf.Owner("user-1"),
			gf.Updated(tStart),
			gf.CQ(+1, tStart, gf.U("user-2")),
			gf.Vote(quickLabel, +1, tStart, gf.U("user-2")),
			// Spurious vote from user-3.
			gf.Vote(quickLabel, +5, tStart.Add(-10*time.Second), gf.U("user-3")),
		)))

		/////////////////////////    Run CV   ////////////////////////////////
		ct.LogPhase(ctx, "CV notices CL and starts the Run")
		So(ct.PMNotifier.UpdateConfig(ctx, lProject), ShouldBeNil)
		var r *run.Run
		ct.RunUntil(ctx, func() bool {
			r = ct.EarliestCreatedRunOf(ctx, lProject)
			return r != nil && r.Status == run.Status_RUNNING
		})
		So(r.Mode, ShouldEqual, run.QuickDryRun)

		ct.LogPhase(ctx, "CQDaemon posts starting message to the Gerrit CL")
		ct.RunUntil(ctx, func() bool {
			m := ct.LastMessage(gHost, gChange).GetMessage()
			return strings.Contains(m, cqdfake.StartingMessage) && strings.Contains(m, string(r.ID))
		})

		ct.LogPhase(ctx, "CQDaemon decides that QuickDryRun has passed and notifies CV")
		ct.Clock.Add(time.Minute)
		ct.MustCQD(ctx, lProject).SetVerifyClbk(
			func(r *migrationpb.ReportedRun, cvInCharge bool) *migrationpb.ReportedRun {
				r = proto.Clone(r).(*migrationpb.ReportedRun)
				r.Attempt.Status = cvbqpb.AttemptStatus_SUCCESS
				r.Attempt.Substatus = cvbqpb.AttemptSubstatus_NO_SUBSTATUS
				return r
			},
		)
		ct.RunUntil(ctx, func() bool {
			return nil == datastore.Get(ctx, &migration.VerifiedCQDRun{ID: r.ID})
		})

		// At this point CV must either report the `gChange` to be excluded from
		// active attempts OR (if CV has already finalized this Run) not return this
		// Run as active.
		activeRuns := ct.MigrationFetchActiveRuns(ctx, lProject)
		excludedCLs := ct.MigrationFetchExcludedCLs(ctx, lProject)
		if len(activeRuns) > 0 {
			So(activeRuns[0].Id, ShouldResemble, string(r.ID))
			So(excludedCLs, ShouldResemble, []string{fmt.Sprintf("%s/%d", gHost, gChange)})
		}

		ct.LogPhase(ctx, "CV finalizes the run and sends BQ event")
		var finalRun *run.Run
		ct.RunUntil(ctx, func() bool {
			finalRun = ct.LoadRun(ctx, r.ID)
			proj := ct.LoadProject(ctx, lProject)
			cl := ct.LoadGerritCL(ctx, gHost, gChange)
			return run.IsEnded(finalRun.Status) &&
				len(proj.State.GetComponents()) == 0 &&
				!cl.IncompleteRuns.ContainsSorted(r.ID)
		})

		So(finalRun.Status, ShouldEqual, run.Status_SUCCEEDED)
		So(ct.MaxCQVote(ctx, gHost, gChange), ShouldEqual, 0)
		So(ct.MaxVote(ctx, gHost, gChange, quickLabel), ShouldEqual, 0)
		So(ct.LastMessage(gHost, gChange).GetMessage(), ShouldContainSubstring, "Quick dry run: This CL passed the CQ dry run.")

		// Verify that BQ row was exported.
		// TODO(qyearsley): implement.
	})
}

func TestCreatesSingularFullRunSuccess(t *testing.T) {
	t.Parallel()

	Convey("CV creates 1 CL Full Run, which succeeds", t, func() {
		/////////////////////////    Setup   ////////////////////////////////
		ct := Test{}
		ctx, cancel := ct.SetUp()
		defer cancel()

		const lProject = "infra"
		const gHost = "g-review"
		const gRepo = "re/po"
		const gRef = "refs/heads/main"
		const gChange = 33
		const gPatchSet = 6

		// TODO(tandrii): remove this once Run creation is not conditional on CV
		// managing Runs for a project.
		ct.EnableCVRunManagement(ctx, lProject)
		// Start CQDaemon.
		ct.MustCQD(ctx, lProject)

		cfg := MakeCfgSingular("cg0", gHost, gRepo, gRef)
		ct.Cfg.Create(ctx, lProject, cfg)

		tStart := ct.Clock.Now()

		ct.GFake.AddFrom(gf.WithCIs(gHost, gf.ACLRestricted(lProject), gf.CI(
			gChange, gf.Project(gRepo), gf.Ref(gRef),
			gf.PS(gPatchSet),
			gf.Owner("user-1"),
			gf.Updated(tStart),
			gf.CQ(+2, tStart, gf.U("user-2")),
		)))

		/////////////////////////    Run CV   ////////////////////////////////
		ct.LogPhase(ctx, "CV notices CL and starts the Run")
		So(ct.PMNotifier.UpdateConfig(ctx, lProject), ShouldBeNil)
		var r *run.Run
		ct.RunUntil(ctx, func() bool {
			r = ct.EarliestCreatedRunOf(ctx, lProject)
			return r != nil && r.Status == run.Status_RUNNING
		})
		So(r.Mode, ShouldEqual, run.FullRun)

		ct.LogPhase(ctx, "CQDaemon posts starting message to the Gerrit CL")
		ct.RunUntil(ctx, func() bool {
			m := ct.LastMessage(gHost, gChange).GetMessage()
			return strings.Contains(m, cqdfake.StartingMessage) && strings.Contains(m, string(r.ID))
		})

		ct.LogPhase(ctx, "CQDaemon decides that FullRun has passed and notifies CV to submit")
		ct.Clock.Add(time.Minute)
		ct.MustCQD(ctx, lProject).SetVerifyClbk(
			func(r *migrationpb.ReportedRun, cvInCharge bool) *migrationpb.ReportedRun {
				r = proto.Clone(r).(*migrationpb.ReportedRun)
				r.Attempt.Status = cvbqpb.AttemptStatus_SUCCESS
				r.Attempt.Substatus = cvbqpb.AttemptSubstatus_NO_SUBSTATUS
				return r
			},
		)
		ct.RunUntil(ctx, func() bool {
			res, err := datastore.Exists(ctx, &migration.VerifiedCQDRun{ID: r.ID})
			return err == nil && res.All()
		})

		// At this point CV must either report the `gChange` to be excluded from
		// active attempts OR (if CV has already finalized this Run) not return this
		// Run as active.
		activeRuns := ct.MigrationFetchActiveRuns(ctx, lProject)
		excludedCLs := ct.MigrationFetchExcludedCLs(ctx, lProject)
		if len(activeRuns) > 0 {
			So(activeRuns[0].Id, ShouldResemble, string(r.ID))
			So(excludedCLs, ShouldResemble, []string{fmt.Sprintf("%s/%d", gHost, gChange)})
		}

		ct.LogPhase(ctx, "CV submits the run and sends BQ event")
		var finalRun *run.Run
		ct.RunUntil(ctx, func() bool {
			finalRun = ct.LoadRun(ctx, r.ID)
			proj := ct.LoadProject(ctx, lProject)
			cl := ct.LoadGerritCL(ctx, gHost, gChange)
			return run.IsEnded(finalRun.Status) &&
				len(proj.State.GetComponents()) == 0 &&
				!cl.IncompleteRuns.ContainsSorted(r.ID)
		})

		So(finalRun.Status, ShouldEqual, run.Status_SUCCEEDED)
		ci := ct.GFake.GetChange(gHost, gChange).Info
		So(ci.GetStatus(), ShouldEqual, gerritpb.ChangeStatus_MERGED)
		So(ci.GetRevisions()[ci.GetCurrentRevision()].GetNumber(), ShouldEqual, int32(gPatchSet+1))

		// Verify that BQ row was exported.
		// TODO(qyearsley): implement.
	})
}

func TestCreatesSingularDryRunAborted(t *testing.T) {
	t.Parallel()

	Convey("CV creates 1 CL Run, which gets canceled by the user", t, func() {
		/////////////////////////    Setup   ////////////////////////////////
		ct := Test{}
		ctx, cancel := ct.SetUp()
		defer cancel()

		const lProject = "infra"
		const gHost = "g-review"
		const gRepo = "re/po"
		const gRef = "refs/heads/main"
		const gChange = 33

		ct.EnableCVRunManagement(ctx, lProject)
		// Start CQDaemon.
		ct.MustCQD(ctx, lProject)

		cfg := MakeCfgSingular("cg0", gHost, gRepo, gRef)
		ct.Cfg.Create(ctx, lProject, cfg)

		tStart := ct.Clock.Now()

		ct.GFake.AddFrom(gf.WithCIs(gHost, gf.ACLRestricted(lProject), gf.CI(
			gChange, gf.Project(gRepo), gf.Ref(gRef),
			gf.Owner("user-1"),
			gf.Updated(tStart),
			gf.CQ(+1, tStart, gf.U("user-2")),
		)))

		/////////////////////////    Run CV   ////////////////////////////////
		ct.LogPhase(ctx, "CV notices CL and starts the Run")
		So(ct.PMNotifier.UpdateConfig(ctx, lProject), ShouldBeNil)
		var r *run.Run
		ct.RunUntil(ctx, func() bool {
			r = ct.EarliestCreatedRunOf(ctx, lProject)
			return r != nil && r.Status == run.Status_RUNNING
		})
		So(r.Mode, ShouldEqual, run.DryRun)

		ct.LogPhase(ctx, "CQDaemon posts starting message to the Gerrit CL")
		ct.RunUntil(ctx, func() bool {
			m := ct.LastMessage(gHost, gChange).GetMessage()
			return strings.Contains(m, cqdfake.StartingMessage) && strings.Contains(m, string(r.ID))
		})

		ct.LogPhase(ctx, "User aborts the run by removing a vote")
		ct.Clock.Add(time.Minute)
		ct.GFake.MutateChange(gHost, gChange, func(c *gf.Change) {
			gf.ResetVotes(c.Info, trigger.CQLabelName)
			gf.Updated(ct.Clock.Now())(c.Info)
		})

		ct.LogPhase(ctx, "CQDaemon stops working on the Run")
		ct.RunUntil(ctx, func() bool {
			return len(ct.MustCQD(ctx, lProject).ActiveAttemptKeys()) == 0
		})

		ct.LogPhase(ctx, "CV finalizes the Run and sends BQ event")
		ct.RunUntil(ctx, func() bool {
			r = ct.LoadRun(ctx, r.ID)
			return run.IsEnded(r.Status)
		})
		So(r.Status, ShouldEqual, run.Status_CANCELLED)
		// Verify that BQ row was exported.
		// TODO(qyearsley): implement.
	})
}

func TestCreatesSingularRunWithDeps(t *testing.T) {
	Convey("CV creates Run in singular config in presence of Git dependencies.", t, func() {
		/////////////////////////    Setup   ////////////////////////////////
		ct := Test{}
		ctx, cancel := ct.SetUp()
		defer cancel()

		const lProject = "infra"
		const gHost = "g-review"
		const gRepo = "re/po"
		const gRef = "refs/heads/main"

		// TODO(tandrii): remove this once Run creation is not conditional on CV
		// managing Runs for a project.
		ct.EnableCVRunManagement(ctx, lProject)
		cfg := MakeCfgSingular("cg0", gHost, gRepo, gRef)
		ct.Cfg.Create(ctx, lProject, cfg)

		tStart := ct.Clock.Now()

		ct.LogPhase(ctx, "Set Git chain of 13 depends on 12, and vote CQ+1 on 13")
		ct.GFake.AddFrom(gf.WithCIs(gHost, gf.ACLRestricted(lProject), gf.CI(
			12, gf.Project(gRepo), gf.Ref(gRef), gf.PS(2),
		)))
		ct.GFake.AddFrom(gf.WithCIs(gHost, gf.ACLRestricted(lProject), gf.CI(
			13, gf.Project(gRepo), gf.Ref(gRef), gf.PS(3),
			gf.Updated(tStart),
			gf.CQ(+1, tStart, gf.U("user-1")),
		)))
		ct.GFake.SetDependsOn(gHost, "13_3", "12_2")

		ct.LogPhase(ctx, "CV starts dry Run on just the 13")
		So(ct.PMNotifier.UpdateConfig(ctx, lProject), ShouldBeNil)
		var r13 *run.Run
		ct.RunUntil(ctx, func() bool {
			r13 = ct.EarliestCreatedRunOf(ctx, lProject)
			return r13 != nil && r13.Status == run.Status_RUNNING
		})
		So(r13.Mode, ShouldResemble, run.DryRun)
		So(r13.CLs, ShouldResemble, common.CLIDs{ct.LoadGerritCL(ctx, gHost, 13).ID})

		ct.LogPhase(ctx, "User votes CQ+2 on 12, CV starts Run on just 12 and doesn't touch 13")
		ct.Clock.Add(time.Minute)
		ct.GFake.MutateChange(gHost, 12, func(c *gf.Change) {
			gf.CQ(+2, ct.Clock.Now(), "user-1")(c.Info)
			gf.Updated(ct.Clock.Now())(c.Info)
		})
		ct.RunUntil(ctx, func() bool {
			return len(ct.LoadGerritCL(ctx, gHost, 12).IncompleteRuns) > 0
		})
		r12 := ct.LoadRun(ctx, ct.LoadGerritCL(ctx, gHost, 12).IncompleteRuns[0])
		So(r12.Mode, ShouldResemble, run.FullRun)
		So(r12.CLs, ShouldResemble, common.CLIDs{ct.LoadGerritCL(ctx, gHost, 12).ID})
		So(ct.LoadRun(ctx, r13.ID).Status, ShouldEqual, run.Status_RUNNING)
		So(r12.CreateTime, ShouldHappenAfter, r13.CreateTime)

		ct.LogPhase(ctx, "User upgrades to CQ+2 on 13, which terminates old Run.\n"+
			"Due to 13 depending on not yet submitted 12, CV purges 13 by removing CQ+2 vote")
		ct.Clock.Add(time.Minute)
		ct.GFake.MutateChange(gHost, 13, func(c *gf.Change) {
			gf.CQ(+2, ct.Clock.Now(), "user-1")(c.Info)
			gf.Updated(ct.Clock.Now())(c.Info)
		})
		// TODO(crbug/1188763): fix Run Manager to cancel since trigger has changed.
		// ct.RunUntil(ctx, func() bool {
		// 	// CQ+2 vote removed.
		// 	return trigger.Find(ct.GFake.GetChange(gHost, 13).Info) == nil
		// })
	})
}

func TestCreatesMultiCLsFullRunSuccess(t *testing.T) {
	t.Parallel()

	Convey("CV creates 3 CLs Full Run, which succeeds", t, func() {
		/////////////////////////    Setup   ////////////////////////////////
		ct := Test{}
		ctx, cancel := ct.SetUp()
		defer cancel()

		const lProject = "infra"
		const gHost = "g-review"
		const gRepo = "re/po"
		const gRef = "refs/heads/main"
		const gChange1 = 11
		const gChange2 = 22
		const gChange3 = 33
		const gPatchSet = 6

		// TODO(tandrii): remove this once Run creation is not conditional on CV
		// managing Runs for a project.
		ct.EnableCVRunManagement(ctx, lProject)
		// Start CQDaemon.
		ct.MustCQD(ctx, lProject)

		cfg := MakeCfgCombinable("cg0", gHost, gRepo, gRef)
		ct.Cfg.Create(ctx, lProject, cfg)

		tStart := ct.Clock.Now()

		// Git Relationship: <base> -> ci2 -> ci3 -> ci1
		// and ci2 depends on ci1 using Cq-Depend git footer which forms a cycle.
		ci1 := gf.CI(
			gChange1, gf.Project(gRepo), gf.Ref(gRef),
			gf.PS(gPatchSet),
			gf.Owner("user-1"),
			gf.Updated(tStart),
			gf.CQ(+2, tStart, gf.U("user-2")),
			gf.Desc(fmt.Sprintf("This is the first CL\nCq-Depend: %d", gChange3)),
		)
		ci2 := gf.CI(
			gChange2, gf.Project(gRepo), gf.Ref(gRef),
			gf.PS(gPatchSet),
			gf.Owner("user-1"),
			gf.Updated(tStart),
			gf.CQ(+2, tStart, gf.U("user-2")),
		)
		ci3 := gf.CI(
			gChange3, gf.Project(gRepo), gf.Ref(gRef),
			gf.PS(gPatchSet),
			gf.Owner("user-1"),
			gf.Updated(tStart),
			gf.CQ(+2, tStart, gf.U("user-2")),
		)
		ct.GFake.AddFrom(gf.WithCIs(gHost, gf.ACLRestricted(lProject), ci1, ci2, ci3))
		ct.GFake.SetDependsOn(gHost, ci3, ci2)
		ct.GFake.SetDependsOn(gHost, ci1, ci3)

		/////////////////////////    Run CV   ////////////////////////////////
		ct.LogPhase(ctx, "CV discovers all CLs")
		So(ct.PMNotifier.UpdateConfig(ctx, lProject), ShouldBeNil)
		clids := make(common.CLIDs, 3)
		ct.RunUntil(ctx, func() bool {
			for i, change := range []int64{gChange1, gChange2, gChange3} {
				cl := ct.LoadGerritCL(ctx, gHost, change)
				if cl == nil {
					return false
				}
				clids[i] = cl.ID
			}
			sort.Sort(clids)
			return true
		})

		ct.LogPhase(ctx, "CV starts a Run with all 3 CLs")
		var r *run.Run
		ct.RunUntil(ctx, func() bool {
			r = ct.EarliestCreatedRunOf(ctx, lProject)
			return r != nil && r.Status == run.Status_RUNNING
		})
		So(r.Mode, ShouldEqual, run.FullRun)
		runCLIDs := r.CLs
		sort.Sort(runCLIDs)
		So(r.CLs, ShouldResemble, clids)

		ct.LogPhase(ctx, "CQDaemon posts starting message to each Gerrit CLs")
		ct.RunUntil(ctx, func() bool {
			for _, change := range []int64{gChange1, gChange2, gChange3} {
				m := ct.LastMessage(gHost, change).GetMessage()
				if !strings.Contains(m, cqdfake.StartingMessage) || !strings.Contains(m, string(r.ID)) {
					return false
				}
			}
			return true
		})

		ct.LogPhase(ctx, "CQDaemon decides that FullRun has passed and notifies CV to submit")
		ct.Clock.Add(time.Minute)
		ct.MustCQD(ctx, lProject).SetVerifyClbk(
			func(r *migrationpb.ReportedRun, cvInCharge bool) *migrationpb.ReportedRun {
				r = proto.Clone(r).(*migrationpb.ReportedRun)
				r.Attempt.Status = cvbqpb.AttemptStatus_SUCCESS
				r.Attempt.Substatus = cvbqpb.AttemptSubstatus_NO_SUBSTATUS
				return r
			},
		)
		ct.RunUntil(ctx, func() bool {
			res, err := datastore.Exists(ctx, &migration.VerifiedCQDRun{ID: r.ID})
			return err == nil && res.All()
		})

		// At this point CV must either report the `gChange` to be excluded from
		// active attempts OR (if CV has already finalized this Run) not return this
		// Run as active.
		activeRuns := ct.MigrationFetchActiveRuns(ctx, lProject)
		excludedCLs := ct.MigrationFetchExcludedCLs(ctx, lProject)
		if len(activeRuns) > 0 {
			So(activeRuns[0].Id, ShouldResemble, string(r.ID))
			sort.Strings(excludedCLs)
			expected := []string{
				fmt.Sprintf("%s/%d", gHost, gChange1),
				fmt.Sprintf("%s/%d", gHost, gChange2),
				fmt.Sprintf("%s/%d", gHost, gChange3),
			}
			sort.Strings(expected)
			So(excludedCLs, ShouldResemble, expected)
		}

		ct.LogPhase(ctx, "CV submits the run and sends BQ event")
		var finalRun *run.Run
		ct.RunUntil(ctx, func() bool {
			finalRun = ct.LoadRun(ctx, r.ID)
			if !run.IsEnded(finalRun.Status) {
				return false
			}
			if proj := ct.LoadProject(ctx, lProject); len(proj.State.GetComponents()) > 0 {
				return false
			}
			for _, change := range []int64{gChange1, gChange2, gChange3} {
				cl := ct.LoadGerritCL(ctx, gHost, change)
				if cl.IncompleteRuns.ContainsSorted(r.ID) {
					return false
				}
			}
			return true
		})

		So(finalRun.Status, ShouldEqual, run.Status_SUCCEEDED)
		ci1 = ct.GFake.GetChange(gHost, gChange1).Info
		ci2 = ct.GFake.GetChange(gHost, gChange2).Info
		ci3 = ct.GFake.GetChange(gHost, gChange3).Info
		for _, ci := range []*gerritpb.ChangeInfo{ci1, ci2, ci3} {
			So(ci.GetStatus(), ShouldEqual, gerritpb.ChangeStatus_MERGED)
			So(ci.GetRevisions()[ci.GetCurrentRevision()].GetNumber(), ShouldEqual, int32(gPatchSet+1))
		}
		// verify submission order: [ci2, ci3, ci1]
		So(ci2.GetUpdated().AsTime(), ShouldHappenBefore, ci3.GetUpdated().AsTime())
		So(ci3.GetUpdated().AsTime(), ShouldHappenBefore, ci1.GetUpdated().AsTime())

		// Verify that BQ row was exported.
		// TODO(qyearsley): implement.
	})
}
