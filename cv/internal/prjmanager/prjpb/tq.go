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

package prjpb

import (
	"context"
	"fmt"
	"time"

	"go.chromium.org/luci/common/clock"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/cv/internal/eventbox"
	"go.chromium.org/luci/gae/service/datastore"
	"go.chromium.org/luci/server/tq"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	// PMTaskInterval is target frequency of executions of ManageProjectTask.
	//
	// See Dispatch() for details.
	PMTaskInterval = time.Second

	// MaxAcceptableDelay prevents TQ tasks which arrive too late from invoking PM.
	//
	// MaxAcceptableDelay / PMTaskInterval effectively limits # concurrent
	// invocations of PM on the same project that may happen due to task retries,
	// delays, and queue throttling.
	//
	// Do not set too low, as this may prevent actual PM invoking from happening at
	// all if the TQ is overloaded.
	MaxAcceptableDelay = 60 * time.Second

	ManageProjectTaskClass     = "manage-project"
	KickManageProjectTaskClass = "kick-" + ManageProjectTaskClass
	PurgeProjectCLTaskClass    = "purge-project-cl"
)

// TaskRefs are task refs used by PM to separate task creation and handling,
// which in turns avoids circular dependency.
type TaskRefs struct {
	ManageProject     tq.TaskClassRef
	KickManageProject tq.TaskClassRef
	PurgeProjectCL    tq.TaskClassRef

	tqd *tq.Dispatcher
}

func Register(tqd *tq.Dispatcher) TaskRefs {
	return TaskRefs{
		tqd: tqd,

		ManageProject: tqd.RegisterTaskClass(tq.TaskClass{
			ID:        ManageProjectTaskClass,
			Prototype: &ManageProjectTask{},
			Queue:     "manage-project",
			Kind:      tq.NonTransactional,
			Quiet:     true,
		}),
		KickManageProject: tqd.RegisterTaskClass(tq.TaskClass{
			ID:        KickManageProjectTaskClass,
			Prototype: &KickManageProjectTask{},
			Queue:     "kick-manage-project",
			Kind:      tq.Transactional,
			Quiet:     true,
		}),
		PurgeProjectCL: tqd.RegisterTaskClass(tq.TaskClass{
			ID:        PurgeProjectCLTaskClass,
			Prototype: &PurgeCLTask{},
			Queue:     "purge-project-cl",
			Kind:      tq.Transactional,
			Quiet:     false, // these tasks are rare enough that verbosity only helps.
		}),
	}
}

// Dispatch ensures invocation of ProjectManager via ManageProjectTask.
//
// ProjectManager will be invoked at approximately no earlier than both:
// * eta time
// * next possible.
//
// To avoid actually dispatching TQ tasks in tests, use pmtest.MockDispatch().
func (tr TaskRefs) Dispatch(ctx context.Context, luciProject string, eta time.Time) error {
	mock, mocked := ctx.Value(&mockDispatcherContextKey).(func(string, time.Time))

	if datastore.CurrentTransaction(ctx) != nil {
		// TODO(tandrii): use txndefer to immediately trigger a ManageProjectTask after
		// transaction completes to reduce latency in *most* circumstances.
		// The KickManageProjectTask is still required for correctness.
		payload := &KickManageProjectTask{LuciProject: luciProject}
		if !eta.IsZero() {
			payload.Eta = timestamppb.New(eta)
		}

		if mocked {
			mock(luciProject, eta)
			return nil
		}
		return tr.tqd.AddTask(ctx, &tq.Task{
			Title:            luciProject,
			DeduplicationKey: "", // not allowed in a transaction
			Payload:          payload,
		})
	}

	// If actual local clock is more than `clockDrift` behind, the "next" computed
	// ManageProjectTask moment might be already executing, meaning task dedup
	// will ensure no new task will be scheduled AND the already executing run
	// might not have read the Event that was just written.
	// Thus, this should be large for safety. However, large value leads to higher
	// latency of event processing of non-busy ProjectManagers.
	// TODO(tandrii): this can be reduced significantly once safety "ping" events
	// are originated from Config import cron tasks.
	const clockDrift = 100 * time.Millisecond
	now := clock.Now(ctx).Add(clockDrift) // Use the worst possible time.
	if eta.IsZero() || eta.Before(now) {
		eta = now
	}
	eta = eta.Truncate(PMTaskInterval).Add(PMTaskInterval)

	if mocked {
		mock(luciProject, eta)
		return nil
	}
	return tr.tqd.AddTask(ctx, &tq.Task{
		Title:            luciProject,
		DeduplicationKey: fmt.Sprintf("%s\n%d", luciProject, eta.UnixNano()),
		ETA:              eta,
		Payload:          &ManageProjectTask{LuciProject: luciProject, Eta: timestamppb.New(eta)},
	})
}

var mockDispatcherContextKey = "prjpb.mockDispatcher"

// InstallMockDispatcher is used in test to run tests emitting PM events without
// actually dispatching PM tasks.
//
// See pmtest.MockDispatch().
func InstallMockDispatcher(ctx context.Context, f func(luciProject string, eta time.Time)) context.Context {
	return context.WithValue(ctx, &mockDispatcherContextKey, f)
}

// SendNow sends the event to Project's eventbox and invokes PM immediately.
func (tr TaskRefs) SendNow(ctx context.Context, luciProject string, e *Event) error {
	if err := Send(ctx, luciProject, e); err != nil {
		return err
	}
	return tr.Dispatch(ctx, luciProject, time.Time{} /*asap*/)
}

// Send sends the event to Project's eventbox without invoking a PM.
func Send(ctx context.Context, luciProject string, e *Event) error {
	value, err := proto.Marshal(e)
	if err != nil {
		return errors.Annotate(err, "failed to marshal").Err()
	}
	// Must be same as prjmanager.ProjectKind, but can't import due to circular
	// imports.
	to := datastore.MakeKey(ctx, "Project", luciProject)
	return eventbox.Emit(ctx, value, to)
}

// SchedulePurgeCL schedules a task to purge a CL.
func (tr TaskRefs) SchedulePurgeCL(ctx context.Context, t *PurgeCLTask) error {
	return tr.tqd.AddTask(ctx, &tq.Task{
		Payload: t,
		// No DeduplicationKey as these tasks are created transactionally by PM.
		Title: fmt.Sprintf("%s/%d/%s", t.GetLuciProject(), t.GetPurgingCl().GetClid(), t.GetPurgingCl().GetOperationId()),
	})
}

var (
	// DefaultTaskRefs is for backwards compat during migration away from Default
	// tq Dispatcher.
	// TODO(tandrii): remove this.
	DefaultTaskRefs TaskRefs
)

func init() {
	DefaultTaskRefs = Register(&tq.Default)
}
