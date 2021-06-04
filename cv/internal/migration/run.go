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

package migration

import (
	"context"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"

	"go.chromium.org/luci/common/clock"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	gerritpb "go.chromium.org/luci/common/proto/gerrit"
	"go.chromium.org/luci/common/retry/transient"
	"go.chromium.org/luci/common/sync/parallel"
	"go.chromium.org/luci/gae/service/datastore"

	cvbqpb "go.chromium.org/luci/cv/api/bigquery/v1"
	migrationpb "go.chromium.org/luci/cv/api/migration"
	"go.chromium.org/luci/cv/internal/changelist"
	"go.chromium.org/luci/cv/internal/common"
	"go.chromium.org/luci/cv/internal/gerrit/trigger"
	"go.chromium.org/luci/cv/internal/run"
)

func fetchActiveRuns(ctx context.Context, project string) ([]*migrationpb.ActiveRun, error) {
	runs, err := fetchRunsWithStatus(ctx, project, run.Status_RUNNING)
	switch {
	case err != nil:
		return nil, err
	case len(runs) == 0:
		return nil, nil
	}
	// Remove runs with corresponding VerifiedCQDRun entities or those being
	// cancelled.
	runs, err = pruneInactiveRuns(ctx, runs)
	switch {
	case err != nil:
		return nil, err
	case len(runs) == 0:
		return nil, nil
	}

	// Load all RunCLs and populate ActiveRuns concurrently, but leave FyiDeps
	// computation for later, since these can't be filled from RunCLs anyway.
	ret := make([]*migrationpb.ActiveRun, len(runs))
	err = parallel.WorkPool(min(len(runs), 32), func(workCh chan<- func() error) {
		for i, r := range runs {
			i, r := i, r
			workCh <- func() error {
				var err error
				ret[i], err = makeActiveRun(ctx, r)
				return err
			}
		}
	})
	if err != nil {
		return nil, common.MostSevereError(err)
	}

	// Finally, process all FyiDeps at once.
	cls := map[common.CLID]*changelist.CL{}
	for _, r := range ret {
		for _, d := range r.GetFyiDeps() {
			clid := common.CLID(d.GetId())
			if _, exists := cls[clid]; !exists {
				cls[clid] = &changelist.CL{ID: clid}
			}
		}
	}
	if len(cls) == 0 {
		return ret, nil
	}
	if _, err := changelist.LoadCLsMap(ctx, cls); err != nil {
		return nil, err
	}
	for _, r := range ret {
		for _, d := range r.GetFyiDeps() {
			cl := cls[common.CLID(d.GetId())]
			d.Gc = &cvbqpb.GerritChange{
				Host:                       cl.Snapshot.GetGerrit().GetHost(),
				Project:                    cl.Snapshot.GetGerrit().GetInfo().GetProject(),
				Change:                     cl.Snapshot.GetGerrit().GetInfo().GetNumber(),
				Patchset:                   int64(cl.Snapshot.GetPatchset()),
				EarliestEquivalentPatchset: int64(cl.Snapshot.GetMinEquivalentPatchset()),
			}
			d.Files = cl.Snapshot.GetGerrit().GetFiles()
			d.Info = cl.Snapshot.GetGerrit().GetInfo()
		}
	}
	return ret, nil
}

// makeActiveRun makes ActiveRun except for filling FYI deps with details,
// which is done later for all Runs in order to de-dupe common FYI deps.
// This is especially helpful for large CL stacks in single-CL Run mode.
func makeActiveRun(ctx context.Context, r *run.Run) (*migrationpb.ActiveRun, error) {
	runCLs, err := run.LoadRunCLs(ctx, r.ID, r.CLs)
	if err != nil {
		return nil, err
	}
	known := make(map[common.CLID]struct{}, len(r.CLs))
	allDeps := map[common.CLID]struct{}{}
	mcls := make([]*migrationpb.RunCL, len(runCLs))
	for i, cl := range runCLs {
		known[cl.ID] = struct{}{}
		trigger := &migrationpb.RunCL_Trigger{
			Email:     cl.Trigger.GetEmail(),
			Time:      cl.Trigger.GetTime(),
			AccountId: cl.Trigger.GetGerritAccountId(),
		}
		mcl := &migrationpb.RunCL{
			Id: int64(cl.ID),
			Gc: &cvbqpb.GerritChange{
				Host:                       cl.Detail.GetGerrit().GetHost(),
				Project:                    cl.Detail.GetGerrit().GetInfo().GetProject(),
				Change:                     cl.Detail.GetGerrit().GetInfo().GetNumber(),
				Patchset:                   int64(cl.Detail.GetPatchset()),
				EarliestEquivalentPatchset: int64(cl.Detail.GetMinEquivalentPatchset()),
				Mode:                       r.Mode.BQAttemptMode(),
			},
			Files:   cl.Detail.GetGerrit().GetFiles(),
			Info:    cl.Detail.GetGerrit().GetInfo(),
			Trigger: trigger,
			Deps:    make([]*migrationpb.RunCL_Dep, len(cl.Detail.GetDeps())),
		}
		for i, dep := range cl.Detail.GetDeps() {
			allDeps[common.CLID(dep.GetClid())] = struct{}{}
			mcl.Deps[i] = &migrationpb.RunCL_Dep{
				Id: dep.GetClid(),
			}
			if dep.GetKind() == changelist.DepKind_HARD {
				mcl.Deps[i].Hard = true
			}
		}
		mcls[i] = mcl
	}
	var fyiDeps []*migrationpb.RunCL
	for clid := range allDeps {
		if _, yes := known[clid]; yes {
			continue
		}
		fyiDeps = append(fyiDeps, &migrationpb.RunCL{Id: int64(clid)})
	}
	return &migrationpb.ActiveRun{
		Id:      string(r.ID),
		Cls:     mcls,
		FyiDeps: fyiDeps,
	}, nil
}

func fetchExcludedCLs(ctx context.Context, project string) ([]*cvbqpb.GerritChange, error) {
	// Find all Runs with active status.
	// This requires multiple queries per each Run status, since each query must
	// use inequality on LUCI project (part of Run ID).
	var m sync.Mutex
	eg, egCtx := errgroup.WithContext(ctx)
	var activeRuns []*run.Run
	for v := range run.Status_name {
		status := run.Status(v)
		if status < run.Status_RUNNING || status >= run.Status_ENDED_MASK {
			continue
		}
		eg.Go(func() error {
			switch runs, err := fetchRunsWithStatus(egCtx, project, status); {
			case err != nil:
				return err
			case len(runs) > 0:
				m.Lock()
				activeRuns = append(activeRuns, runs...)
				m.Unlock()
			}
			return nil
		})
	}
	if err := eg.Wait(); err != nil {
		return nil, err
	}
	if len(activeRuns) == 0 {
		return nil, nil
	}

	// Check which Runs have corresponding VerifiedCQDRun entities,
	// and record CLID(s) of these Runs.
	verified := make([]*datastore.Key, len(activeRuns))
	for i, r := range activeRuns {
		verified[i] = datastore.MakeKey(ctx, "migration.VerifiedCQDRun", string(r.ID))
	}
	exists, err := datastore.Exists(ctx, verified)
	if err != nil {
		return nil, errors.Annotate(err, "failed to check existence of VerifiedCQDRuns").Tag(transient.Tag).Err()
	}
	var excludedIDs common.CLIDs
	for i, r := range activeRuns {
		if !exists.Get(0, i) {
			continue
		}
		excludedIDs = append(excludedIDs, r.CLs...)
	}

	// Convert CLIDs to Gerrit host/change for CQDaemon.
	excludedIDs.Dedupe()
	cls, err := changelist.LoadCLs(ctx, excludedIDs)
	if err != nil {
		return nil, err
	}
	ret := make([]*cvbqpb.GerritChange, len(cls))
	for i, cl := range cls {
		ret[i] = &cvbqpb.GerritChange{
			Host:   cl.Snapshot.GetGerrit().GetHost(),
			Change: cl.Snapshot.GetGerrit().Info.GetNumber(),
		}
	}
	return ret, nil
}

func fetchRunsWithStatus(ctx context.Context, project string, status run.Status) ([]*run.Run, error) {
	var runs []*run.Run
	q := run.NewQueryWithLUCIProject(ctx, project).Eq("Status", status)
	if err := datastore.GetAll(ctx, q, &runs); err != nil {
		return nil, errors.Annotate(err, "failed to fetch Run entities").Tag(transient.Tag).Err()
	}
	return runs, nil
}

// fetchAttempt loads Run from Datastore given its CQD attempt key hash.
//
// Returns nil, nil if such Run doesn't exist.
func fetchAttempt(ctx context.Context, key string) (*run.Run, error) {
	q := datastore.NewQuery(run.RunKind).Eq("CQDAttemptKey", key)
	var out []*run.Run
	if err := datastore.GetAll(ctx, q, &out); err != nil {
		return nil, errors.Annotate(err, "failed to fetch Run with CQDAttemptKeyHash=%q", key).Tag(transient.Tag).Err()
	}
	switch l := len(out); l {
	case 0:
		return nil, nil
	case 1:
		return out[0], nil
	default:
		sb := strings.Builder{}
		for _, r := range out {
			sb.WriteRune(' ')
			sb.WriteString(string(r.ID))
		}
		logging.Errorf(ctx, "BUG: found %d Runs with CQDAttemptKeyHash=%q: [%s]", l, key, sb.String())
		// To unblock CQDaemon, choose the latest Run, which given ID generation
		// scheme must be the first in the output.
		return out[0], nil
	}
}

// fetchRun loads Run from Datastore by ID if given, falling back to CQD attempt
// key hash otherwise.
//
// Returns nil, nil if such Run doesn't exist.
func fetchRun(ctx context.Context, id common.RunID, attemptKey string) (*run.Run, error) {
	if id == "" {
		return fetchAttempt(ctx, attemptKey)
	}
	res := &run.Run{ID: id}
	switch err := datastore.Get(ctx, res); {
	case err == datastore.ErrNoSuchEntity:
		return nil, nil
	case err != nil:
		return nil, errors.Annotate(err, "failed to fetch Run entity").Tag(transient.Tag).Err()
	default:
		return res, nil
	}
}

// VerifiedCQDRun is the Run reported by CQDaemon after verification completes.
type VerifiedCQDRun struct {
	_kind string `gae:"$kind,migration.VerifiedCQDRun"`
	// ID is ID of this Run in CV.
	ID common.RunID `gae:"$id"`
	// Payload is what CQDaemon has reported.
	Payload *migrationpb.ReportVerifiedRunRequest
	// RecordTime is when this entity was inserted.
	UpdateTime time.Time `gae:",noindex"`
}

func saveVerifiedCQDRun(ctx context.Context, req *migrationpb.ReportVerifiedRunRequest, notify func(context.Context) error) error {
	runID := common.RunID(req.GetRun().GetId())
	req.GetRun().Id = "" // will be stored as VerifiedCQDRun.ID

	try := 0
	err := datastore.RunInTransaction(ctx, func(ctx context.Context) error {
		try++
		v := VerifiedCQDRun{ID: runID}
		switch err := datastore.Get(ctx, &v); {
		case err == datastore.ErrNoSuchEntity:
			// expected.
		case err != nil:
			return err
		default:
			// Do not overwrite existing one, since CV must be already finalizing it.
			logging.Warningf(ctx, "VerifiedCQDRun %q in %d-th try: already exists", runID, try)
			return nil
		}
		v = VerifiedCQDRun{
			ID:         runID,
			UpdateTime: datastore.RoundTime(clock.Now(ctx).UTC()),
			Payload:    req,
		}
		if err := datastore.Put(ctx, &v); err != nil {
			return err
		}
		return notify(ctx)
	}, nil)
	return errors.Annotate(err, "failed to record VerifiedCQDRun %q after %d tries", runID, try).Tag(transient.Tag).Err()
}

// pruneInactiveRuns removes Runs for which VerifiedCQDRun have already been
// written or iff the run is about to be cancelled.
//
// Modifies the Runs slice in place, but also returns it for readability.
func pruneInactiveRuns(ctx context.Context, in []*run.Run) ([]*run.Run, error) {
	out := in[:0]
	keys := make([]*datastore.Key, len(in))
	for i, r := range in {
		keys[i] = datastore.MakeKey(ctx, "migration.VerifiedCQDRun", string(r.ID))
	}
	exists, err := datastore.Exists(ctx, keys)
	if err != nil {
		return nil, errors.Annotate(err, "failed to check VerifiedCQDRun existence").Tag(transient.Tag).Err()
	}
	for i, r := range in {
		if !exists.Get(0, i) && r.DelayCancelUntil.IsZero() {
			out = append(out, r)
		}
	}
	return out, nil
}

// FinishedCQDRun contains info about a finished Run reported by the CQDaemon.
//
// To be removed after the first milestone is reached.
type FinishedCQDRun struct {
	_kind string `gae:"$kind,migration.FinishedCQDRun"`
	// AttemptKey is the CQD ID of the Run.
	//
	// Once CV starts creating Runs, the CV's Run for the same Run will contain
	// the AttemptKey as a substring.
	AttemptKey string `gae:"$id"`
	// RunID may be set if CQD's Attempt has corresponding CV Run at the time of
	// saving of this entity.
	//
	// Although the CV RunID is also stored in the Payload, a separate field is
	// necessary for Datastore indexing.
	RunID common.RunID
	// RecordTime is when this entity was inserted.
	UpdateTime time.Time `gae:",noindex"`
	// Everything that CQD has sent.
	Payload *migrationpb.ReportedRun
}

func saveFinishedCQDRun(ctx context.Context, mr *migrationpb.ReportedRun, notify func(context.Context) error) error {
	key := mr.GetAttempt().GetKey()
	try := 0
	err := datastore.RunInTransaction(ctx, func(ctx context.Context) error {
		try++
		f := FinishedCQDRun{AttemptKey: key}
		switch err := datastore.Get(ctx, &f); {
		case err == datastore.ErrNoSuchEntity:
			// expected.
		case err != nil:
			return err
		default:
			logging.Warningf(ctx, "Overwriting FinishedCQDRun %q in %d-th try", key, try)
		}
		f = FinishedCQDRun{
			AttemptKey: key,
			UpdateTime: datastore.RoundTime(clock.Now(ctx).UTC()),
			Payload:    mr,
		}
		if id := f.Payload.GetId(); id != "" {
			f.RunID = common.RunID(id)
		}
		if err := datastore.Put(ctx, &f); err != nil {
			return err
		}
		return notify(ctx)
	}, nil)
	return errors.Annotate(err, "failed to record FinishedCQDRun %q after %d tries", key, try).Tag(transient.Tag).Err()
}

// LoadFinishedCQDRun loads from Datastore a FinishedCQDRun.
//
// Expects exactly 1 FinishedCQDRun to exist.
func LoadFinishedCQDRun(ctx context.Context, rid common.RunID) (*FinishedCQDRun, error) {
	var frs []*FinishedCQDRun
	q := datastore.NewQuery("migration.FinishedCQDRun").Eq("RunID", rid).Limit(2)
	switch err := datastore.GetAll(ctx, q, &frs); {
	case err != nil:
		return nil, errors.Annotate(err, "failed to fetch FinishedCQDRun").Tag(transient.Tag).Err()
	case len(frs) == 1:
		return frs[0], nil

		// 2 checks below are defensive coding: neither is supposed to happen in
		// practice unless the way Attempt key is generated differs between CV and
		// CQDaemon.
	case len(frs) > 1:
		return nil, errors.Reason(">1 FinishedCQDRun for Run %s", rid).Err()
	default:
		return nil, errors.Reason("no FinishedCQDRun for Run %s", rid).Err()
	}
}

// makeGerritSetReviewRequest creates request to post a message to Gerrit at
// CQDaemon's request.
func makeGerritSetReviewRequest(r *run.Run, ci *gerritpb.ChangeInfo, msg, curRevision string, sendEmail bool) *gerritpb.SetReviewRequest {
	req := &gerritpb.SetReviewRequest{
		Number:     ci.GetNumber(),
		Project:    ci.GetProject(),
		RevisionId: curRevision,
		Message:    msg,
		Tag:        "autogenerated:cq",
		Notify:     gerritpb.Notify_NOTIFY_OWNER, // by default
	}
	switch {
	case !sendEmail:
		req.Notify = gerritpb.Notify_NOTIFY_NONE
	case r.Mode == run.FullRun:
		req.Notify = gerritpb.Notify_NOTIFY_OWNER_REVIEWERS
		fallthrough
	default:
		// notify CQ label voters, too.
		// This doesn't take into account additional labels, but it's good enough
		// during the migration.
		var accounts []int64
		for _, vote := range ci.GetLabels()[trigger.CQLabelName].GetAll() {
			if vote.GetValue() != 0 {
				accounts = append(accounts, vote.GetUser().GetAccountId())
			}
		}
		req.NotifyDetails = &gerritpb.NotifyDetails{
			Recipients: []*gerritpb.NotifyDetails_Recipient{
				{
					RecipientType: gerritpb.NotifyDetails_RECIPIENT_TYPE_TO,
					Info: &gerritpb.NotifyDetails_Info{
						Accounts: accounts,
					},
				},
			},
		}
	}
	return req
}

func min(i, j int) int {
	if i < j {
		return i
	}
	return j
}
