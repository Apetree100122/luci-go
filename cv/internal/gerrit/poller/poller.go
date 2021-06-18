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

// Package implements stateful Gerrit polling.
package poller

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"go.chromium.org/luci/common/clock"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/common/retry/transient"
	"go.chromium.org/luci/common/sync/parallel"
	"go.chromium.org/luci/gae/service/datastore"
	"go.chromium.org/luci/server/tq"

	"go.chromium.org/luci/cv/internal/changelist"
	"go.chromium.org/luci/cv/internal/common"
	"go.chromium.org/luci/cv/internal/configs/prjcfg"
	"go.chromium.org/luci/cv/internal/gerrit/gobmap"
	"go.chromium.org/luci/cv/internal/gerrit/poller/task"
	"go.chromium.org/luci/cv/internal/gerrit/updater"
)

// PM encapsulates interaction with Project Manager by the Gerrit Poller.
type PM interface {
	NotifyCLsUpdated(ctx context.Context, luciProject string, cls []*changelist.CL) error
}

// Poller polls Gerrit to discover new CLs and modifications of the existing
// ones.
type Poller struct {
	tqd       *tq.Dispatcher
	clUpdater *updater.Updater
	pm        PM
}

// New creates a new Poller, registering it in the given TQ dispatcher.
func New(tqd *tq.Dispatcher, clUpdater *updater.Updater, pm PM) *Poller {
	p := &Poller{tqd, clUpdater, pm}
	tqd.RegisterTaskClass(tq.TaskClass{
		ID:        task.ClassID,
		Prototype: &task.PollGerritTask{},
		Queue:     "poll-gerrit",
		Quiet:     true,
		Kind:      tq.NonTransactional,
		Handler: func(ctx context.Context, payload proto.Message) error {
			task := payload.(*task.PollGerritTask)
			ctx = logging.SetField(ctx, "project", task.GetLuciProject())
			err := p.poll(ctx, task.GetLuciProject(), task.GetEta().AsTime())
			return common.TQIfy{
				KnownRetry: []error{errConcurrentStateUpdate},
			}.Error(ctx, err)
		},
	})
	return p
}

// Poke schedules the next poll via task queue.
//
// Under perfect operation, this is redundant, but not harmful.
// Given bugs or imperfect operation, this ensures poller continues operating.
//
// Must not be called inside a datastore transaction.
func (p *Poller) Poke(ctx context.Context, luciProject string) error {
	if datastore.CurrentTransaction(ctx) != nil {
		panic("must be called outside of transaction context")
	}
	return p.schedule(ctx, luciProject, time.Time{})
}

var errConcurrentStateUpdate = errors.New("concurrent change to poller state", transient.Tag)

// poll executes the next poll with the latest known to poller config.
//
// For each discovered CL, enqueues a task for CL updater to refresh CL state.
// Automatically enqueues a new task to perform next poll.
func (p *Poller) poll(ctx context.Context, luciProject string, eta time.Time) error {
	if delay := clock.Now(ctx).Sub(eta); delay > maxAcceptableDelay {
		logging.Warningf(ctx, "poll %s arrived %s late; scheduling next poll instead", eta, delay)
		return p.schedule(ctx, luciProject, time.Time{})
	}
	// TODO(tandrii): avoid concurrent polling of the same project via cheap
	// best-effort locking in Redis.
	meta, err := prjcfg.GetLatestMeta(ctx, luciProject)
	switch {
	case err != nil:
	case (meta.Status == prjcfg.StatusDisabled ||
		meta.Status == prjcfg.StatusNotExists):
		if err := gobmap.Update(ctx, luciProject); err != nil {
			return err
		}
		if err = datastore.Delete(ctx, &State{LuciProject: luciProject}); err != nil {
			return errors.Annotate(err, "failed to disable poller for %q", luciProject).Err()
		}
		return nil
	case meta.Status == prjcfg.StatusEnabled:
		err = p.pollWithConfig(ctx, luciProject, meta)
	default:
		panic(fmt.Errorf("unknown project config status: %d", meta.Status))
	}

	switch {
	case err == nil:
		return p.schedule(ctx, luciProject, eta)
	case clock.Now(ctx).After(eta.Add(pollInterval - time.Second)):
		// Time to finish this task despite error, and trigger a new one.
		err = errors.Annotate(err, "failed to do poll %s for %q", eta, luciProject).Err()
		common.LogError(ctx, err, errConcurrentStateUpdate)
		return p.schedule(ctx, luciProject, eta)
	default:
		return err
	}
}

// pollInterval is an approximate and merely best-effort average interval
// between polls of a single project.
//
// TODO(tandrii): revisit interval and error handling in pollWithConfig once CV
// subscribes to Gerrit PubSub.
const pollInterval = 10 * time.Second

// maxAcceptableDelay prevents polls which arrive too late from doing actual
// polling.
//
// maxAcceptableDelay / pollInterval effectively limits # concurrent polls of
// the same project that may happen due to task retries, delays, and queue
// throttling.
//
// Do not set too low, as this may prevent actual polling from happening at all
// if the poll TQ is overloaded.
const maxAcceptableDelay = 6 * pollInterval

// schedule schedules the future poll.
//
// Optional `after` can be set to the current task's ETA to ensure that next
// poll's task isn't de-duplicated with the current task.
func (p *Poller) schedule(ctx context.Context, luciProject string, after time.Time) error {
	// Desired properties:
	//   * for a single LUCI project, minimize p99 of actually observed poll
	//     intervals.
	//   * keep polling load on Gerrit at `1/pollInterval` per LUCI project;
	//   * avoid bursts of polls on Gerrit, i.e. distribute polls of diff projects
	//     throughout `pollInterval`.
	//
	// So,
	//   * de-duplicate poll tasks to 1 task per LUCI project per pollInterval;
	//   * vary epoch time, from which increments of pollInterval are done, by
	//     LUCI project. See projectOffset().
	if now := clock.Now(ctx); after.IsZero() || now.After(after) {
		after = now
	}
	offset := common.ProjectOffset("gerrit-poller", pollInterval, luciProject)
	offset = offset.Truncate(time.Millisecond) // more readable logs
	eta := after.UTC().Truncate(pollInterval).Add(offset)
	for !eta.After(after) {
		eta = eta.Add(pollInterval)
	}
	task := &tq.Task{
		Title: luciProject,
		Payload: &task.PollGerritTask{
			LuciProject: luciProject,
			Eta:         timestamppb.New(eta),
		},
		ETA:              eta,
		DeduplicationKey: fmt.Sprintf("%s:%d", luciProject, eta.UnixNano()),
	}
	if err := p.tqd.AddTask(ctx, task); err != nil {
		return err
	}
	return nil
}

// State persists poller's State in datastore.
//
// State is exported for exposure via Admin API.
type State struct {
	_kind string `gae:"$kind,GerritPoller"`

	// Project is the name of the LUCI Project for which poller is working.
	LuciProject string `gae:"$id"`
	// UpdateTime is the timestamp when this state was last updated.
	UpdateTime time.Time `gae:",noindex"`
	// EVersion is the latest version number of the state.
	//
	// It increments by 1 every time state is updated either due to new project config
	// being updated OR after each successful poll.
	EVersion int64 `gae:",noindex"`
	// ConfigHash defines which Config version was last worked on.
	ConfigHash string `gae:",noindex"`
	// SubPollers track individual states of sub pollers.
	//
	// Most LUCI projects will have just 1 per Gerrit host,
	// but CV may split the set of watched Gerrit projects (aka Git repos) on the
	// same Gerrit host among several SubPollers.
	SubPollers *SubPollers
}

// pollWithConfig performs the poll and if necessary updates to newest project config.
func (p *Poller) pollWithConfig(ctx context.Context, luciProject string, meta prjcfg.Meta) error {
	stateBefore := State{LuciProject: luciProject}
	switch err := datastore.Get(ctx, &stateBefore); {
	case err != nil && err != datastore.ErrNoSuchEntity:
		return errors.Annotate(err, "failed to get poller state for %q", luciProject).Tag(transient.Tag).Err()
	case err == datastore.ErrNoSuchEntity || stateBefore.ConfigHash != meta.Hash():
		if err = p.updateConfig(ctx, &stateBefore, meta); err != nil {
			return err
		}
	}

	// Use WorkPool to limit concurrency, but keep track of errors per SubPollers
	// ourselves because WorkPool doesn't guarantee specific errors order.
	errs := make(errors.MultiError, len(stateBefore.SubPollers.GetSubPollers()))
	err := parallel.WorkPool(10, func(work chan<- func() error) {
		for i, sp := range stateBefore.SubPollers.GetSubPollers() {
			i, sp := i, sp
			work <- func() error {
				err := p.subpoll(ctx, luciProject, sp)
				errs[i] = errors.Annotate(err, "subpoller %s", sp).Err()
				return nil
			}
		}
	})
	if err != nil {
		panic(err)
	}
	// Save state regardless of failure of individual subpollers.
	if saveErr := save(ctx, &stateBefore); saveErr != nil {
		// saving error supersedes subpoller errors.
		return saveErr
	}
	err = common.MostSevereError(errs)
	switch n, first := errs.Summary(); {
	case n == len(errs):
		return errors.Annotate(first, "no progress on any poller, first error").Err()
	case err != nil:
		// Some progress. We'll retry during next poll.
		// TODO(tandrii): revisit this logic once CV subscribes to PubSub and makes
		// polling much less frequent.
		err = errors.Annotate(err, "failed %d/%d pollers for %q. Most severe error:", n, len(errs), luciProject).Err()
		common.LogError(ctx, err)
	}
	return nil
}

// updateConfig fetches latest config and updates poller's state
// in RAM only.
func (p *Poller) updateConfig(ctx context.Context, s *State, meta prjcfg.Meta) error {
	s.ConfigHash = meta.Hash()
	cgs, err := meta.GetConfigGroups(ctx)
	if err != nil {
		return err
	}
	// TODO(tandrii): gobmap.Update will need meta & cgs. Pass it to it.
	if err := gobmap.Update(ctx, s.LuciProject); err != nil {
		return err
	}
	proposed := partitionConfig(cgs)
	toUse, discarded := reuseIfPossible(s.SubPollers.GetSubPollers(), proposed)
	for _, d := range discarded {
		if err := p.scheduleRefreshTasks(ctx, s.LuciProject, d.GetHost(), d.Changes); err != nil {
			return err
		}
	}
	s.SubPollers = &SubPollers{SubPollers: toUse}
	return nil
}

// save saves the state of poller after the poll.
func save(ctx context.Context, s *State) error {
	var innerErr error
	var copied State
	err := datastore.RunInTransaction(ctx, func(ctx context.Context) (err error) {
		defer func() { innerErr = err }()
		latest := State{LuciProject: s.LuciProject}
		switch err = datastore.Get(ctx, &latest); {
		case err == datastore.ErrNoSuchEntity:
			if s.EVersion > 0 {
				// At the beginning of the poll, we read an existing state.
				// So, there was a concurrent deletion.
				return errors.Reason("poller state was unexpectedly missing").Err()
			}
			// Then, we'll create it.
		case err != nil:
			return errors.Annotate(err, "failed to get poller state").Tag(transient.Tag).Err()
		case latest.EVersion != s.EVersion:
			return errConcurrentStateUpdate
		}
		copied = *s
		copied.EVersion++
		copied.UpdateTime = clock.Now(ctx).UTC()
		if err = datastore.Put(ctx, &copied); err != nil {
			return errors.Annotate(err, "failed to save poller state").Tag(transient.Tag).Err()
		}
		return nil
	}, nil)

	switch {
	case innerErr != nil:
		return innerErr
	case err != nil:
		return errors.Annotate(err, "failed to save poller state").Tag(transient.Tag).Err()
	default:
		*s = copied
		return nil
	}
}
