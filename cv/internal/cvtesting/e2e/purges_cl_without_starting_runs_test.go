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
	"testing"
	"time"

	"google.golang.org/protobuf/types/known/durationpb"

	cfgpb "go.chromium.org/luci/cv/api/config/v2"
	gf "go.chromium.org/luci/cv/internal/gerrit/gerritfake"

	. "github.com/smartystreets/goconvey/convey"
)

func TestPurgesCLWithoutOwner(t *testing.T) {
	t.Parallel()

	Convey("PM purges CLs without owner's email", t, func() {
		/////////////////////////    Setup   ////////////////////////////////
		ct := Test{}
		ctx, cancel := ct.SetUp()
		defer cancel()

		const (
			lProject = "infra"
			gHost    = "g-review"
			gRepo    = "re/po"
			gRef     = "refs/heads/main"
			gChange  = 43
		)

		ct.EnableCVRunManagement(ctx, lProject)
		ct.Cfg.Create(ctx, lProject, MakeCfgSingular("cg0", gHost, gRepo, gRef))

		ci := gf.CI(
			gChange, gf.Project(gRepo), gf.Ref(gRef),
			gf.Updated(ct.Now()), gf.CQ(+2, ct.Now(), gf.U("user-1")),
			gf.Owner("user-1"),
		)
		ci.GetOwner().Email = ""
		ct.GFake.AddFrom(gf.WithCIs(gHost, gf.ACLRestricted(lProject), ci))
		So(ct.MaxCQVote(ctx, gHost, gChange), ShouldEqual, 2)

		ct.LogPhase(ctx, "Run CV until CQ+2 vote is removed")
		So(ct.PMNotifier.UpdateConfig(ctx, lProject), ShouldBeNil)
		ct.RunUntil(ctx, func() bool {
			return ct.MaxCQVote(ctx, gHost, gChange) == 0
		})
		So(ct.LastMessage(gHost, gChange).GetMessage(), ShouldContainSubstring, "doesn't have a preferred email")

		ct.LogPhase(ctx, "Ensure PM had a chance to react to CLUpdated event")
		ct.RunUntil(ctx, func() bool {
			return len(ct.LoadProject(ctx, lProject).State.GetPcls()) == 0
		})
	})
}

func TestPurgesCLWithUnwatchedDeps(t *testing.T) {
	t.Parallel()

	Convey("PM purges CL with dep outside the project after waiting stabilization_delay", t, func() {
		/////////////////////////    Setup   ////////////////////////////////
		ct := Test{}
		ctx, cancel := ct.SetUp()
		defer cancel()

		const (
			lProject = "chromium"
			gHost    = "chromium-review.example.com"
			gRepo    = "chromium/src"
			gRef     = "refs/heads/main"
			gChange  = 33

			lProject2 = "webrtc"
			gHost2    = "webrtc-review.example.com"
			gRepo2    = "src"
			gChange2  = 22
		)
		// Enable CV management of both projects.
		ct.EnableCVRunManagement(ctx, lProject)
		ct.EnableCVRunManagement(ctx, lProject2)

		const stabilizationDelay = 2 * time.Minute
		cfg1 := MakeCfgSingular("cg0", gHost, gRepo, gRef)
		cfg1.GetConfigGroups()[0].CombineCls = &cfgpb.CombineCLs{
			StabilizationDelay: durationpb.New(stabilizationDelay),
		}
		ct.Cfg.Create(ctx, lProject, cfg1)
		ct.Cfg.Create(ctx, lProject2, MakeCfgSingular("cg0", gHost2, gRepo2, gRef))

		tStart := ct.Now()

		ct.GFake.AddFrom(gf.WithCIs(gHost, gf.ACLRestricted(lProject), gf.CI(
			gChange, gf.Project(gRepo), gf.Ref(gRef),
			gf.Updated(tStart), gf.CQ(+2, tStart, gf.U("user-1")),
			gf.Owner("user-1"),
			gf.Desc(fmt.Sprintf("T\n\nCq-Depend: webrtc:%d", gChange2)),
		)))
		ct.GFake.AddFrom(gf.WithCIs(gHost2, gf.ACLRestricted(lProject2), gf.CI(
			gChange2, gf.Project(gRepo2), gf.Ref(gRef),
			gf.Updated(tStart), gf.CQ(+2, tStart, gf.U("user-1")),
			gf.Owner("user-1"),
		)))

		ct.LogPhase(ctx, "Run CV until CQ+2 vote is removed")
		So(ct.PMNotifier.UpdateConfig(ctx, lProject), ShouldBeNil)
		ct.RunUntil(ctx, func() bool {
			return ct.MaxCQVote(ctx, gHost, gChange) == 0
		})
		m := ct.LastMessage(gHost, gChange)
		So(m, ShouldNotBeNil)
		So(m.GetDate().AsTime(), ShouldHappenAfter, tStart.Add(stabilizationDelay))
		So(m.GetMessage(), ShouldContainSubstring, "its deps are not watched by the same LUCI project")
		So(m.GetMessage(), ShouldContainSubstring, "https://webrtc-review.example.com/22")

		ct.LogPhase(ctx, "Ensure PM had a chance to react to CLUpdated event")
		ct.RunUntil(ctx, func() bool {
			return len(ct.LoadProject(ctx, lProject).State.GetPcls()) == 0
		})
	})
}

func TestPurgesCLWithMismatchedDepsMode(t *testing.T) {
	t.Parallel()

	Convey("PM purges CL with dep outside the project after waiting stabilization_delay", t, func() {
		/////////////////////////    Setup   ////////////////////////////////
		ct := Test{}
		ctx, cancel := ct.SetUp()
		defer cancel()

		const (
			lProject   = "chromiumos"
			gHost      = "chromium-review.example.com"
			gRepo      = "cros/platform"
			gRef       = "refs/heads/main"
			gChange44  = 44
			gChange45  = 45
			quickLabel = "Quick-Label"
		)

		ct.LogPhase(ctx, "Set up stack of 2 CLs with active combine_cls setting but differing modes")
		ct.EnableCVRunManagement(ctx, lProject)

		const stabilizationDelay = 5 * time.Minute
		cfg := MakeCfgSingular("cg0", gHost, gRepo, gRef)
		cfg.GetConfigGroups()[0].CombineCls = &cfgpb.CombineCLs{
			StabilizationDelay: durationpb.New(stabilizationDelay),
		}
		cfg.GetConfigGroups()[0].AdditionalModes = []*cfgpb.Mode{{
			CqLabelValue:    1,
			Name:            "QUICK_DRY_RUN",
			TriggeringLabel: quickLabel,
			TriggeringValue: 1,
		}}
		ct.Cfg.Create(ctx, lProject, cfg)

		tStart := ct.Now()
		ci44 := gf.CI(
			gChange44, gf.Project(gRepo), gf.Ref(gRef), gf.Updated(tStart),
			gf.Owner("user-1"),
			gf.CQ(+1, tStart, gf.U("user-1")), // Just DRY_RUN.
			gf.Desc(fmt.Sprintf("T\n\nCq-Depend: %d", gChange45)),
		)
		ci45 := gf.CI(
			gChange45, gf.Project(gRepo), gf.Ref(gRef), gf.Updated(tStart),
			gf.Owner("user-1"),
			// These 2 votes trigger QUICK_DRY_RUN.
			gf.CQ(+1, tStart, gf.U("user-1")),
			gf.Vote(quickLabel, +1, tStart, gf.U("user-1")),
			// Some other user triggering just the quickLabel, which is a noop.
			gf.Vote(quickLabel, +1, tStart.Add(-time.Minute), gf.U("user-2")),
		)
		ct.GFake.AddFrom(gf.WithCIs(gHost, gf.ACLRestricted(lProject), ci45, ci44))
		// Make ci45 depend on ci44 via Git relationship (ie make it a CL stack).
		ct.GFake.SetDependsOn(gHost, ci45, ci44)
		// Now, ci45 and ci44 must be tested only together.

		ct.LogPhase(ctx, "Run CV until both CLs are purged")
		So(ct.PMNotifier.UpdateConfig(ctx, lProject), ShouldBeNil)
		ct.RunUntil(ctx, func() bool {
			return (ct.MaxCQVote(ctx, gHost, gChange44) == 0 &&
				ct.MaxCQVote(ctx, gHost, gChange45) == 0 &&
				ct.MaxVote(ctx, gHost, gChange45, quickLabel) == 0)
		})

		ct.LogPhase(ctx, "Ensure purging happened only after stabilizationDelay")
		So(ct.LastMessage(gHost, gChange44).GetDate().AsTime(), ShouldHappenAfter, tStart.Add(stabilizationDelay))
		So(ct.LastMessage(gHost, gChange45).GetDate().AsTime(), ShouldHappenAfter, tStart.Add(stabilizationDelay))

		ct.LogPhase(ctx, "Ensure CL is no longer active in CV")
		ct.RunUntil(ctx, func() bool {
			return len(ct.LoadProject(ctx, lProject).State.GetPcls()) == 0
		})
	})
}

func TestPurgesSingularFullRunWithOpenDeps(t *testing.T) {
	t.Parallel()

	Convey("PM purges CL in Full Run mode if it has not submitted deps", t, func() {
		ct := Test{}
		ctx, cancel := ct.SetUp()
		defer cancel()

		const (
			lProject    = "infra"
			gHost       = "chromium-review.example.com"
			gRepo       = "luci-go"
			gRef        = "refs/heads/main"
			gChangeOpen = 44
			gChangeCQed = 45
		)

		ct.LogPhase(ctx, "Set up stack of 2 CLs")
		ct.EnableCVRunManagement(ctx, lProject)

		ct.Cfg.Create(ctx, lProject, MakeCfgSingular("cg0", gHost, gRepo, gRef))

		tStart := ct.Now()
		ciOpen := gf.CI(
			gChangeOpen, gf.Project(gRepo), gf.Ref(gRef), gf.Updated(tStart),
		)
		ciCQed := gf.CI(
			gChangeCQed, gf.Project(gRepo), gf.Ref(gRef), gf.Updated(tStart),
			gf.Owner("user-1"),
			gf.CQ(+2, tStart, gf.U("user-1")),
		)
		ct.GFake.AddFrom(gf.WithCIs(gHost, gf.ACLRestricted(lProject), ciOpen, ciCQed))
		ct.GFake.SetDependsOn(gHost, ciCQed, ciOpen)

		ct.LogPhase(ctx, "Run CV until CQ+2 is removed")
		So(ct.PMNotifier.UpdateConfig(ctx, lProject), ShouldBeNil)
		ct.RunUntil(ctx, func() bool {
			return ct.MaxCQVote(ctx, gHost, gChangeCQed) == 0
		})
		So(ct.LastMessage(gHost, gChangeCQed).GetMessage(), ShouldContainSubstring, `it has not yet submitted dependencies`)

		ct.LogPhase(ctx, "Ensure CL is no longer active in CV")
		ct.RunUntil(ctx, func() bool {
			return len(ct.LoadProject(ctx, lProject).State.GetPcls()) == 0
		})
	})
}

func TestPurgesCLCQDependingOnItself(t *testing.T) {
	t.Parallel()

	Convey("PM purges CL which CQ-Depends on itself", t, func() {
		/////////////////////////    Setup   ////////////////////////////////
		ct := Test{}
		ctx, cancel := ct.SetUp()
		defer cancel()

		const (
			lProject  = "chromiumos"
			gHost     = "chromium-review.example.com"
			gRepo     = "cros/platform"
			gRef      = "refs/heads/main"
			gChange44 = 44
		)

		ct.LogPhase(ctx, "Set up a CL depending on itself")
		ct.EnableCVRunManagement(ctx, lProject)
		cfg := MakeCfgSingular("cg0", gHost, gRepo, gRef)
		ct.Cfg.Create(ctx, lProject, cfg)
		tStart := ct.Now()
		ct.GFake.AddFrom(gf.WithCIs(gHost, gf.ACLRestricted(lProject), gf.CI(
			gChange44, gf.Project(gRepo), gf.Ref(gRef), gf.Updated(tStart),
			gf.Owner("user-1"),
			gf.CQ(+1, tStart, gf.U("user-1")),
			gf.Desc(fmt.Sprintf("T\n\nCq-Depend: %d", gChange44)),
		)))

		ct.LogPhase(ctx, "Run CV until CL is purged")
		So(ct.PMNotifier.UpdateConfig(ctx, lProject), ShouldBeNil)
		ct.RunUntil(ctx, func() bool {
			return ct.MaxCQVote(ctx, gHost, gChange44) == 0
		})
		So(ct.LastMessage(gHost, gChange44).GetMessage(), ShouldContainSubstring, "because it depends on itself")

		ct.LogPhase(ctx, "Ensure CL is no longer active in CV")
		ct.RunUntil(ctx, func() bool {
			return len(ct.LoadProject(ctx, lProject).State.GetPcls()) == 0
		})
	})
}
