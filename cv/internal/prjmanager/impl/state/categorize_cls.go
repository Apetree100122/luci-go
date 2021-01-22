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

	"go.chromium.org/luci/cv/internal/changelist"
	"go.chromium.org/luci/cv/internal/prjmanager/prjpb"
)

type categorizedCLs struct {
	// active CLs must remain in PCLs.
	//
	// A CL is active iff either:
	//   * it has non-nil .Trigger and is watched by this LUCI project
	//     (see `isActiveStandalonePCL`);
	//   * OR it belongs to an incomplete Run.
	//     NOTE: In this case, CL may be no longer watched by this project or even
	//     be with status=DELETED. Such state may temporary arise due to changes
	//     in project's config.  Eventually Run Manager will cancel the Run,
	//     resulting in removal of the Run from the PM's State and hence removal
	//     of the CL from the active set.
	active clidsSet
	// deps CLs are non-active CLs which should be tracked in PCLs because they
	// are deps of active CLs.
	//
	// Similar to active CLs of incomplete Run, these CLs may not even be watched
	// by this project or be with status=DELETED, but such state should be
	// temporary.
	deps clidsSet
	// unused CLs are CLs in PCLs that are neither active nor deps and should be
	// deleted from PCLs.
	unused clidsSet
	// unloaded are CLs that are either active or deps but not present in PCLs.
	//
	// NOTE: if this is not empty, it means `deps` and `unused` aren't yet final
	// and may be changed after the `unloaded` CLs are loaded.
	unloaded clidsSet
}

// isActiveStandalonePCL returns true if PCL is active on its own.
//
// See categorizedCLs.active spec.
func isActiveStandalonePCL(pcl *prjpb.PCL) bool {
	return pcl.GetStatus() == prjpb.PCL_OK && pcl.GetTrigger() != nil
}

// categorizeCLs computes categorizedCLs based on current State.
//
// The resulting categorizeCLs spans not just PCLs, but also CreatedRuns, since
// newly created Runs may reference CLs not yet tracked in PCLs.
func (s *State) categorizeCLs() *categorizedCLs {
	s.ensurePCLIndex()

	// reduce typing and redundant !=nil check in GetPcls().
	pcls := s.PB.GetPcls()

	res := &categorizedCLs{
		// Allocate the maps guessing initial size:
		//  * most PCLs must be active, with very few being pure deps or unused,
		//  * unloaded come from CreatedRuns, assume 2 CL per Run.
		active:   make(clidsSet, len(pcls)),
		deps:     make(clidsSet, 16),
		unused:   make(clidsSet, 16),
		unloaded: make(clidsSet, len(s.PB.GetCreatedPruns())*2),
	}

	// First, compute all active CLs and if any of them are unloaded.
	for _, r := range s.PB.GetCreatedPruns() {
		for _, id := range r.GetClids() {
			res.active.addI64(id)
			if !s.pclIndex.hasI64(id) {
				res.unloaded.addI64(id)
			}
		}
	}
	for _, c := range s.PB.GetComponents() {
		for _, r := range c.GetPruns() {
			for _, id := range r.GetClids() {
				res.active.addI64(id)
			}
		}
	}
	for _, pcl := range pcls {
		id := pcl.GetClid()
		if isActiveStandalonePCL(pcl) {
			res.active.addI64(id)
		}
	}

	// Second, compute `deps` and if any of them are unloaded.
	for _, pcl := range pcls {
		if res.active.hasI64(pcl.GetClid()) {
			for _, dep := range pcl.GetDeps() {
				id := dep.GetClid()
				if !res.active.hasI64(id) {
					res.deps.addI64(id)
					if !s.pclIndex.hasI64(id) {
						res.unloaded.addI64(id)
					}
				}
			}
		}
	}
	// Third, compute `unused` as all unreferenced CLs in PCLs.
	for _, pcl := range s.PB.GetPcls() {
		id := pcl.GetClid()
		if !res.active.hasI64(id) && !res.deps.hasI64(id) {
			res.unused.addI64(id)
		}
	}
	return res
}

// loadActiveIntoPCLs ensures PCLs contain all active CLs and modifies
// categorized CLs accordingly.
//
// Doesn't guarantee that all their deps are loaded.
func (s *State) loadActiveIntoPCLs(ctx context.Context, cat *categorizedCLs) error {
	// Consider a long *chain* of dependent CLs each with CQ+1 vote:
	//    A <- B  (B depends on A)
	//    B <- C
	//    ...
	//    Y <- Z
	// Suppose CV first notices just Z, s.t. CL updater notifies PM with
	// OnCLUpdated event of just Z. Upon receiving which, PM will add Z(deps: Y)
	// into PCL, and then calls this function.
	// Even if at this point Datastore contains all {Y..A} CLs, it's unreasonable
	// to load them all in this function because it'd require O(len(chain)) of RPCs to
	// Datastore, whereby each (% last one) detects yet another dep.
	// Thus, no guarantee to load deps is provided.
	//
	// Normally, the next PM invocation to receive remaining {Y..A} OnCLUpdated
	// events, which would allow to load them all in 1 Datastore GetMulti.
	//
	// Also, such CL chains are rare; most frequently CV deals with long CL stacks,
	// where the latest CL depends on most others, e.g. Z{deps: A..Y}. So, loading
	// all already known deps is beneficial.
	// Furthermore, we must load not yet known CLs which are referenced in
	// CreatedRuns. The categorizeCLs() bundles both missing active and deps into
	// unloaded CLs.
	for len(cat.unloaded) == 0 {
		return nil
	}
	if err := s.loadUnloadedCLsOnce(ctx, cat); err != nil {
		return err
	}
	for u := range cat.unloaded {
		if cat.active.has(u) {
			panic(fmt.Errorf("%d CL is not loaded but active", u))
		}
	}
	return nil
}

// loadUnloadedCLsOnce loads `unloaded` CLs from Datastore, updates PCLs and
// categorizedCLs.
//
// If any previously `unloaded` CLs were or turned out to be active,
// then their deps may end up in `unloaded`.
func (s *State) loadUnloadedCLsOnce(ctx context.Context, cat *categorizedCLs) error {
	cls := make([]*changelist.CL, 0, len(cat.unloaded))
	for clid := range cat.unloaded {
		cls = append(cls, &changelist.CL{ID: clid})
	}
	if err := s.evalCLsFromDS(ctx, cls); err != nil {
		return err
	}
	// This is inefficient, since this could have been done only for loaded CLs.
	// Consider adding callback to the evalCLsFromDS.
	for _, pcl := range s.PB.GetPcls() {
		id := pcl.GetClid()
		if !cat.unloaded.hasI64(id) {
			continue
		}
		cat.unloaded.delI64(id)
		if !cat.active.hasI64(id) {
			// CL was a mere dep before, but its details weren't known.
			if !isActiveStandalonePCL(pcl) {
				continue // pcl was and remains just a dep.
			} else {
				// Promote CL to active.
				cat.deps.delI64(id)
				cat.active.addI64(id)
			}
		}
		// Recurse into deps of a just loaded active CLs.
		for _, dep := range pcl.GetDeps() {
			id := dep.GetClid()
			if !cat.active.hasI64(id) && !cat.deps.hasI64(id) {
				cat.deps.addI64(id)
				if cat.unused.hasI64(id) {
					cat.unused.delI64(id)
				} else {
					cat.unloaded.addI64(id)
				}
			}
		}
	}
	return nil
}
