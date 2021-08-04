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

// Package runcreator creates new Runs.
package runcreator

import (
	"context"
	"crypto/sha256"
	"fmt"
	"sort"
	"strconv"
	"time"

	"go.chromium.org/luci/auth/identity"
	"go.chromium.org/luci/common/clock"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/common/retry/transient"
	"go.chromium.org/luci/common/trace"
	"go.chromium.org/luci/gae/service/datastore"

	commonpb "go.chromium.org/luci/cv/api/common/v1"
	"go.chromium.org/luci/cv/internal/changelist"
	"go.chromium.org/luci/cv/internal/common"
	"go.chromium.org/luci/cv/internal/configs/prjcfg"
	"go.chromium.org/luci/cv/internal/prjmanager"
	"go.chromium.org/luci/cv/internal/prjmanager/prjpb"
	"go.chromium.org/luci/cv/internal/run"
)

// Creator creates a new Run.
//
// If Expected<...> parameters differ from what's read from Datastore during
// transaction, the creation is aborted with error tagged with StateChangedTag.
// See Creator.Create doc.
type Creator struct {
	// LUCIProject. Required.
	LUCIProject string
	// ConfigGroupID for the Run. Required.
	//
	// TODO(tandrii): support triggering via API calls by validating Run creation
	// request against the latest config *before* transaction.
	ConfigGroupID prjcfg.ConfigGroupID
	// InputCLs will reference the newly created Run via their IncompleteRuns
	// field, and Run's RunCL entities will reference these InputCLs back.
	// Required.
	InputCLs []CL
	// Mode is the Run's mode. Required.
	Mode run.Mode
	// Owner is the Run Owner. Required.
	Owner identity.Identity
	// Options is metadata of the Run. Required.
	Options *run.Options
	// ExpectedIncompleteRunIDs are a sorted slice of Run IDs which may be associated with
	// CLs.
	//
	// If CLs.IncompleteRuns reference any other Run ID, the creation will be
	// aborted and error tagged with StateChangedTag.
	//
	// Nil by default, which doesn't permit any incomplete Run.
	ExpectedIncompleteRunIDs common.RunIDs
	// OperationID is an arbitrary string uniquely identifying this creation
	// attempt. Required.
	//
	// TODO(tandrii): for CV API, record this ID in a separate entity index by
	// this ID for full idempotency of CV API.
	OperationID string
	// CreateTime is rounded by datastore.RoundTime and used as Run.CreateTime as
	// well as RunID generation.
	//
	// Optional. By default, it's the current time.
	CreateTime time.Time

	// Internal state: pre-computed once before transaction starts.

	runIDBuilder struct {
		version int
		digest  []byte
	}

	// Internal state: computed during transaction.
	// Since transactions are retried, these can be re-set multiple times.

	// dsBatcher flattens multiple Datastore Get/Put calls into a single call.
	dsBatcher dsGetBatcher
	// runID is computed runID at the time beginning of a transaction, since RunID
	// depends on time.
	runID common.RunID
	// cls are the read & updated CLs.
	cls []*changelist.CL
	// run stores the resulting Run, eventually returned by Create().
	run *run.Run
}

// CL is a helper struct for per-CL input for run creation.
type CL struct {
	ID               common.CLID
	ExpectedEVersion int
	TriggerInfo      *run.Trigger
	Snapshot         *changelist.Snapshot // Only needed for compat with CQDaemon.
}

// pmNotifier encapsulates interaction with Project Manager.
//
// In production, implemented by prjmanager.Notifier.
type pmNotifier interface {
	NotifyRunCreated(ctx context.Context, runID common.RunID) error
	NotifyCLsUpdated(ctx context.Context, luciProject string, cls *changelist.CLUpdatedEvents) error
}

// rmNotifier encapsulates interaction with Run Manager.
//
// In production, implemented by run.Notifier.
type rmNotifier interface {
	Start(ctx context.Context, runID common.RunID) error
}

// StateChangedTag is an error tag used to indicate that state read from
// Datastore differs from the expected state.
var StateChangedTag = errors.BoolTag{Key: errors.NewTagKey("Run Creator: state changed")}

// Create atomically creates a new Run.
//
// Returns the newly created Run.
//
// Returns 3 kinds of errors:
//
//   * tagged with transient.Tag, meaning it's reasonable to retry.
//     Typically due to contention on simultaneously updated CL entity or
//     transient Datastore Get/Put problem.
//
//   * tagged with StateChangedTag.
//     This means the desired Run might still be possible to create,
//     but it must be re-validated against updated CLs or Project config
//     version.
//
//   * all other errors are non retryable and typically indicate a bug or severe
//     misconfiguration. For example, lack of ProjectStateOffload entity.
func (rb *Creator) Create(ctx context.Context, clMutator *changelist.Mutator, pm pmNotifier, rm rmNotifier) (ret *run.Run, err error) {
	ctx, span := trace.StartSpan(ctx, "go.chromium.org/luci/cv/internal/prjmanager/run/Create")
	defer func() { span.End(err) }()

	now := clock.Now(ctx)
	rb.prepare(now)
	var innerErr error
	err = datastore.RunInTransaction(ctx, func(ctx context.Context) error {
		ret, innerErr = rb.createTransactionally(ctx, clMutator, pm, rm)
		return innerErr
	}, nil)
	switch {
	case innerErr != nil:
		return nil, innerErr
	case err != nil:
		return nil, errors.Annotate(err, "failed to create run").Tag(transient.Tag).Err()
	default:
		logging.Debugf(ctx, "Created Run %q with %d CLs", ret.ID, len(ret.CLs))
		return ret, nil
	}
}

// ExpectedRunID returns RunID of the Run to be created.
//
// Only works if non-Zero CreateTime is specified. Otherwise, panics.
func (rb *Creator) ExpectedRunID() common.RunID {
	if rb.CreateTime.IsZero() {
		panic("CreateTime must be given")
	}
	rb.prepare(time.Time{})
	return rb.runID
}

func (rb *Creator) prepare(now time.Time) {
	switch {
	case rb.LUCIProject == "":
		panic("LUCIProject is required")
	case rb.ConfigGroupID == "":
		panic("ConfigGroupID is required")
	case len(rb.InputCLs) == 0:
		panic("At least 1 CL is required")
	case rb.Mode == "":
		panic("Mode is required")
	case rb.Owner == "":
		panic("Owner is required")
	case rb.Options == nil:
		panic("Options is required")
	case rb.OperationID == "":
		panic("OperationID is required")
	}
	for _, cl := range rb.InputCLs {
		if cl.ExpectedEVersion == 0 || cl.ID == 0 || cl.Snapshot == nil || cl.TriggerInfo == nil {
			panic("Each CL field is required")
		}
	}
	if rb.CreateTime.IsZero() {
		rb.CreateTime = now
	}
	rb.CreateTime = datastore.RoundTime(rb.CreateTime).UTC()
	rb.computeCLsDigest()
	rb.computeRunID()
}

func (rb *Creator) createTransactionally(ctx context.Context, clMutator *changelist.Mutator, pm pmNotifier, rm rmNotifier) (*run.Run, error) {
	switch err := rb.load(ctx); {
	case err == errAlreadyCreated:
		return rb.run, nil
	case err != nil:
		return nil, err
	}
	if err := rb.save(ctx, clMutator, pm, rm); err != nil {
		return nil, err
	}
	return rb.run, nil
}

// load reads latest state from Datastore and verifies creation can proceed.
func (rb *Creator) load(ctx context.Context) error {
	rb.dsBatcher.reset()
	rb.checkRunExists(ctx)
	rb.checkProjectState(ctx)
	rb.checkCLsUnchanged(ctx)
	return rb.dsBatcher.get(ctx)
}

// errAlreadyCreated is an internal error to signal that save is not necessary,
// since the intended Run is already created.
var errAlreadyCreated = errors.New("already created")

// checkRunExists checks whether a Run already exists in datastore, and whether
// it was created by us.
func (rb *Creator) checkRunExists(ctx context.Context) {
	rb.run = &run.Run{ID: rb.runID}
	rb.dsBatcher.register(rb.run, func(err error) error {
		switch {
		case err == datastore.ErrNoSuchEntity:
			// Run doesn't exist, which is expected.
			return nil
		case err != nil:
			return errors.Annotate(err, "failed to load Run entity").Tag(transient.Tag).Err()

		case rb.run.CreationOperationID == rb.OperationID:
			// This is quite likely if prior transaction attempt actually succeeds on
			// Datastore side, but CV failed to receive an ACK and is now retrying the
			// transaction body.
			logging.Debugf(ctx, "Run(ID:%s) already created by us", rb.runID)
			return errAlreadyCreated
		default:
			return errors.Reason("Run %q already created with OperationID %q", rb.runID, rb.run.CreationOperationID).Err()
		}
	})
}

// checkProjectState checks if the project is enabled and uses the expected
// ConfigHash.
func (rb *Creator) checkProjectState(ctx context.Context) {
	ps := &prjmanager.ProjectStateOffload{
		Project: datastore.MakeKey(ctx, prjmanager.ProjectKind, rb.LUCIProject),
	}
	rb.dsBatcher.register(ps, func(err error) error {
		switch {
		case err == datastore.ErrNoSuchEntity:
			return errors.Annotate(err, "failed to load ProjectStateOffload").Err()
		case err != nil:
			return errors.Annotate(err, "failed to load ProjectStateOffload").Tag(transient.Tag).Err()
		case ps.Status != prjpb.Status_STARTED:
			return errors.Reason("project %q status is %s, expected STARTED", rb.LUCIProject, ps.Status.String()).Tag(StateChangedTag).Err()
		case ps.ConfigHash != rb.ConfigGroupID.Hash():
			return errors.Reason("project config is %s, expected %s", ps.ConfigHash, rb.ConfigGroupID.Hash()).Tag(StateChangedTag).Err()
		default:
			return nil
		}
	})
}

// checkCLsUnchanged sets `.cls` with the latest Datastore value and verifies
// that their EVersion matches what's expected.
func (rb *Creator) checkCLsUnchanged(ctx context.Context) {
	rb.cls = make([]*changelist.CL, len(rb.InputCLs))
	for i, inputCL := range rb.InputCLs {
		rb.cls[i] = &changelist.CL{ID: inputCL.ID}
		i := i
		id := inputCL.ID
		expected := inputCL.ExpectedEVersion
		rb.dsBatcher.register(rb.cls[i], func(err error) error {
			switch {
			case err == datastore.ErrNoSuchEntity:
				return errors.Annotate(err, "CL %d doesn't exist", id).Err()
			case err != nil:
				return errors.Annotate(err, "failed to load CL %d", id).Tag(transient.Tag).Err()
			case rb.cls[i].EVersion != expected:
				return errors.Reason("CL %d changed since EVersion %d", id, expected).Tag(StateChangedTag).Err()
			}
			diff := rb.cls[i].IncompleteRuns.DifferenceSorted(rb.ExpectedIncompleteRunIDs)
			if len(diff) > 0 {
				return errors.Reason("CL %d has unexpected incomplete runs: %v", id, diff).Tag(StateChangedTag).Err()
			}
			return nil
		})
	}
}

// save saves all modified and created Datastore entities.
//
// It may be retried multiple times on failure.
func (rb *Creator) save(ctx context.Context, clMutator *changelist.Mutator, pm pmNotifier, rm rmNotifier) error {
	// NOTE: within the Datastore transaction,
	//  * NotifyRunCreated && run.Start put a Reminder entity in Datastore (see
	//    server/tq/txn).
	//  * Cloud Datastore client buffers all Puts in RAM, and sends all at once to
	//    Datastore server at transaction's Commit().
	// Therefore, there is no advantage in parallelizing calls below.

	// Keep .CreateTime and .UpdateTime entities the same across all saved
	// entities (except possibly Run, whose createTime can be overridden). Do
	// pre-emptive rounding before Datastore layer does it, such that rb.run
	// entity has the exact same fields' values as if entity was read from the
	// Datastore.
	now := datastore.RoundTime(clock.Now(ctx).UTC())
	if err := rb.saveRun(ctx, now); err != nil {
		return err
	}
	for i := range rb.InputCLs {
		if err := rb.saveRunCL(ctx, i); err != nil {
			return err
		}
	}
	clMutations := make([]*changelist.CLMutation, len(rb.cls))
	for i, cl := range rb.cls {
		clMutations[i] = clMutator.Adopt(ctx, rb.LUCIProject, cl)
		clMutations[i].CL.IncompleteRuns.InsertSorted(rb.runID)
	}
	if _, err := clMutator.FinalizeBatch(ctx, clMutations); err != nil {
		return err
	}
	if err := pm.NotifyRunCreated(ctx, rb.runID); err != nil {
		return err
	}
	return rm.Start(ctx, rb.runID)
}

func (rb *Creator) saveRun(ctx context.Context, now time.Time) error {
	ids := make(common.CLIDs, len(rb.InputCLs))
	for i, cl := range rb.InputCLs {
		ids[i] = cl.ID
	}
	rb.run = &run.Run{
		ID:                  rb.run.ID,
		CQDAttemptKey:       rb.run.ID.AttemptKey(),
		EVersion:            1,
		CreationOperationID: rb.OperationID,
		CreateTime:          rb.CreateTime,
		UpdateTime:          now,
		// EndTime & StartTime intentionally left unset.

		CLs:           ids,
		ConfigGroupID: rb.ConfigGroupID,
		Mode:          rb.Mode,
		Status:        commonpb.Run_PENDING,
		Owner:         rb.Owner,
		Options:       rb.Options,
	}
	if err := datastore.Put(ctx, rb.run); err != nil {
		return errors.Annotate(err, "failed to save Run").Tag(transient.Tag).Err()
	}
	return nil
}

func (rb *Creator) saveRunCL(ctx context.Context, index int) error {
	inputCL := rb.InputCLs[index]
	entity := &run.RunCL{
		Run:        datastore.MakeKey(ctx, run.RunKind, string(rb.run.ID)),
		ID:         inputCL.ID,
		IndexedID:  inputCL.ID,
		ExternalID: rb.cls[index].ExternalID,
		Trigger:    inputCL.TriggerInfo,
		Detail:     rb.cls[index].Snapshot,
	}
	if err := datastore.Put(ctx, entity); err != nil {
		return errors.Annotate(err, "failed to save RunCL %d", inputCL.ID).Tag(transient.Tag).Err()
	}
	return nil
}

// computeCLsDigest populates `.runIDBuilder` for use by computeRunID.
func (rb *Creator) computeCLsDigest() {
	// The first version uses CQDaemon's `attempt_key_hash` aimed for
	// ease of comparison and log grepping during migration. However, this
	// assumes Gerrit CLs and requires having changelist.Snapshot pre-loaded.
	// TODO(tandrii): after migration is over, change to hash CLIDs instead, since
	// it's good enough for the purpose of avoiding spurious collision of RunIDs.
	rb.runIDBuilder.version = 1

	cls := append([]CL(nil), rb.InputCLs...) // copy
	sort.Slice(cls, func(i, j int) bool {
		a := cls[i].Snapshot.GetGerrit()
		b := cls[j].Snapshot.GetGerrit()
		switch {
		case a.GetHost() < b.GetHost():
			return true
		case a.GetHost() > b.GetHost():
			return false
		default:
			return a.GetInfo().GetNumber() < b.GetInfo().GetNumber()
		}
	})
	separator := []byte{0}
	h := sha256.New()
	for i, cl := range cls {
		if i > 0 {
			h.Write(separator)
		}
		h.Write([]byte(cl.Snapshot.GetGerrit().GetHost()))
		h.Write(separator)
		h.Write([]byte(strconv.FormatInt(cl.Snapshot.GetGerrit().GetInfo().GetNumber(), 10)))
		h.Write(separator)
		h.Write([]byte(cl.Snapshot.GetGerrit().GetInfo().GetCurrentRevision()))
		h.Write(separator)

		// CQDaemon truncates ns precision to microseconds in gerrit_util.parse_time.
		microsecs := int64(cl.TriggerInfo.GetTime().GetNanos() / 1000)
		secs := cl.TriggerInfo.GetTime().GetSeconds()
		h.Write([]byte(strconv.FormatInt(secs*1000*1000+microsecs, 10)))
		h.Write(separator)

		h.Write([]byte(cl.TriggerInfo.GetMode()))
		h.Write(separator)
		h.Write([]byte(strconv.FormatInt(cl.TriggerInfo.GetGerritAccountId(), 10)))
	}
	rb.runIDBuilder.digest = h.Sum(nil)[:8]
}

// computeRunID generates and saves new Run ID in `.runID`.
func (rb *Creator) computeRunID() {
	b := &rb.runIDBuilder
	rb.runID = common.MakeRunID(rb.LUCIProject, rb.CreateTime, b.version, b.digest)
}

// dsGetBatcher facilitates processing of many different kind of entities in a
// single Get operation while handling errors in entity-specific code.
type dsGetBatcher struct {
	entities  []interface{}
	callbacks []func(error) error
}

func (d *dsGetBatcher) reset() {
	d.entities = d.entities[:0]
	d.callbacks = d.callbacks[:0]
}

// register registers an entity for future get and a callback to handle errors.
//
// entity must be a pointer to an entity object.
// callback is called with entity specific error, possibly nil.
// callback must not be nil.
func (d *dsGetBatcher) register(entity interface{}, callback func(error) error) {
	d.entities = append(d.entities, entity)
	d.callbacks = append(d.callbacks, callback)
}

// get loads entities from datastore and performs error handling.
//
// Aborts and returns the first non-nil error returned by a callback.
// Otherwise, returns nil.
func (d *dsGetBatcher) get(ctx context.Context) error {
	err := datastore.Get(ctx, d.entities)
	if err == nil {
		return d.execCallbacks(nil)
	}
	switch errs, ok := err.(errors.MultiError); {
	case !ok:
		return errors.Annotate(err, "failed to load entities from Datastore").Tag(transient.Tag).Err()
	case len(errs) != len(d.entities):
		panic(fmt.Errorf("%d errors for %d entities", len(errs), len(d.entities)))
	default:
		return d.execCallbacks(errs)
	}
}

func (d *dsGetBatcher) execCallbacks(errs errors.MultiError) error {
	if errs == nil {
		for _, f := range d.callbacks {
			if err := f(nil); err != nil {
				return err
			}
		}
		return nil
	}

	for i, f := range d.callbacks {
		if err := f(errs[i]); err != nil {
			return err
		}
	}
	return nil
}
