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
	"go.chromium.org/luci/common/trace"

	"go.chromium.org/luci/cv/internal/changelist"
	"go.chromium.org/luci/cv/internal/common"
	"go.chromium.org/luci/cv/internal/configs/prjcfg"
	"go.chromium.org/luci/cv/internal/gerrit/cfgmatcher"
	"go.chromium.org/luci/cv/internal/gerrit/poller"
	"go.chromium.org/luci/cv/internal/prjmanager"
	"go.chromium.org/luci/cv/internal/prjmanager/clpurger"
	"go.chromium.org/luci/cv/internal/prjmanager/itriager"
	"go.chromium.org/luci/cv/internal/prjmanager/prjpb"
)

// State is a state of Project Manager.
//
// The state object must not be re-used except for serializing public state
// after its public methods returned a modified State or an error.
// This allows for efficient evolution of cached helper datastructures which
// would otherwise have to be copied, too.
//
// To illustrate correct and incorrect usages:
//     s0 := &State{...}
//     s1, _, err := s0.Mut1()
//     if err != nil {
//       // ... := s0.Mut2()             // NOT OK, 2nd call on s0
//       return proto.Marshal(s0.PB)     // OK
//     }
//     //  ... := s0.Mut2()              // NOT OK, 2nd call on s0
//     s2, _, err := s1.Mut2()           // OK, s1 may be s0 if Mut1() was noop
//     if err != nil {
//       // return proto.Marshal(s0.PB)  // OK
//       return proto.Marshal(s0.PB)     // OK
//     }
type State struct {
	// PB is the serializable part of State mutated using copy-on-write approach
	// https://en.wikipedia.org/wiki/Copy-on-write
	PB *prjpb.PState

	// LogReasons is append-only accumulation of reasons to record this state for
	// posterity.
	LogReasons []prjpb.LogReason

	// Dependencies used to prepare state transitions.
	CLMutator       *changelist.Mutator
	PMNotifier      *prjmanager.Notifier
	RunNotifier     RunNotifier
	CLPurger        *clpurger.Purger
	CLPoller        *poller.Poller
	ComponentTriage itriager.Triage

	// Helper private fields used during mutations.

	// alreadyCloned is set to true after state is cloned to prevent incorrect
	// usage.
	alreadyCloned bool
	// configGroups are cacehd config groups.
	configGroups []*prjcfg.ConfigGroup
	// cfgMatcher is lazily created, cached, and passed on to State clones.
	cfgMatcher *cfgmatcher.Matcher
	// pclIndex provides O(1) check if PCL exists for a CL.
	//
	// lazily created, see ensurePCLIndex().
	pclIndex pclIndex // CLID => index in PB.Pcls slice.
}

// cloneShallow returns cloned state ready for in-place mutation.
func (s *State) cloneShallow(reasons ...prjpb.LogReason) *State {
	ret := &State{}
	*ret = *s
	if len(reasons) > 0 {
		ret.LogReasons = append(ret.LogReasons, reasons...)
	}

	// Don't use proto.merge to avoid deep copy.
	ret.PB = &prjpb.PState{
		LuciProject:         s.PB.GetLuciProject(),
		Status:              s.PB.GetStatus(),
		ConfigHash:          s.PB.GetConfigHash(),
		ConfigGroupNames:    s.PB.GetConfigGroupNames(),
		Pcls:                s.PB.GetPcls(),
		Components:          s.PB.GetComponents(),
		RepartitionRequired: s.PB.GetRepartitionRequired(),
		CreatedPruns:        s.PB.GetCreatedPruns(),
		NextEvalTime:        s.PB.GetNextEvalTime(),
		PurgingCls:          s.PB.GetPurgingCls(),
	}

	s.alreadyCloned = true
	return ret
}

func (s *State) ensureNotYetCloned() {
	if s.alreadyCloned {
		panic("Incorrect use. This State object has already been cloned. See State doc")
	}
}

type RunNotifier interface {
	Start(ctx context.Context, id common.RunID) error
	PokeNow(ctx context.Context, id common.RunID) error
	Cancel(ctx context.Context, id common.RunID) error
	UpdateConfig(ctx context.Context, id common.RunID, hash string, eversion int64) error
}

// Handler handles state transitions of a project.
type Handler struct {
	CLMutator       *changelist.Mutator
	PMNotifier      *prjmanager.Notifier
	RunNotifier     RunNotifier
	CLPurger        *clpurger.Purger
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
			if err := s.CLPoller.Poke(ctx, s.PB.GetLuciProject()); err != nil {
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
	if err := s.pokeRuns(ctx); err != nil {
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

	ctx, span := trace.StartSpan(ctx, "go.chromium.org/luci/cv/internal/prjmanager/impl/state/OnRunsCreated")
	defer func() { span.End(err) }()

	// Check if PM is already aware of these Runs.
	remaining := created.Set()
	s.PB.IterIncompleteRuns(func(r *prjpb.PRun, _ *prjpb.Component) (stop bool) {
		id := common.RunID(r.GetId())
		if _, ok := remaining[id]; ok {
			delete(remaining, id)
		}
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
		// This should not normally happen, but may under rare conditons.
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
func (h *Handler) OnRunsFinished(ctx context.Context, s *State, finished common.RunIDs) (_ *State, __ SideEffect, err error) {
	s.ensureNotYetCloned()

	ctx, span := trace.StartSpan(ctx, "go.chromium.org/luci/cv/internal/prjmanager/impl/state/OnRunsFinished")
	defer func() { span.End(err) }()

	// This is rarely a noop, so assume state is modified for simplicity.
	s = s.cloneShallow()
	incompleteRunsCount := s.removeFinishedRuns(finished.Set())
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

	ctx, span := trace.StartSpan(ctx, "go.chromium.org/luci/cv/internal/prjmanager/impl/state/OnCLsUpdated")
	defer func() { span.End(err) }()

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

	ctx, span := trace.StartSpan(ctx, "go.chromium.org/luci/cv/internal/prjmanager/impl/state/OnPurgesCompleted")
	defer func() { span.End(err) }()

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

	ctx, span := trace.StartSpan(ctx, "go.chromium.org/luci/cv/internal/prjmanager/impl/state/ExecDeferred")
	defer func() { span.End(err) }()

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
	switch actions, saveForDebug, err := s.triageComponents(ctx); {
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
		sideEffect, err = s.actOnComponents(ctx, actions)
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

// UpgradeIfNecessary upgrades old state to new format if necessary.
//
// Returns the new state, or this state if nothing was changed.
func (s *State) UpgradeIfNecessary() *State {
	if !s.needUpgrade() {
		return s
	}
	s = s.cloneShallow()
	s.upgrade()
	return s
}
