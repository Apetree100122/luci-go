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

package state

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"go.chromium.org/luci/common/clock"
	"go.chromium.org/luci/common/data/stringset"
	"go.chromium.org/luci/common/logging"

	"go.chromium.org/luci/cv/internal/changelist"
	"go.chromium.org/luci/cv/internal/common"
	"go.chromium.org/luci/cv/internal/configs/prjcfg"
	"go.chromium.org/luci/cv/internal/gerrit/cfgmatcher"
	"go.chromium.org/luci/cv/internal/gerrit/poller"
	"go.chromium.org/luci/cv/internal/prjmanager"
	"go.chromium.org/luci/cv/internal/prjmanager/clpurger"
	"go.chromium.org/luci/cv/internal/prjmanager/cltriggerer"
	"go.chromium.org/luci/cv/internal/prjmanager/itriager"
	"go.chromium.org/luci/cv/internal/prjmanager/prjpb"
	"go.chromium.org/luci/cv/internal/run"
	"go.chromium.org/luci/cv/internal/tracing"
)

type RunNotifier interface {
	Start(ctx context.Context, id common.RunID) error
	PokeNow(ctx context.Context, id common.RunID) error
	Cancel(ctx context.Context, id common.RunID, reason string) error
	UpdateConfig(ctx context.Context, id common.RunID, hash string, eversion int64) error
}

// Handler handles state transitions of a project.
type Handler struct {
	CLMutator       *changelist.Mutator
	PMNotifier      *prjmanager.Notifier
	RunNotifier     RunNotifier
	CLPurger        *clpurger.Purger
	CLTriggerer     *cltriggerer.Triggerer
	CLPoller        *poller.Poller
	ComponentTriage itriager.Triage
}

// UpdateConfig updates PM to the latest config version.
func (h *Handler) UpdateConfig(ctx context.Context, s *State) (*State, SideEffect, error) {
	s.ensureNotYetCloned()

	meta, err := prjcfg.GetLatestMeta(ctx, s.PB.GetLuciProject())
	if err != nil {
		return nil, nil, err
	}

	switch meta.Status {
	case prjcfg.StatusEnabled:
		if s.PB.GetStatus() == prjpb.Status_STARTED && meta.Hash() == s.PB.GetConfigHash() {
			return s, nil, nil // already up-to-date.
		}

		// Tell poller to update ASAP. It doesn't need to wait for a transaction as
		// it's OK for poller to be temporarily more up-to-date than PM.
		if err := h.CLPoller.Poke(ctx, s.PB.GetLuciProject()); err != nil {
			return nil, nil, err
		}

		if s.PB.Status == prjpb.Status_STARTED {
			s = s.cloneShallow(prjpb.LogReason_CONFIG_CHANGED)
		} else {
			s = s.cloneShallow(prjpb.LogReason_CONFIG_CHANGED, prjpb.LogReason_STATUS_CHANGED)
			s.PB.Status = prjpb.Status_STARTED
		}
		s.PB.ConfigHash = meta.Hash()
		s.PB.ConfigGroupNames = meta.ConfigGroupNames

		if s.configGroups, err = meta.GetConfigGroups(ctx); err != nil {
			return nil, nil, err
		}
		s.cfgMatcher = cfgmatcher.LoadMatcherFromConfigGroups(ctx, s.configGroups, &meta)

		if err = s.reevalPCLs(ctx); err != nil {
			return nil, nil, err
		}
		// New config may mean new conditions for Run creation. Re-triaging all
		// components is required.
		s.PB.Components = markForTriage(s.PB.GetComponents())

		// We may have been in STOPPING phase, in which case incomplete runs may
		// still be finalizing themselves after receiving Cancel event from us.
		// It's harmless to send them UpdateConfig message, too. Eventually, they'll
		// complete finalization, send us OnRunFinished event and then we'll remove
		// them from the state anyway.
		return s, &UpdateIncompleteRunsConfig{
			RunNotifier: h.RunNotifier,
			EVersion:    meta.EVersion,
			Hash:        meta.Hash(),
			RunIDs:      s.PB.IncompleteRuns(),
		}, err

	case prjcfg.StatusDisabled, prjcfg.StatusNotExists:
		// Intentionally not catching up with new ConfigHash (if any),
		// since it's not actionable and also simpler.
		switch s.PB.GetStatus() {
		case prjpb.Status_STATUS_UNSPECIFIED:
			// Project entity doesn't exist. No need to create it.
			return s, nil, nil
		case prjpb.Status_STOPPED:
			return s, nil, nil
		case prjpb.Status_STARTED:
			s = s.cloneShallow(prjpb.LogReason_STATUS_CHANGED)
			s.PB.Status = prjpb.Status_STOPPING
			fallthrough
		case prjpb.Status_STOPPING:
			if err := h.CLPoller.Poke(ctx, s.PB.GetLuciProject()); err != nil {
				return nil, nil, err
			}
			runs := s.PB.IncompleteRuns()
			if len(runs) == 0 {
				s = s.cloneShallow(prjpb.LogReason_STATUS_CHANGED)
				s.PB.Status = prjpb.Status_STOPPED
				return s, nil, nil
			}
			return s, &CancelIncompleteRuns{
				RunNotifier: h.RunNotifier,
				RunIDs:      s.PB.IncompleteRuns(),
			}, nil
		default:
			panic(fmt.Errorf("unexpected project status: %d", s.PB.GetStatus()))
		}
	default:
		panic(fmt.Errorf("unexpected config status: %d", meta.Status))
	}
}

// Poke propagates "the poke" downstream to Poller & Runs.
func (h *Handler) Poke(ctx context.Context, s *State) (*State, SideEffect, error) {
	s.ensureNotYetCloned()

	// First, check if UpdateConfig if necessary.
	switch newState, sideEffect, err := h.UpdateConfig(ctx, s); {
	case err != nil:
		return nil, nil, err
	case newState != s:
		// UpdateConfig noticed a change and its SideEffectFn will propagate it
		// downstream.
		return newState, sideEffect, nil
	}

	// Propagate downstream directly.
	if err := h.CLPoller.Poke(ctx, s.PB.GetLuciProject()); err != nil {
		return nil, nil, err
	}
	if err := h.pokeRuns(ctx, s); err != nil {
		return nil, nil, err
	}
	// Force re-triage of all components.
	s = s.cloneShallow()
	s.PB.Components = markForTriage(s.PB.GetComponents())
	return s, nil, nil
}

// OnRunsCreated updates state after new Runs were created.
func (h *Handler) OnRunsCreated(ctx context.Context, s *State, created common.RunIDs) (_ *State, __ SideEffect, err error) {
	s.ensureNotYetCloned()

	ctx, span := tracing.Start(ctx, "go.chromium.org/luci/cv/internal/prjmanager/impl/state/OnRunsCreated")
	defer func() { tracing.End(span, err) }()

	// Check if PM is already aware of these Runs.
	remaining := created.Set()
	s.PB.IterIncompleteRuns(func(r *prjpb.PRun, _ *prjpb.Component) (stop bool) {
		delete(remaining, common.RunID(r.GetId()))
		return len(remaining) == 0 // stop if nothing left
	})
	if len(remaining) == 0 {
		return s, nil, nil
	}

	switch s.PB.GetStatus() {
	case prjpb.Status_STARTED:
		s = s.cloneShallow()
		if err := s.addCreatedRuns(ctx, remaining); err != nil {
			return nil, nil, err
		}
		return s, nil, nil
	case prjpb.Status_STOPPED, prjpb.Status_STOPPING:
		// This should not normally happen, but may under rare conditions.
		switch incomplete, err := incompleteRuns(ctx, remaining); {
		case err != nil:
			return nil, nil, err
		case len(incomplete) == 0:
			// All the Runs have actually already finished. Nothing to do, and this if
			// fine.
			return s, nil, nil
		default:
			logging.Errorf(ctx, "RunCreated events for %s on %s Project Manager", incomplete, s.PB.GetStatus())
			return s, &CancelIncompleteRuns{RunNotifier: h.RunNotifier, RunIDs: incomplete}, nil
		}
	default:
		panic(fmt.Errorf("unexpected project status: %d", s.PB.GetStatus()))
	}
}

// OnRunsFinished updates state after Runs were finished.
func (h *Handler) OnRunsFinished(ctx context.Context, s *State, finished map[common.RunID]run.Status) (_ *State, __ SideEffect, err error) {
	s.ensureNotYetCloned()

	_, span := tracing.Start(ctx, "go.chromium.org/luci/cv/internal/prjmanager/impl/state/OnRunsFinished")
	defer func() { tracing.End(span, err) }()

	// This is rarely a noop, so assume state is modified for simplicity.
	s = s.cloneShallow()
	incompleteRunsCount := s.removeFinishedRuns(finished)
	if s.PB.GetStatus() == prjpb.Status_STOPPING && incompleteRunsCount == 0 {
		s.LogReasons = append(s.LogReasons, prjpb.LogReason_STATUS_CHANGED)
		s.PB.Status = prjpb.Status_STOPPED
		return s, nil, nil
	}
	return s, nil, nil
}

// OnCLsUpdated updates state as a result of new changes to CLs.
//
// clEVersions must map CL's ID to CL's EVersion.
// clEVersions is mutated.
func (h *Handler) OnCLsUpdated(ctx context.Context, s *State, clEVersions map[int64]int64) (_ *State, __ SideEffect, err error) {
	s.ensureNotYetCloned()

	ctx, span := tracing.Start(ctx, "go.chromium.org/luci/cv/internal/prjmanager/impl/state/OnCLsUpdated")
	defer func() { tracing.End(span, err) }()

	if s.PB.GetStatus() != prjpb.Status_STARTED {
		// Ignore all incoming CL events. If PM is re-enabled, then first full
		// poll will force re-sending of OnCLsUpdated event for all still
		// interesting CLs.
		return s, nil, nil
	}

	// Avoid doing anything in cases where all CL updates sent due to recent full
	// poll iff we already know about each CL based on its EVersion.
	s.filterOutUpToDate(clEVersions)
	if len(clEVersions) == 0 {
		return s, nil, nil
	}

	// Most likely there will be changes to state.
	s = s.cloneShallow()
	if err := s.evalUpdatedCLs(ctx, clEVersions); err != nil {
		return nil, nil, err
	}
	return s, nil, nil
}

// OnPurgesCompleted updates state as a result of completed purge operations.
func (h *Handler) OnPurgesCompleted(ctx context.Context, s *State, events []*prjpb.PurgeCompleted) (_ *State, __ SideEffect, err error) {
	s.ensureNotYetCloned()

	ctx, span := tracing.Start(ctx, "go.chromium.org/luci/cv/internal/prjmanager/impl/state/OnPurgesCompleted")
	defer func() { tracing.End(span, err) }()

	opIDs := stringset.New(len(events))
	for _, e := range events {
		opIDs.Add(e.GetOperationId())
	}
	// Give 1 minute grace before expiring purging tasks. This doesn't change
	// correctness, but decreases probability of starting another purge before
	// PM observes CLUpdated event with results of prior purge.
	expireCutOff := clock.Now(ctx).Add(-time.Minute)

	deleted := map[int64]struct{}{}
	out, mutated := s.PB.COWPurgingCLs(func(p *prjpb.PurgingCL) *prjpb.PurgingCL {
		if opIDs.Has(p.GetOperationId()) {
			deleted[p.GetClid()] = struct{}{}
			return nil // delete
		}
		if p.GetDeadline().AsTime().Before(expireCutOff) {
			logging.Debugf(ctx, "PurgingCL %d %q expired", p.GetClid(), p.GetOperationId())
			deleted[p.GetClid()] = struct{}{}
			return nil // delete
		}
		return p // keep as is
	}, nil)
	if !mutated {
		return s, nil, nil
	}
	s = s.cloneShallow()
	s.PB.PurgingCls = out

	// Must mark affected components for re-evaluation.
	if s.PB.GetRepartitionRequired() {
		return s, nil, nil
	}
	cs, mutatedComponents := s.PB.COWComponents(func(c *prjpb.Component) *prjpb.Component {
		if c.GetTriageRequired() {
			return c
		}
		for _, id := range c.GetClids() {
			if _, yes := deleted[id]; yes {
				c = c.CloneShallow()
				c.TriageRequired = true
				return c
			}
		}
		return c
	}, nil)
	if mutatedComponents {
		s.PB.Components = cs
	}
	return s, nil, nil
}

// ExecDeferred performs previously postponed actions, notably creating Runs.
func (h *Handler) ExecDeferred(ctx context.Context, s *State) (_ *State, __ SideEffect, err error) {
	s.ensureNotYetCloned()

	ctx, span := tracing.Start(ctx, "go.chromium.org/luci/cv/internal/prjmanager/impl/state/ExecDeferred")
	defer func() { tracing.End(span, err) }()

	if s.PB.GetStatus() != prjpb.Status_STARTED {
		return s, nil, nil
	}

	mutated := false
	if s.PB.GetRepartitionRequired() || len(s.PB.GetCreatedPruns()) > 0 {
		s = s.cloneShallow()
		mutated = true
		cat := s.categorizeCLs(ctx)
		if err := s.loadActiveIntoPCLs(ctx, cat); err != nil {
			return nil, nil, err
		}
		s.repartition(cat)
	}

	var sideEffect SideEffect
	switch actions, saveForDebug, err := h.triageComponents(ctx, s); {
	case err != nil:
		if !mutated {
			return nil, nil, err
		}
		// Don't lose progress made so far.
		logging.Warningf(ctx, "Failed to triageComponents %s, but proceeding to save repartitioned state", err)
	case len(actions) > 0 || saveForDebug:
		if !mutated {
			if saveForDebug {
				s = s.cloneShallow(prjpb.LogReason_DEBUG)
			} else {
				s = s.cloneShallow()
			}
			mutated = true
		}
		sideEffect, err = h.actOnComponents(ctx, s, actions)
		if err != nil {
			return nil, nil, err
		}
	}

	switch t, tPB, asap := earliestDecisionTime(s.PB.GetComponents()); {
	case asap:
		t = clock.Now(ctx)
		tPB = timestamppb.New(t)
		fallthrough
	case tPB != nil && !proto.Equal(tPB, s.PB.GetNextEvalTime()):
		if !mutated {
			s = s.cloneShallow()
		}
		s.PB.NextEvalTime = tPB
		fallthrough
	case tPB != nil:
		// Always create a new task if there is NextEvalTime. If it is in the
		// future, it'll be deduplicated as needed.
		if err := h.PMNotifier.TasksBinding.Dispatch(ctx, s.PB.GetLuciProject(), t); err != nil {
			return nil, nil, err
		}
	}
	return s, sideEffect, nil
}

// OnTriggeringCLsCompleted manages the tracked TriggeringCL ops with the op completion results.
func (h *Handler) OnTriggeringCLsCompleted(ctx context.Context, s *State, succeeded, failed, skipped []*prjpb.TriggeringCLsCompleted_OpResult) (_ *State, __ SideEffect, err error) {
	s.ensureNotYetCloned()

	ctx, span := tracing.Start(ctx, "go.chromium.org/luci/cv/internal/prjmanager/impl/state/OnTriggeringCLsCompleted")
	defer func() { tracing.End(span, err) }()

	// This function handles the following cases.
	// 1. The PCL of a given TriggeringCL has the CQ vote.
	// The happiest path. Remove the TriggeringCL task.
	//
	// 2. The op succceeded but the PCL hasn't been updated yet.
	// Keep the TriggeringCL in the state to prevent the PCL from being
	// triaged again for TriggeringCLsTask.
	//
	// 3. The Op was expired or skipped.
	// Remove it from the state, and then let the component triager decides
	// what to do next.
	//
	// 4. The op failed.
	// This can happen only if the op failed due to a permanent failure before
	// the op gets expired. Then, this function will schedule PurgingCLTasks for
	// the originating CLs.
	opsToRemove := stringset.New(len(failed) + len(skipped))
	for _, e := range failed {
		opsToRemove.Add(e.GetOperationId())
	}
	for _, e := range skipped {
		opsToRemove.Add(e.GetOperationId())
	}
	// Give 1 minute grace before expiring tasks. This doesn't change
	// correctness, but decreases probability of starting another triggering
	// task before PM observes and processes CLUpdated event from the prior
	// operation.
	expireCutOff := clock.Now(ctx).Add(-time.Minute)

	deleted := map[int64]struct{}{}
	out, mutated := s.PB.COWTriggeringCLs(func(tcl *prjpb.TriggeringCL) *prjpb.TriggeringCL {
		if opsToRemove.Has(tcl.GetOperationId()) {
			deleted[tcl.GetClid()] = struct{}{}
			return nil // delete
		}
		if tcl.GetDeadline().AsTime().Before(expireCutOff) {
			logging.Debugf(ctx, "TriggeringCL %d %q expired", tcl.GetClid(), tcl.GetOperationId())
			deleted[tcl.GetClid()] = struct{}{}
			return nil // delete
		}

		// Note that the below doesn't check whether the tcl was one of
		// the ops reported in `suceeded`.
		//
		// This is to ensure that OnTriggeringCLsCompleted() removes all the ops
		// that were reported as succeeded to the previous
		// OnTriggeringCLsCompleted() but the deletion was postponed because
		// the PCL were not updated for the CQ vote yet. Also, it could be that
		// the CL author manualy voted the CQ label before cltriggerer executes
		// the task. In any cases, this function should remove the TriggeringCL
		// tasks of all the PCLs that already have the intended CQ vote.
		if pcl := s.PB.GetPCL(tcl.GetClid()); shouldRemoveOp(tcl, pcl) {
			deleted[tcl.GetClid()] = struct{}{}
			return nil // delete
		}
		return tcl // keep as is
	}, nil)
	if !mutated {
		return s, nil, nil
	}
	s = s.cloneShallow()
	s.PB.TriggeringCls = out
	// Must mark affected components for re-evaluation.
	if s.PB.GetRepartitionRequired() {
		// all the components will be retriaged during the repartition process.
		return s, nil, nil
	}
	cs, mutatedComponents := s.PB.COWComponents(func(c *prjpb.Component) *prjpb.Component {
		if c.GetTriageRequired() {
			return c
		}
		for _, id := range c.GetClids() {
			if _, yes := deleted[id]; yes {
				c = c.CloneShallow()
				c.TriageRequired = true
				return c
			}
		}
		return c
	}, nil)
	if mutatedComponents {
		s.PB.Components = cs
	}
	// If there are failed ops, schedule a PurgeCL for the originating PCL.
	se := h.addCLsToPurge(ctx, s, makePurgeCLTasksForFailedTriggerDeps(ctx, s, failed))
	return s, se, nil
}

func shouldRemoveOp(tcl *prjpb.TriggeringCL, pcl *prjpb.PCL) bool {
	if pcl == nil {
		// Delete the Op if the PCL is no longer tracked.
		return true
	}

	switch tr := pcl.GetTriggers().GetCqVoteTrigger(); {
	case tcl.GetTrigger().GetMode() == tr.GetMode():
		return true
	case tr.GetMode() == "":
		// The PCL hasn't been updated yet. Keep the Op in state.
		return false
	default:
		// This is the case where the PCL has a CQ vote, but not the intended
		// value. e.g., the PCL has CQ+1, whereas the intended value is CQ+2.
		//
		// If the vote was made before the Op creation time, consider that
		// the op was suceeded, but the PCL hasn't been updated yet.
		// i.e., keep the Op until the PCL gets updated.
		opCreationTime := tcl.GetDeadline().AsTime().Add(-prjpb.MaxTriggeringCLsDuration)
		if tr.GetTime().AsTime().Before(opCreationTime) {
			return false
		}
	}
	// The PCL has unintended CQ vote that was made at or after the Op creation
	// time. There can be two possible scenarios.
	// (1) The unintended vote was made before cltriggerrer called SetReview().
	// (2) The unintended vote was made after cltriggerer called SetReview().
	//
	// It's hard to firmly distinguish (1) from (2), because it requires
	// the vote time to be compared with the time of SetReview()
	// call, but it'd likely cause racy issues.
	//
	// Note that this can happen only if all the following conditions are true
	// - someone votes CQ+2 on a child CL.
	// - CV triages the deps of the child CL, and creates TriggeringCLTasks
	// for the deps.
	// - The one manually votes CQ+1 on the child CL after the CL scheduled, but
	// before the Op is processed by OnTriggeringsCompleted, which is the time
	// before the Run creation.
	//
	// If the intention was to stop a CQ *Run*, then the manual CQ override will
	// be honored, and it wouldn't reach this point. It's all good.
	// If the intention was to stop CV to create a CQ Run, then the user should
	// remove the CQ vote on the originating CL.
	//
	// Hence, this returns true to remove the TriggeringCLTask. Then,
	// the component triager will reschedule TriggeringCLTask for the dep with
	// a manual vote override.
	return true
}

func makePurgeCLTasksForFailedTriggerDeps(ctx context.Context, s *State, failed []*prjpb.TriggeringCLsCompleted_OpResult) []*prjpb.PurgeCLTask {
	tasks := make(map[int64]*prjpb.PurgeCLTask)
	for _, f := range failed {
		// check the originating PCL status
		origin := f.GetOriginClid()
		opcl := s.PB.GetPCL(origin)
		if opcl == nil {
			// It's rare, but it's possible that the origin CL is no longer
			// tracked. e.g., abandoned. Maybe, that's why the vote failed.
			logging.Infof(ctx, "originating PCL %d is no longer tracked in the component", f.GetOriginClid())
			continue
		}
		// skip creating a new PurgeCLTask, if there exists already.
		if s.PB.GetPurgingCL(origin) != nil {
			continue
		}
		// does the originating CL still have the CQ vote?
		triggerToPurge := opcl.GetTriggers().GetCqVoteTrigger()
		if run.Mode(triggerToPurge.GetMode()) != run.FullRun {
			continue
		}
		if task, ok := tasks[origin]; ok {
			proto.Merge(task.PurgeReasons[0].GetClError().GetTriggerDeps(), f.GetReason())
		} else {
			tasks[origin] = &prjpb.PurgeCLTask{
				PurgeReasons: []*prjpb.PurgeReason{{
					ClError: &changelist.CLError{
						Kind: &changelist.CLError_TriggerDeps_{
							TriggerDeps: f.GetReason(),
						},
					},
					ApplyTo: &prjpb.PurgeReason_Triggers{
						Triggers: &run.Triggers{
							CqVoteTrigger: triggerToPurge,
						},
					},
				}},
				PurgingCl: &prjpb.PurgingCL{Clid: origin},
			}
		}
	}
	if len(tasks) == 0 {
		return nil
	}
	ret := make([]*prjpb.PurgeCLTask, 0, len(tasks))
	for _, t := range tasks {
		ret = append(ret, t)
	}
	return ret
}
