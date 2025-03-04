// Copyright 2023 The LUCI Authors.
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

package model

import (
	"context"
	"fmt"
	"time"

	"go.chromium.org/luci/auth/identity"
	"go.chromium.org/luci/gae/service/datastore"

	apipb "go.chromium.org/luci/swarming/proto/api_v2"
)

const (
	// ChunkSize is the size of each TaskOutputChunk.Chunk in uncompressed form.
	//
	// The rationale is that appending data to an entity requires reading it
	// first, so it must not be too big. On the other hand, having thousands of
	// small entities is pure overhead.
	//
	// Note that changing this value is a breaking change for existing logs.
	ChunkSize = 100 * 1024

	// MaxChunks sets a limit on how much of a task output to store.
	MaxChunks = 1024
)

// TaskResultCommon contains properties that are common to both TaskRunResult
// and TaskResultSummary.
//
// It is not meant to be stored in the datastore on its own, only as an embedded
// struct inside TaskRunResult or TaskResultSummary.
type TaskResultCommon struct {
	// State is the current state of the task.
	//
	// The index is used in task listing queries to filter by.
	State apipb.TaskState `gae:"state"`

	// Modified is when this entity was created or last modified.
	Modified time.Time `gae:"modified_ts,noindex"`

	// BotVersion is a version of the bot code running the task, if already known.
	BotVersion string `gae:"bot_version,noindex"`

	// BotDimensions are bot dimensions at the moment the bot claimed the task.
	BotDimensions BotDimensions `gae:"bot_dimensions"`

	// BotIdleSince is when the bot running this task finished its previous task,
	// if already known.
	BotIdleSince datastore.Optional[time.Time, datastore.Unindexed] `gae:"bot_idle_since_ts"`

	// BotLogsCloudProject is a GCP project where the bot uploads its logs.
	BotLogsCloudProject string `gae:"bot_logs_cloud_project,noindex"`

	// ServerVersions is a set of server version(s) that touched this entity.
	ServerVersions []string `gae:"server_versions,noindex"`

	// CurrentTaskSlice is an index of the active task slice in the TaskRequest.
	//
	// For TaskResultSummary it is the currently running slice (and it changes
	// over lifetime of a task if it has many slices). For TaskRunResult it is the
	// index representing this concrete run.
	CurrentTaskSlice int64 `gae:"current_task_slice,noindex"`

	// Started is when the bot started executing the task.
	//
	// The index is used in task listing queries to order by this property.
	Started datastore.Nullable[time.Time, datastore.Indexed] `gae:"started_ts"`

	// Completed is when the task switched into a final state.
	//
	// This is either when the bot finished executing it, or when it expired or
	// was canceled.
	//
	// The index is used in a bunch of places:
	//   1. In task listing queries to order by this property.
	//   2. In crashed bot detection to list still pending or running tasks.
	//   3. In BQ export pagination.
	Completed datastore.Nullable[time.Time, datastore.Indexed] `gae:"completed_ts"`

	// Abandoned is set when a task had an internal failure, timed out or was
	// killed by a client request.
	//
	// The index is used in task listing queries to order by this property.
	Abandoned datastore.Nullable[time.Time, datastore.Indexed] `gae:"abandoned_ts"`

	// DurationSecs is how long the task process was running.
	//
	// This is reported explicitly by the bot and it excludes all overhead.
	DurationSecs datastore.Optional[float64, datastore.Unindexed] `gae:"durations"` // legacy plural property name

	// ExitCode is the task process exit code for tasks in COMPLETED state.
	ExitCode datastore.Optional[int64, datastore.Unindexed] `gae:"exit_codes"` // legacy plural property name

	// Failure is true if the task finished with a non-zero process exit code.
	//
	// The index is used in task listing queries to filter by failure.
	Failure bool `gae:"failure"`

	// InternalFailure is true if the task failed due to an internal error.
	InternalFailure bool `gae:"internal_failure,noindex"`

	// StdoutChunks is the number of TaskOutputChunk entities with the output.
	StdoutChunks int64 `gae:"stdout_chunks,noindex"`

	// CASOutputRoot is the digest of the output root uploaded to RBE-CAS.
	CASOutputRoot CASReference `gae:"cas_output_root,lsp,noindex"`

	// CIPDPins is resolved versions of all the CIPD packages used in the task.
	CIPDPins CIPDInput `gae:"cipd_pins,lsp,noindex"`

	// MissingCIPD is the missing CIPD packages in CLIENT_ERROR state.
	MissingCIPD []CIPDPackage `gae:"missing_cipd,lsp,noindex"`

	// MissingCAS is the missing CAS digests in CLIENT_ERROR state.
	MissingCAS []CASReference `gae:"missing_cas,lsp,noindex"`

	// ResultDBInfo is ResultDB invocation info for this task.
	//
	// If this task was deduplicated, this contains invocation info from the
	// previously ran the task deduplicated from.
	ResultDBInfo ResultDBInfo `gae:"resultdb_info,lsp,noindex"`

	// LegacyOutputsRef is no longer used.
	LegacyOutputsRef LegacyProperty `gae:"outputs_ref"`
}

// ResultDBInfo contains invocation ID of the task.
type ResultDBInfo struct {
	// Hostname is the ResultDB service hostname e.g. "results.api.cr.dev".
	Hostname string `gae:"hostname"`
	// Invocation is the ResultDB invocation name for the task, if any.
	Invocation string `gae:"invocation"`
}

// TaskResultSummary represents the overall result of a task.
//
// Parent is a TaskRequest. Key id is always 1.
//
// It is created (in PENDING state) as soon as the task is created and then
// mutated whenever the task changes its state. Its used by various task listing
// APIs.
type TaskResultSummary struct {
	// TaskResultCommon embeds most of result properties.
	TaskResultCommon

	// Extra are entity properties that didn't match any declared ones below.
	//
	// Should normally be empty.
	Extra datastore.PropertyMap `gae:"-,extra"`

	// Key identifies the task, see TaskResultSummaryKey().
	Key *datastore.Key `gae:"$key"`

	// BotID is ID of a bot that runs this task, if already known.
	BotID datastore.Optional[string, datastore.Unindexed] `gae:"bot_id"`

	// Created is a timestamp when the task was submitted.
	//
	// The index is used in task listing queries to order by this property.
	Created time.Time `gae:"created_ts"`

	// Tags are copied from the corresponding TaskRequest entity.
	//
	// The index is used in global task listing queries to filter by.
	Tags []string `gae:"tags"`

	// RequestName is copied from the corresponding TaskRequest entity.
	RequestName string `gae:"name,noindex"`

	// RequestUser is copied from the corresponding TaskRequest entity.
	RequestUser string `gae:"user,noindex"`

	// RequestPriority is copied from the corresponding TaskRequest entity.
	RequestPriority int64 `gae:"priority,noindex"`

	// RequestAuthenticated is copied from the corresponding TaskRequest entity.
	RequestAuthenticated identity.Identity `gae:"request_authenticated,noindex"`

	// RequestRealm is copied from the corresponding TaskRequest entity.
	RequestRealm string `gae:"request_realm,noindex"`

	// RequestPool is a pool the task is targeting.
	//
	// Derived from dimensions in the corresponding TaskRequest entity.
	RequestPool string `gae:"request_pool,noindex"`

	// RequestBotID is a concrete bot the task is targeting (if any).
	//
	// Derived from dimensions in the corresponding TaskRequest entity.
	RequestBotID string `gae:"request_bot_id,noindex"`

	// PropertiesHash is used to find duplicate tasks.
	//
	// It is set to a hash of properties of the task slice only when these
	// conditions are met:
	// - TaskSlice.TaskProperties.Idempotent is true.
	// - State is COMPLETED.
	// - Failure is false (i.e. ExitCode is 0).
	// - InternalFailure is false.
	//
	// The index is used to find an an existing duplicate task when submitting
	// a new task.
	PropertiesHash datastore.Optional[[]byte, datastore.Indexed] `gae:"properties_hash"`

	// TryNumber indirectly identifies if this task was dedupped.
	//
	// This field is leftover from when swarming had internal retries and for that
	// reason has a weird semantics. See https://crbug.com/1065101.
	//
	// Possible values:
	//   null: if the task is still pending, has expired or was canceled.
	//   1: if the task was assigned to a bot and either currently runs or has
	//      finished or crashed already.
	//   0: if the task was dedupped.
	//
	// The index is used in global task listing queries to find what tasks were
	// dedupped.
	TryNumber datastore.Nullable[int64, datastore.Indexed] `gae:"try_number"`

	// CostUSD is an approximate bot time cost spent executing this task.
	CostUSD float64 `gae:"costs_usd,noindex"` // legacy plural property name

	// CostSavedUSD is an approximate cost saved by deduping this task.
	CostSavedUSD float64 `gae:"cost_saved_usd,noindex"`

	// DedupedFrom is set if this task reused results of some previous task.
	//
	// See PropertiesHash for conditions when this is possible.
	//
	// DedupedFrom is a packed TaskRunResult key of the reused result. Note that
	// there's no TaskRunResult representing this task, since it didn't run.
	DedupedFrom string `gae:"deduped_from,noindex"`

	// ExpirationDelay is a delay from TaskRequest.ExpirationTS to the actual
	// expiry time.
	//
	// This is set at expiration process if the last task slice expired by
	// reaching its deadline. Unset if the last slice expired because there were
	// no bots that could run it.
	//
	// Exclusively for monitoring.
	ExpirationDelay datastore.Optional[float64, datastore.Unindexed] `gae:"expiration_delay"`
}

// TaskRequestKey returns the parent task request key or panics if it is unset.
func (e *TaskResultSummary) TaskRequestKey() *datastore.Key {
	key := e.Key.Parent()
	if key == nil || key.Kind() != "TaskRequest" {
		panic(fmt.Sprintf("invalid TaskResultSummary key %q", e.Key))
	}
	return key
}

// TaskResultSummaryKey construct a summary key given a task request key.
func TaskResultSummaryKey(ctx context.Context, taskReq *datastore.Key) *datastore.Key {
	return datastore.NewKey(ctx, "TaskResultSummary", "", 1, taskReq)
}

// TaskRunResult contains result of an attempt to run a task on a bot.
//
// Parent is a TaskResultSummary. Key id is 1.
//
// Unlike TaskResultSummary it is created only when the task is assigned to a
// bot, i.e. existence of this entity means a bot claimed the task and started
// executing it.
type TaskRunResult struct {
	// TaskResultCommon embeds most of result properties.
	TaskResultCommon

	// Extra are entity properties that didn't match any declared ones below.
	//
	// Should normally be empty.
	Extra datastore.PropertyMap `gae:"-,extra"`

	// Key identifies the task, see TaskRunResultKey().
	Key *datastore.Key `gae:"$key"`

	// BotID is ID of a bot that runs this task attempt.
	//
	// The index is used in task listing queries to filter by a specific bot.
	BotID string `gae:"bot_id"`

	// CostUSD is an approximate bot time cost spent executing this task.
	CostUSD float64 `gae:"cost_usd,noindex"`

	// Killing is true if a user requested the task to be canceled while it is
	// already running.
	//
	// Setting this field eventually results in the task moving to KILLED state.
	// This transition also unsets Killing back to false.
	Killing bool `gae:"killing,noindex"`

	// DeadAfter specifies the time after which the bot executing this task
	// is considered dead.
	//
	// It is set after every ping from the bot if the task is in RUNNING state
	// and get unset once the task terminates.
	DeadAfter datastore.Optional[time.Time, datastore.Unindexed] `gae:"dead_after_ts"`
}

// TaskRunResultKey constructs a task run result key given a task request key.
func TaskRunResultKey(ctx context.Context, taskReq *datastore.Key) *datastore.Key {
	return datastore.NewKey(ctx, "TaskRunResult", "", 1, TaskResultSummaryKey(ctx, taskReq))
}

// PerformanceStats contains various timing and performance information about
// the task as reported by the bot.
type PerformanceStats struct {
	// Extra are entity properties that didn't match any declared ones below.
	//
	// Should normally be empty.
	Extra datastore.PropertyMap `gae:"-,extra"`

	// Key identifies the task, see PerformanceStatsKey.
	Key *datastore.Key `gae:"$key"`

	// BotOverheadSecs is the total overhead in seconds, summing overhead from all
	// sections defined below.
	BotOverheadSecs float64 `gae:"bot_overhead,noindex"`

	// CacheTrim is stats of cache trimming before the dependency installations.
	CacheTrim OperationStats `gae:"cache_trim,lsp,noindex"`

	// PackageInstallation is stats of installing CIPD packages before the task.
	PackageInstallation OperationStats `gae:"package_installation,lsp,noindex"`

	// NamedCachesInstall is stats of named cache mounting before the task.
	NamedCachesInstall OperationStats `gae:"named_caches_install,lsp,noindex"`

	// NamedCachesUninstall is stats of named cache unmounting after the task.
	NamedCachesUninstall OperationStats `gae:"named_caches_uninstall,lsp,noindex"`

	// IsolatedDownload is stats of CAS dependencies download before the task.
	IsolatedDownload CASOperationStats `gae:"isolated_download,lsp,noindex"`

	// IsolatedUpload is stats of CAS uploading operation after the task.
	IsolatedUpload CASOperationStats `gae:"isolated_upload,lsp,noindex"`

	// Cleanup is stats of work directory cleanup operation after the task.
	Cleanup OperationStats `gae:"cleanup,lsp,noindex"`
}

// OperationStats is performance stats of a particular operation.
//
// Stored as a unindexed subentity of PerformanceStats entity.
type OperationStats struct {
	// DurationSecs is how long the operation ran.
	DurationSecs float64 `gae:"duration"`
}

// CASOperationStats is performance stats of a CAS operation.
//
// Stored as a unindexed subentity of PerformanceStats entity.
type CASOperationStats struct {
	// DurationSecs is how long the operation ran.
	DurationSecs float64 `gae:"duration"`

	// InitialItems is the number of items in the cache before the operation.
	InitialItems int64 `gae:"initial_number_items"`

	// InitialSize is the total cache size before the operation.
	InitialSize int64 `gae:"initial_size"`

	// ItemsCold is a set of sizes of items that were downloaded or uploaded.
	//
	// It is encoded in a special way, see packedintset.Unpack.
	ItemsCold []byte `gae:"items_cold"`

	// ItemsHot is a set of sizes of items that were already present in the cache.
	//
	// It is encoded in a special way, see packedintset.Unpack.
	ItemsHot []byte `gae:"items_hot"`
}

// PerformanceStatsKey builds a PerformanceStats key given a task request key.
func PerformanceStatsKey(ctx context.Context, taskReq *datastore.Key) *datastore.Key {
	return datastore.NewKey(ctx, "PerformanceStats", "", 1, TaskRunResultKey(ctx, taskReq))
}

// TaskOutputChunk represents a chunk of the task console output.
//
// The number of such chunks per task is tracked in TaskRunResult.StdoutChunks.
// Each entity holds a compressed segment of the task output. For all entities
// except the last one, the uncompressed size of the segment is ChunkSize. That
// way it is always possible to figure out what chunk to use for a given output
// offset.
type TaskOutputChunk struct {
	// Extra are entity properties that didn't match any declared ones below.
	//
	// Should normally be empty.
	Extra datastore.PropertyMap `gae:"-,extra"`

	// Key identifies the task and the chunk index, see TaskOutputChunkKey.
	Key *datastore.Key `gae:"$key"`

	// Chunk is zlib-compressed chunk of the output.
	Chunk []byte `gae:"chunk,noindex"`

	// Gaps is a series of 2 integer pairs, which specifies the part that are
	// invalid. Normally it should be empty. All values are relative to the start
	// of this chunk offset.
	Gaps []int32 `gae:"gaps,noindex"`
}

// TaskOutputChunkKey builds a TaskOutputChunk key for the given chunk index.
func TaskOutputChunkKey(ctx context.Context, taskReq *datastore.Key, idx int64) *datastore.Key {
	return datastore.NewKey(ctx, "TaskOutputChunk", "", idx+1,
		datastore.NewKey(ctx, "TaskOutput", "", 1, TaskRunResultKey(ctx, taskReq)))
}
