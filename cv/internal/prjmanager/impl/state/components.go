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

package state

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"

	"go.chromium.org/luci/common/clock"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/common/runtime/paniccatcher"
	"go.chromium.org/luci/common/sync/parallel"

	"go.chromium.org/luci/cv/internal/common"
	"go.chromium.org/luci/cv/internal/config"
	"go.chromium.org/luci/cv/internal/migration/migrationcfg"
	"go.chromium.org/luci/cv/internal/prjmanager/impl/state/itriager"
	"go.chromium.org/luci/cv/internal/prjmanager/prjpb"
	"go.chromium.org/luci/cv/internal/prjmanager/runcreator"
)

// earliestDecisionTime returns the earliest decision time of all components.
//
// Returns the same time as time.Time and proto.
//
// Re-uses DecisionTime of one of the components, assuming that components are
// modified copy-on-write.
func earliestDecisionTime(cs []*prjpb.Component) (time.Time, *timestamppb.Timestamp) {
	var ret time.Time
	var retPB *timestamppb.Timestamp
	for _, c := range cs {
		if dt := c.GetDecisionTime(); dt != nil {
			if t := dt.AsTime(); ret.IsZero() || ret.After(t) {
				ret = t
				retPB = dt
			}
		}
	}
	return ret, retPB
}

// cAction is a component action to be taken during current PM mutation.
//
// An action may involve creating one or more new Runs,
// or removing CQ votes from CL(s) which can't form a Run for some reason.
type cAction struct {
	itriager.Result
	componentIndex int

	// runsFailed is modified during actOnComponents.
	runsFailed int32
}

const concurrentComponentProcessing = 16

// triageComponents triages components.
//
// Doesn't modify the state itself.
//
// Returns an action per each component that needs acting upon.
func (s *State) triageComponents(ctx context.Context) ([]*cAction, error) {
	var sup itriager.PMState
	sup, err := s.makeActorSupporter(ctx)
	if err != nil {
		return nil, err
	}

	poolSize := concurrentComponentProcessing
	if n := len(s.PB.GetComponents()); n < poolSize {
		poolSize = n
	}
	now := clock.Now(ctx)

	var mutex sync.Mutex
	var actions []*cAction
	poolErr := parallel.WorkPool(poolSize, func(work chan<- func() error) {
		for i, oldC := range s.PB.GetComponents() {
			i, oldC := i, oldC
			if !needsTriage(oldC, now) {
				continue
			}
			work <- func() error {
				switch res, err := s.triageOneComponent(ctx, oldC, sup); {
				case err != nil:
					// Log error here since only total errs count will be propagated up
					// the stack.
					logging.Errorf(ctx, "%s while processing component: %s", err, protojson.Format(oldC))
					return err
				default:
					mutex.Lock()
					actions = append(actions, &cAction{componentIndex: i, Result: res})
					mutex.Unlock()
					return nil
				}
			}
		}
	})
	switch merrs, ok := poolErr.(errors.MultiError); {
	case poolErr == nil || (ok && len(merrs) == 0):
		return actions, nil
	case !ok:
		panic(fmt.Errorf("Unexpected return from parallel.WorkPool: %s", poolErr))
	case len(actions) > 0:
		// Components are independent, so proceed despite errors on some components
		// since partial progress is better than none.
		logging.Warningf(ctx, "triageComponents: %d errors, but proceeding to act on %d components", len(merrs), len(actions))
		return actions, nil
	default:
		err := common.MostSevereError(merrs)
		return nil, errors.Annotate(err, "failed to triage %d components, keeping the most severe error", len(merrs)).Err()
	}
}

func needsTriage(c *prjpb.Component, now time.Time) bool {
	if c.GetDirty() {
		return true // external event arrived
	}
	next := c.GetDecisionTime()
	if next == nil {
		return false // wait for an external event
	}
	if next.AsTime().After(now) {
		return false // too early
	}
	return true
}

func (s *State) triageOneComponent(ctx context.Context, oldC *prjpb.Component, sup itriager.PMState) (res itriager.Result, err error) {
	defer paniccatcher.Catch(func(p *paniccatcher.Panic) {
		logging.Errorf(ctx, "caught panic %s:\n%s", p.Reason, p.Stack)
		logging.Debugf(ctx, "caught panic on component:\n%s", protojson.Format(oldC))
		logging.Debugf(ctx, "caught panic current state:\n%s", protojson.Format(s.PB))
		err = errors.Reason("caught panic: %s", p.Reason).Err()
	})
	res, err = s.ComponentTriage(ctx, oldC, sup)
	if err != nil {
		return
	}
	if res.NewValue != nil && res.NewValue == oldC {
		panic(fmt.Errorf("New value re-uses prior component object, must use copy-on-write instead"))
	}
	if len(res.CLsToPurge) > 0 {
		s.validatePurgeCLTasks(oldC, res.CLsToPurge)
	}
	return
}

// actOnComponents executes actions on components produced by triageComponents.
//
// Expects the state to be already shallow cloned.
func (s *State) actOnComponents(ctx context.Context, actions []*cAction) (SideEffect, error) {
	// First, create Runs in parallel.
	poolSize := concurrentComponentProcessing
	if l := len(actions); l < poolSize {
		poolSize = l
	}
	runsErr := parallel.WorkPool(poolSize, func(work chan<- func() error) {
		for _, action := range actions {
			for _, rc := range action.RunsToCreate {
				action, rc := action, rc
				c := s.PB.GetComponents()[action.componentIndex]
				work <- func() error {
					err := s.createOneRun(ctx, rc, c)
					if err != nil {
						atomic.AddInt32(&action.runsFailed, 1)
						// Log error here since only total errs count will be propagated up
						// the stack.
						logging.Errorf(ctx, "%s: %s", protojson.Format(c), err)
					}
					return err
				}
			}
		}
	})

	// Keep runsErr for now and try to make progress on actions/components without
	// errors.

	// Shallow-copy the components slice, as one or more components are highly
	// likely to be modified in a loop below.
	s.PB.Components = append(([]*prjpb.Component)(nil), s.PB.GetComponents()...)
	runsCreated, componentsUpdated := 0, 0
	var clsToPurge []*prjpb.PurgeCLTask
	for _, action := range actions {
		if action.runsFailed > 0 {
			continue
		}
		runsCreated += len(action.RunsToCreate)
		clsToPurge = append(clsToPurge, action.CLsToPurge...)
		if action.NewValue != nil {
			s.PB.Components[action.componentIndex] = action.NewValue
			componentsUpdated++
		}
	}
	var sideEffect SideEffect
	if len(clsToPurge) > 0 {
		sideEffect = s.addCLsToPurge(ctx, clsToPurge)
	}
	proceedMsg := fmt.Sprintf("proceeding to save %d components and purge %d CLs", componentsUpdated, len(clsToPurge))

	// Finally, decide the final result.
	switch merrs, ok := runsErr.(errors.MultiError); {
	case runsErr == nil || (ok && len(merrs) == 0):
		logging.Infof(ctx, "actOnComponents: created %d Runs, %s", runsCreated, proceedMsg)
		return sideEffect, nil
	case !ok:
		panic(fmt.Errorf("Unexpected return from parallel.WorkPool"))
	default:
		logging.Warningf(ctx, "actOnComponents: created %d Runs, failed to create %d Runs", runsCreated, len(merrs))
		if componentsUpdated+len(clsToPurge) == 0 {
			err := common.MostSevereError(merrs)
			return nil, errors.Annotate(err, "failed to actOnComponents, most severe error").Err()
		}
		// All actions are independent, so proceed despite the errors since partial
		// progress is better than none.
		logging.Debugf(ctx, "actOnComponents: %s", proceedMsg)
		return sideEffect, nil
	}
}

func (s *State) createOneRun(ctx context.Context, rc *runcreator.Creator, c *prjpb.Component) (err error) {
	defer paniccatcher.Catch(func(p *paniccatcher.Panic) {
		logging.Errorf(ctx, "caught panic while creating a Run %s:\n%s", p.Reason, p.Stack)
		logging.Debugf(ctx, "caught panic: component %s", protojson.Format(c))
		err = errors.Reason("caught panic: %s", p.Reason).Err()
	})

	switch yes, err := migrationcfg.IsCQDUsingMyRuns(ctx, s.PB.GetLuciProject()); {
	case err != nil:
		return err
	case yes || s.PB.GetLuciProject() == "e2e-always-create-runs":
		// Proceed to creating a Run.
	default:
		// This a is temporary safeguard against creation of LOTS of Runs,
		// that won't be finalized.
		// TODO(tandrii): delete this check once RunManager cancels Runs based on
		// user actions and finalizes based on CQD reports.
		logging.Debugf(ctx, "would have created a Run")
		return nil
	}

	_, err = rc.Create(ctx, s.PMNotifier, s.RunNotifier)
	return err
}

// validatePurgeCLTasks verifies correctness of tasks from componentActor.
//
// Modifies given tasks in place.
// Panics in case of problems.
func (s *State) validatePurgeCLTasks(c *prjpb.Component, ts []*prjpb.PurgeCLTask) {
	// First, verify individual tasks have expected fields set.
	m := make(clidsSet, len(ts))
	for _, t := range ts {
		id := t.GetPurgingCl().GetClid()
		switch {
		case id == 0:
			panic(fmt.Errorf("clid must be set"))
		case m.hasI64(id):
			panic(fmt.Errorf("duplicated clid %d", id))
		case t.GetReasons() == nil:
			panic(fmt.Errorf("at least 1 reason must be given"))
		}
		for i, r := range t.GetReasons() {
			if r.GetKind() == nil {
				panic(fmt.Errorf("Reason #%d is nil", i))
			}
		}
		m.addI64(id)
	}
	// Verify only CLs not yet purged are being purged.
	// NOTE: this iterates all CLs currently being purged, but there should be
	// very few such CLs comparing to the total number of tracked CLs.
	for _, p := range s.PB.GetPurgingCls() {
		if m.hasI64(p.GetClid()) {
			panic(fmt.Errorf("can't purge %d CL which is already being purged", p.GetClid()))
		}
	}
	// Verify only CLs from the component are being purged.
	for _, clid := range c.GetClids() {
		m.delI64(clid)
	}
	if len(m) > 0 {
		panic(fmt.Errorf("purging %v CLs outside the component", m))
	}
}

// addCLsToPurge changes PB.PurgingCLs and prepares for atomic creation of TQ
// tasks to do actual purge.
//
// Expects given tasks to be correct (see validatePurgeCLTasks).
func (s *State) addCLsToPurge(ctx context.Context, ts []*prjpb.PurgeCLTask) SideEffect {
	if len(ts) == 0 {
		return nil
	}
	s.populatePurgeCLTasks(ctx, ts)
	purgingCLs := make([]*prjpb.PurgingCL, len(ts))
	for i, t := range ts {
		purgingCLs[i] = t.GetPurgingCl()
	}
	s.PB.PurgingCls, _ = s.PB.COWPurgingCLs(nil, purgingCLs)
	return &TriggerPurgeCLTasks{payloads: ts, clPurger: s.CLPurger}
}

// maxPurgingCLDuration limits the time that a TQ task has to execute
// PurgeCLTask.
const maxPurgingCLDuration = 10 * time.Minute

// populatePurgeCLTasks populates all remaining fields in PurgeCLsTasks created
// by componentActor.
//
// Modifies given tasks in place.
func (s *State) populatePurgeCLTasks(ctx context.Context, ts []*prjpb.PurgeCLTask) {
	deadline := timestamppb.New(clock.Now(ctx).Add(maxPurgingCLDuration))
	opInt := deadline.AsTime().Unix()
	for _, t := range ts {
		id := t.GetPurgingCl().GetClid()
		pcl := s.PB.GetPcls()[s.pclIndex[common.CLID(id)]]
		t.Trigger = pcl.GetTrigger()
		t.LuciProject = s.PB.GetLuciProject()
		t.PurgingCl.Deadline = deadline
		t.PurgingCl.OperationId = fmt.Sprintf("%d-%d", opInt, id)
	}
}

func (s *State) makeActorSupporter(ctx context.Context) (*actorSupporterImpl, error) {
	if s.configGroups == nil {
		meta, err := config.GetHashMeta(ctx, s.PB.GetLuciProject(), s.PB.GetConfigHash())
		if err != nil {
			return nil, err
		}
		if s.configGroups, err = meta.GetConfigGroups(ctx); err != nil {
			return nil, err
		}
	}
	s.ensurePCLIndex()
	purging := make(map[int64]*prjpb.PurgingCL, len(s.PB.GetPurgingCls()))
	for _, p := range s.PB.GetPurgingCls() {
		purging[p.GetClid()] = p
	}
	return &actorSupporterImpl{
		pcls:         s.PB.GetPcls(),
		pclIndex:     s.pclIndex,
		purging:      purging,
		configGroups: s.configGroups,
	}, nil
}

type actorSupporterImpl struct {
	pcls         []*prjpb.PCL
	pclIndex     map[common.CLID]int
	purging      map[int64]*prjpb.PurgingCL
	configGroups []*config.ConfigGroup
}

func (a *actorSupporterImpl) PCL(clid int64) *prjpb.PCL {
	i, ok := a.pclIndex[common.CLID(clid)]
	if !ok {
		return nil
	}
	return a.pcls[i]
}

func (a *actorSupporterImpl) PurgingCL(clid int64) *prjpb.PurgingCL {
	return a.purging[clid]
}

func (a *actorSupporterImpl) ConfigGroup(index int32) *config.ConfigGroup {
	return a.configGroups[index]
}
