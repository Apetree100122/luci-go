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

package triager

import (
	"fmt"
	"testing"
	"time"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"go.chromium.org/luci/common/clock/testclock"

	cfgpb "go.chromium.org/luci/cv/api/config/v2"
	"go.chromium.org/luci/cv/internal/changelist"
	"go.chromium.org/luci/cv/internal/common"
	"go.chromium.org/luci/cv/internal/configs/prjcfg"
	"go.chromium.org/luci/cv/internal/cvtesting"
	"go.chromium.org/luci/cv/internal/prjmanager/prjpb"
	"go.chromium.org/luci/cv/internal/run"

	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"
)

func TestDepsTriage(t *testing.T) {
	t.Parallel()

	Convey("Component's PCL deps triage", t, func() {
		ct := cvtesting.Test{}
		ctx, cancel := ct.SetUp(t)
		defer cancel()

		// Truncate start time point s.t. easy to see diff in test failures.
		epoch := testclock.TestRecentTimeUTC.Truncate(10000 * time.Second)
		dryRun := func(t time.Time) *run.Triggers {
			return &run.Triggers{CqVoteTrigger: &run.Trigger{Mode: string(run.DryRun), Time: timestamppb.New(t)}}
		}
		fullRun := func(t time.Time) *run.Triggers {
			return &run.Triggers{CqVoteTrigger: &run.Trigger{Mode: string(run.FullRun), Time: timestamppb.New(t)}}
		}

		sup := &simplePMState{
			pb: &prjpb.PState{},
			cgs: []*prjcfg.ConfigGroup{
				{ID: "hash/singular", Content: &cfgpb.ConfigGroup{}},
				{ID: "hash/combinable", Content: &cfgpb.ConfigGroup{CombineCls: &cfgpb.CombineCLs{}}},
				{ID: "hash/another", Content: &cfgpb.ConfigGroup{}},
			},
		}
		const singIdx, combIdx, anotherIdx = 0, 1, 2

		do := func(pcl *prjpb.PCL, cgIdx int32) *triagedDeps {
			backup := prjpb.PState{}
			proto.Merge(&backup, sup.pb)

			// Actual component doesn't matter in this test.
			td := triageDeps(ctx, pcl, cgIdx, pmState{sup})
			So(sup.pb, ShouldResembleProto, &backup) // must not be modified
			return td
		}

		Convey("Singluar and Combinable behave the same", func() {
			sameTests := func(name string, cgIdx int32) {
				Convey(name, func() {
					Convey("no deps", func() {
						sup.pb.Pcls = []*prjpb.PCL{
							{Clid: 33, ConfigGroupIndexes: []int32{cgIdx}},
						}
						td := do(sup.pb.Pcls[0], cgIdx)
						So(td, cvtesting.SafeShouldResemble, &triagedDeps{})
						So(td.OK(), ShouldBeTrue)
					})

					Convey("Valid CL stack CQ+1", func() {
						sup.pb.Pcls = []*prjpb.PCL{
							{Clid: 31, ConfigGroupIndexes: []int32{cgIdx}, Triggers: dryRun(epoch.Add(3 * time.Second))},
							{Clid: 32, ConfigGroupIndexes: []int32{cgIdx}, Triggers: dryRun(epoch.Add(2 * time.Second))},
							{Clid: 33, ConfigGroupIndexes: []int32{cgIdx}, Triggers: dryRun(epoch.Add(1 * time.Second)),
								Deps: []*changelist.Dep{
									{Clid: 31, Kind: changelist.DepKind_SOFT},
									{Clid: 32, Kind: changelist.DepKind_HARD},
								}},
						}
						td := do(sup.PCL(33), cgIdx)
						So(td, cvtesting.SafeShouldResemble, &triagedDeps{
							lastCQVoteTriggered: epoch.Add(3 * time.Second),
						})
						So(td.OK(), ShouldBeTrue)
					})

					Convey("Not yet loaded deps", func() {
						sup.pb.Pcls = []*prjpb.PCL{
							// 31 isn't in PCLs yet
							{Clid: 32, Status: prjpb.PCL_UNKNOWN},
							{Clid: 33, ConfigGroupIndexes: []int32{cgIdx}, Triggers: dryRun(epoch.Add(1 * time.Second)),
								Deps: []*changelist.Dep{
									{Clid: 31, Kind: changelist.DepKind_SOFT},
									{Clid: 32, Kind: changelist.DepKind_HARD},
								}},
						}
						pcl33 := sup.PCL(33)
						td := do(pcl33, cgIdx)
						So(td, cvtesting.SafeShouldResemble, &triagedDeps{notYetLoaded: pcl33.GetDeps()})
						So(td.OK(), ShouldBeTrue)
					})

					Convey("Unwatched", func() {
						sup.pb.Pcls = []*prjpb.PCL{
							{Clid: 31, Status: prjpb.PCL_UNWATCHED},
							{Clid: 32, Status: prjpb.PCL_DELETED},
							{Clid: 33, ConfigGroupIndexes: []int32{cgIdx}, Triggers: dryRun(epoch.Add(1 * time.Second)),
								Deps: []*changelist.Dep{
									{Clid: 31, Kind: changelist.DepKind_SOFT},
									{Clid: 32, Kind: changelist.DepKind_HARD},
								}},
						}
						pcl33 := sup.PCL(33)
						td := do(pcl33, cgIdx)
						So(td, cvtesting.SafeShouldResemble, &triagedDeps{
							invalidDeps: &changelist.CLError_InvalidDeps{
								Unwatched: pcl33.GetDeps(),
							},
						})
						So(td.OK(), ShouldBeFalse)
					})

					Convey("Submitted can be in any config group and they are OK deps", func() {
						sup.pb.Pcls = []*prjpb.PCL{
							{Clid: 32, ConfigGroupIndexes: []int32{anotherIdx}, Submitted: true},
							{Clid: 33, ConfigGroupIndexes: []int32{cgIdx}, Triggers: dryRun(epoch.Add(1 * time.Second)),
								Deps: []*changelist.Dep{{Clid: 32, Kind: changelist.DepKind_HARD}}},
						}
						pcl33 := sup.PCL(33)
						td := do(pcl33, cgIdx)
						So(td, cvtesting.SafeShouldResemble, &triagedDeps{submitted: pcl33.GetDeps()})
						So(td.OK(), ShouldBeTrue)
					})

					Convey("Wrong config group", func() {
						sup.pb.Pcls = []*prjpb.PCL{
							{Clid: 31, Triggers: dryRun(epoch.Add(3 * time.Second)), ConfigGroupIndexes: []int32{anotherIdx}},
							{Clid: 32, Triggers: dryRun(epoch.Add(2 * time.Second)), ConfigGroupIndexes: []int32{anotherIdx, cgIdx}},
							{Clid: 33, Triggers: dryRun(epoch.Add(1 * time.Second)), ConfigGroupIndexes: []int32{cgIdx},
								Deps: []*changelist.Dep{
									{Clid: 31, Kind: changelist.DepKind_SOFT},
									{Clid: 32, Kind: changelist.DepKind_HARD},
								}},
						}
						pcl33 := sup.PCL(33)
						td := do(pcl33, cgIdx)
						So(td, cvtesting.SafeShouldResemble, &triagedDeps{
							lastCQVoteTriggered: epoch.Add(3 * time.Second),
							invalidDeps: &changelist.CLError_InvalidDeps{
								WrongConfigGroup: pcl33.GetDeps(),
							},
						})
						So(td.OK(), ShouldBeFalse)
					})

					Convey("Too many deps", func() {
						// Create maxAllowedDeps+1 deps.
						sup.pb.Pcls = make([]*prjpb.PCL, 0, maxAllowedDeps+2)
						deps := make([]*changelist.Dep, 0, maxAllowedDeps+1)
						for i := 1; i <= maxAllowedDeps+1; i++ {
							sup.pb.Pcls = append(sup.pb.Pcls, &prjpb.PCL{
								Clid:               int64(1000 + i),
								ConfigGroupIndexes: []int32{cgIdx},
								Triggers:           dryRun(epoch.Add(time.Second)),
							})
							deps = append(deps, &changelist.Dep{Clid: int64(1000 + i), Kind: changelist.DepKind_SOFT})
						}
						// Add the PCL with the above deps.
						sup.pb.Pcls = append(sup.pb.Pcls, &prjpb.PCL{
							Clid:               2000,
							ConfigGroupIndexes: []int32{cgIdx},
							Triggers:           dryRun(epoch.Add(time.Second)),
							Deps:               deps,
						})
						td := do(sup.PCL(2000), cgIdx)
						So(td, cvtesting.SafeShouldResemble, &triagedDeps{
							lastCQVoteTriggered: epoch.Add(time.Second),
							invalidDeps: &changelist.CLError_InvalidDeps{
								TooMany: &changelist.CLError_InvalidDeps_TooMany{
									Actual:     maxAllowedDeps + 1,
									MaxAllowed: maxAllowedDeps,
								},
							},
						})
						So(td.OK(), ShouldBeFalse)
					})
				})
			}
			sameTests("singular", singIdx)
			sameTests("combinable", combIdx)
		})

		Convey("Singular speciality", func() {
			sup.pb.Pcls = []*prjpb.PCL{
				{
					Clid: 31, ConfigGroupIndexes: []int32{singIdx},
					Triggers: dryRun(epoch.Add(3 * time.Second)),
				},
				{
					Clid: 32, ConfigGroupIndexes: []int32{singIdx},
					Triggers: fullRun(epoch.Add(2 * time.Second)), // not happy about its dep.
					Deps:     []*changelist.Dep{{Clid: 31, Kind: changelist.DepKind_HARD}},
				},
				{
					Clid: 33, ConfigGroupIndexes: []int32{singIdx},
					Triggers: dryRun(epoch.Add(3 * time.Second)), // doesn't care about deps.
					Deps: []*changelist.Dep{
						{Clid: 31, Kind: changelist.DepKind_SOFT},
						{Clid: 32, Kind: changelist.DepKind_HARD},
					},
				},
			}
			Convey("dry run doesn't care about deps' triggers", func() {
				pcl33 := sup.PCL(33)
				td := do(pcl33, singIdx)
				So(td, cvtesting.SafeShouldResemble, &triagedDeps{
					lastCQVoteTriggered: epoch.Add(3 * time.Second),
				})
			})
			Convey("full run doesn't allow any dep by default", func() {
				pcl32 := sup.PCL(32)
				td := do(pcl32, singIdx)
				So(td, cvtesting.SafeShouldResemble, &triagedDeps{
					lastCQVoteTriggered: epoch.Add(3 * time.Second),
					invalidDeps: &changelist.CLError_InvalidDeps{
						SingleFullDeps: pcl32.GetDeps(),
					},
				})
				So(td.OK(), ShouldBeFalse)

				Convey("unless allow_submit_with_open_deps is true", func() {
					sup.cgs[singIdx].Content.Verifiers = &cfgpb.Verifiers{
						GerritCqAbility: &cfgpb.Verifiers_GerritCQAbility{
							AllowSubmitWithOpenDeps: true,
						},
					}
					td := do(pcl32, singIdx)
					So(td, cvtesting.SafeShouldResemble, &triagedDeps{
						lastCQVoteTriggered: epoch.Add(3 * time.Second),
					})
					So(td.OK(), ShouldBeTrue)

					Convey("but not if dep is soft", func() {
						// Soft dependency (ie via Cq-Depend) won't be submitted as part a
						// single Submit gerrit RPC, so it can't be allowed.
						pcl32.GetDeps()[0].Kind = changelist.DepKind_SOFT
						td := do(pcl32, singIdx)
						So(td, cvtesting.SafeShouldResemble, &triagedDeps{
							lastCQVoteTriggered: epoch.Add(3 * time.Second),
							invalidDeps: &changelist.CLError_InvalidDeps{
								SingleFullDeps: pcl32.GetDeps(),
							},
						})
						So(td.OK(), ShouldBeFalse)
					})
				})

				Convey("unless the user is an MCE dogfooder and deps are HARD", func() {
					// TODO(crbug/1470341) remove this test if chained cq votes
					// is enabled by default.
					pcl31 := sup.PCL(31)
					pcl31.Triggers = nil
					pcl32 := sup.PCL(32)
					pcl32.Triggers.CqVoteTrigger.Email = "test@example.org"
					ct.AddMember("test@example.org", common.MCEDogfooderGroup)

					// triage with HARD dep. It should be good.
					td := do(pcl32, singIdx)
					So(td.OK(), ShouldBeTrue)
					// triage with a SOFT dep. This should fail, as chained cq
					// votes only support HARD deps.
					pcl32.GetDeps()[0].Kind = changelist.DepKind_SOFT
					td = do(pcl32, singIdx)
					So(td, cvtesting.SafeShouldResemble, &triagedDeps{
						invalidDeps: &changelist.CLError_InvalidDeps{
							SingleFullDeps: pcl32.GetDeps(),
						},
					})
					So(td.OK(), ShouldBeFalse)
				})
			})
		})

		Convey("Full run with chained CQ votes", func() {
			voter := "test@example.org"
			ct.AddMember(voter, common.MCEDogfooderGroup)
			sup.pb.Pcls = []*prjpb.PCL{
				{
					Clid: 31, ConfigGroupIndexes: []int32{singIdx},
				},
				{
					Clid: 32, ConfigGroupIndexes: []int32{singIdx},
					Deps: []*changelist.Dep{
						{Clid: 31, Kind: changelist.DepKind_HARD},
					},
				},
				{
					Clid: 33, ConfigGroupIndexes: []int32{singIdx},
					Deps: []*changelist.Dep{
						{Clid: 31, Kind: changelist.DepKind_HARD},
						{Clid: 32, Kind: changelist.DepKind_HARD},
					},
				},
			}

			Convey("Single vote on the topmost CL", func() {
				pcl33 := sup.PCL(33)
				pcl33.Triggers = fullRun(epoch)
				pcl33.Triggers.CqVoteTrigger.Email = voter
				td := do(pcl33, singIdx)

				// The triage dep result should be OK(), but have
				// the not-yet-voted deps in needToTrigger
				So(td.OK(), ShouldBeTrue)
				So(td.needToTrigger, ShouldResembleProto, []*changelist.Dep{
					{Clid: 31, Kind: changelist.DepKind_HARD},
					{Clid: 32, Kind: changelist.DepKind_HARD},
				})
			})
			Convey("a dep already has CQ+2", func() {
				pcl31 := sup.PCL(31)
				pcl31.Triggers = fullRun(epoch)
				pcl31.Triggers.CqVoteTrigger.Email = voter
				pcl33 := sup.PCL(33)
				pcl33.Triggers = fullRun(epoch)
				pcl33.Triggers.CqVoteTrigger.Email = voter
				td := do(pcl33, singIdx)

				So(td.OK(), ShouldBeTrue)
				So(td.needToTrigger, ShouldResembleProto, []*changelist.Dep{
					{Clid: 32, Kind: changelist.DepKind_HARD},
				})
			})
			Convey("a dep has CQ+1", func() {
				pcl31 := sup.PCL(31)
				pcl31.Triggers = dryRun(epoch)
				pcl31.Triggers.CqVoteTrigger.Email = voter
				pcl33 := sup.PCL(33)
				pcl33.Triggers = fullRun(epoch)
				pcl33.Triggers.CqVoteTrigger.Email = voter

				// triageDep should still put the dep with CQ+1 in
				// needToTrigger, so that PM will schedule a TQ task to override
				// the CQ vote with CQ+2.
				td := do(pcl33, singIdx)
				So(td.OK(), ShouldBeTrue)
				So(td.needToTrigger, ShouldResembleProto, []*changelist.Dep{
					{Clid: 31, Kind: changelist.DepKind_HARD},
					{Clid: 32, Kind: changelist.DepKind_HARD},
				})
			})
		})
		Convey("Combinable speciality", func() {
			// Setup valid deps; sub-tests wll mutate this to become invalid.
			sup.pb.Pcls = []*prjpb.PCL{
				{
					Clid: 31, ConfigGroupIndexes: []int32{combIdx},
					Triggers: dryRun(epoch.Add(3 * time.Second)),
				},
				{
					Clid: 32, ConfigGroupIndexes: []int32{combIdx},
					Triggers: dryRun(epoch.Add(2 * time.Second)),
					Deps:     []*changelist.Dep{{Clid: 31, Kind: changelist.DepKind_HARD}},
				},
				{
					Clid: 33, ConfigGroupIndexes: []int32{combIdx},
					Triggers: dryRun(epoch.Add(1 * time.Second)),
					Deps: []*changelist.Dep{
						{Clid: 31, Kind: changelist.DepKind_SOFT},
						{Clid: 32, Kind: changelist.DepKind_HARD},
					},
				},
			}
			Convey("dry run expects all deps to be dry", func() {
				pcl32 := sup.PCL(32)
				Convey("ok", func() {
					td := do(pcl32, combIdx)
					So(td, cvtesting.SafeShouldResemble, &triagedDeps{lastCQVoteTriggered: epoch.Add(3 * time.Second)})
				})

				Convey("... not full runs", func() {
					// TODO(tandrii): this can and should be supported.
					sup.PCL(31).Triggers.CqVoteTrigger.Mode = string(run.FullRun)
					td := do(pcl32, combIdx)
					So(td, cvtesting.SafeShouldResemble, &triagedDeps{
						lastCQVoteTriggered: epoch.Add(3 * time.Second),
						invalidDeps: &changelist.CLError_InvalidDeps{
							CombinableMismatchedMode: pcl32.GetDeps(),
						},
					})
				})
			})
			Convey("full run considers any dep incompatible", func() {
				pcl33 := sup.PCL(33)
				Convey("ok", func() {
					for _, pcl := range sup.pb.GetPcls() {
						pcl.Triggers.CqVoteTrigger.Mode = string(run.FullRun)
					}
					td := do(pcl33, combIdx)
					So(td, cvtesting.SafeShouldResemble, &triagedDeps{lastCQVoteTriggered: epoch.Add(3 * time.Second)})
				})
				Convey("... not dry runs", func() {
					sup.PCL(32).Triggers.CqVoteTrigger.Mode = string(run.FullRun)
					td := do(pcl33, combIdx)
					So(td, cvtesting.SafeShouldResemble, &triagedDeps{
						lastCQVoteTriggered: epoch.Add(3 * time.Second),
						invalidDeps: &changelist.CLError_InvalidDeps{
							CombinableMismatchedMode: []*changelist.Dep{{Clid: 32, Kind: changelist.DepKind_HARD}},
						},
					})
					So(td.OK(), ShouldBeFalse)
				})
			})
		})

		Convey("iterateNotSubmitted works", func() {
			d1 := &changelist.Dep{Clid: 1}
			d2 := &changelist.Dep{Clid: 2}
			d3 := &changelist.Dep{Clid: 3}
			pcl := &prjpb.PCL{}
			td := &triagedDeps{}

			iterate := func() (out []*changelist.Dep) {
				td.iterateNotSubmitted(pcl, func(dep *changelist.Dep) { out = append(out, dep) })
				return
			}

			Convey("no deps", func() {
				So(iterate(), ShouldBeEmpty)
			})
			Convey("only submitted", func() {
				td.submitted = []*changelist.Dep{d3, d1, d2}
				pcl.Deps = []*changelist.Dep{d3, d1, d2} // order must be the same
				So(iterate(), ShouldBeEmpty)
			})
			Convey("some submitted", func() {
				pcl.Deps = []*changelist.Dep{d3, d1, d2}
				td.submitted = []*changelist.Dep{d3}
				So(iterate(), ShouldResembleProto, []*changelist.Dep{d1, d2})
				td.submitted = []*changelist.Dep{d1}
				So(iterate(), ShouldResembleProto, []*changelist.Dep{d3, d2})
				td.submitted = []*changelist.Dep{d2}
				So(iterate(), ShouldResembleProto, []*changelist.Dep{d3, d1})
			})
			Convey("none submitted", func() {
				pcl.Deps = []*changelist.Dep{d3, d1, d2}
				So(iterate(), ShouldResembleProto, []*changelist.Dep{d3, d1, d2})
			})
			Convey("notYetLoaded deps are iterated over, too", func() {
				pcl.Deps = []*changelist.Dep{d3, d1, d2}
				td.notYetLoaded = []*changelist.Dep{d3}
				td.submitted = []*changelist.Dep{d2}
				So(iterate(), ShouldResembleProto, []*changelist.Dep{d3, d1})
			})
			Convey("panic on invalid usage", func() {
				Convey("wrong PCL", func() {
					pcl.Deps = []*changelist.Dep{d3, d1, d2}
					td.submitted = []*changelist.Dep{d1, d2, d3} // wrong order
					So(func() { iterate() }, ShouldPanicLike, fmt.Errorf("(wrong PCL?)"))
				})
			})
		})
	})
}
