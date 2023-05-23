// Copyright 2022 The LUCI Authors.
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

package resultingester

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"google.golang.org/genproto/protobuf/field_mask"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"go.chromium.org/luci/analysis/internal/analysis"
	"go.chromium.org/luci/analysis/internal/analysis/clusteredfailures"
	"go.chromium.org/luci/analysis/internal/changepoints"
	tvbexporter "go.chromium.org/luci/analysis/internal/changepoints/bqexporter"
	"go.chromium.org/luci/analysis/internal/clustering/chunkstore"
	"go.chromium.org/luci/analysis/internal/clustering/ingestion"
	"go.chromium.org/luci/analysis/internal/config"
	"go.chromium.org/luci/analysis/internal/ingestion/control"
	"go.chromium.org/luci/analysis/internal/resultdb"
	"go.chromium.org/luci/analysis/internal/services/resultcollector"
	"go.chromium.org/luci/analysis/internal/tasks/taskspb"
	"go.chromium.org/luci/analysis/internal/testresults"
	"go.chromium.org/luci/analysis/internal/testverdicts"
	pb "go.chromium.org/luci/analysis/proto/v1"
	"go.chromium.org/luci/common/clock"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/common/retry/transient"
	"go.chromium.org/luci/common/trace"
	"go.chromium.org/luci/common/tsmon/field"
	"go.chromium.org/luci/common/tsmon/metric"
	rdbpb "go.chromium.org/luci/resultdb/proto/v1"
	"go.chromium.org/luci/server"
	"go.chromium.org/luci/server/span"
	"go.chromium.org/luci/server/tq"
)

const (
	resultIngestionTaskClass = "result-ingestion"
	resultIngestionQueue     = "result-ingestion"

	// ingestionEarliest is the oldest data that may be ingested by
	// LUCI Analysis.
	// This is an offset relative to the current time, and should be kept
	// in sync with the data retention period in Spanner and BigQuery.
	ingestionEarliest = -90 * 24 * time.Hour

	// ingestionLatest is the newest data that may be ingested by
	// LUCI Analysis.
	// This is an offset relative to the current time. It is designed to
	// allow for clock drift.
	ingestionLatest = 24 * time.Hour
)

var (
	taskCounter = metric.NewCounter(
		"analysis/ingestion/task_completion",
		"The number of completed LUCI Analysis ingestion tasks, by build project and outcome.",
		nil,
		// The LUCI Project.
		field.String("project"),
		// "success", "failed_validation",
		// "ignored_no_invocation", "ignored_has_ancestor",
		// "ignored_invocation_not_found", "ignored_resultdb_permission_denied",
		// "ignored_project_not_allowlisted".
		field.String("outcome"))

	testVariantReadMask = &fieldmaskpb.FieldMask{
		Paths: []string{
			"test_id",
			"variant_hash",
			"status",
			"variant",
			"test_metadata",
			"sources_id",
			"exonerations.*.explanation_html",
			"exonerations.*.reason",
			"results.*.result.name",
			"results.*.result.result_id",
			"results.*.result.expected",
			"results.*.result.status",
			"results.*.result.summary_html",
			"results.*.result.start_time",
			"results.*.result.duration",
			"results.*.result.tags",
			"results.*.result.failure_reason",
			"results.*.result.properties",
		},
	}

	buildReadMask = &field_mask.FieldMask{
		Paths: []string{"builder", "infra.resultdb", "status", "input", "output", "ancestor_ids"},
	}

	// chromiumMilestoneProjectPrefix is the LUCI project prefix
	// of chromium milestone projects, e.g. chromium-m100.
	chromiumMilestoneProjectRE = regexp.MustCompile(`^(chrome|chromium)-m[0-9]+$`)
)

// Options configures test result ingestion.
type Options struct {
}

type resultIngester struct {
	// clustering is used to ingest test failures for clustering.
	clustering *ingestion.Ingester
	// verdictExporter is used to export test verdictExporter.
	verdictExporter *testverdicts.Exporter
	// testVariantBranchExporter is use to export change point analysis results.
	testVariantBranchExporter *tvbexporter.Exporter
}

var resultIngestion = tq.RegisterTaskClass(tq.TaskClass{
	ID:        resultIngestionTaskClass,
	Prototype: &taskspb.IngestTestResults{},
	Queue:     resultIngestionQueue,
	Kind:      tq.Transactional,
})

// RegisterTaskHandler registers the handler for result ingestion tasks.
func RegisterTaskHandler(srv *server.Server) error {
	ctx := srv.Context
	cfg, err := config.Get(ctx)
	if err != nil {
		return err
	}
	chunkStore, err := chunkstore.NewClient(ctx, cfg.ChunkGcsBucket)
	if err != nil {
		return err
	}
	srv.RegisterCleanup(func(context.Context) {
		chunkStore.Close()
	})

	cf, err := clusteredfailures.NewClient(ctx, srv.Options.CloudProject)
	if err != nil {
		return err
	}
	srv.RegisterCleanup(func(context.Context) {
		err := cf.Close()
		if err != nil {
			logging.Errorf(ctx, "Cleaning up clustered failures client: %s", err)
		}
	})

	verdictClient, err := testverdicts.NewClient(ctx, srv.Options.CloudProject)
	if err != nil {
		return err
	}
	srv.RegisterCleanup(func(ctx context.Context) {
		err := verdictClient.Close()
		if err != nil {
			logging.Errorf(ctx, "Cleaning up test verdicts client: %s", err)
		}
	})

	tvbBQClient, err := tvbexporter.NewClient(ctx, srv.Options.CloudProject)
	if err != nil {
		return err
	}
	srv.RegisterCleanup(func(ctx context.Context) {
		err := tvbBQClient.Close()
		if err != nil {
			logging.Errorf(ctx, "Cleaning up test variant branch BQExporter client: %s", err)
		}
	})

	analysis := analysis.NewClusteringHandler(cf)
	ri := &resultIngester{
		clustering:                ingestion.New(chunkStore, analysis),
		verdictExporter:           testverdicts.NewExporter(verdictClient),
		testVariantBranchExporter: tvbexporter.NewExporter(tvbBQClient),
	}
	handler := func(ctx context.Context, payload proto.Message) error {
		task := payload.(*taskspb.IngestTestResults)
		return ri.ingestTestResults(ctx, task)
	}
	resultIngestion.AttachHandler(handler)
	return nil
}

// Schedule enqueues a task to ingest test results from a build.
func Schedule(ctx context.Context, task *taskspb.IngestTestResults) {
	tq.MustAddTask(ctx, &tq.Task{
		Title:   fmt.Sprintf("%s-%s-%d-page-%v", task.Build.Project, task.Build.Host, task.Build.Id, task.TaskIndex),
		Payload: task,
	})
}

func (i *resultIngester) ingestTestResults(ctx context.Context, payload *taskspb.IngestTestResults) error {
	if err := validateRequest(ctx, payload); err != nil {
		project := "(unknown)"
		if payload.GetBuild().GetProject() != "" {
			project = payload.Build.Project
		}
		taskCounter.Add(ctx, 1, project, "failed_validation")
		return tq.Fatal.Apply(err)
	}

	isProjectEnabled, err := isProjectEnabledForIngestion(ctx, payload.Build.Project)
	if err != nil {
		return transient.Tag.Apply(err)
	}
	if !isProjectEnabled {
		taskCounter.Add(ctx, 1, payload.Build.Project, "ignored_project_not_allowlisted")
		return nil
	}

	if !payload.Build.HasInvocation {
		// Build does not have a ResultDB invocation to ingest.
		logging.Debugf(ctx, "Skipping ingestion of build %s-%d because it has no ResultDB invocation.",
			payload.Build.Host, payload.Build.Id)
		taskCounter.Add(ctx, 1, payload.Build.Project, "ignored_no_invocation")
		return nil
	}

	if payload.Build.IsIncludedByAncestor {
		// Yes. Do not ingest this build to avoid ingesting the same test
		// results multiple times.
		taskCounter.Add(ctx, 1, payload.Build.Project, "ignored_has_ancestor")
		return nil
	}

	rdbHost := payload.Build.ResultdbHost
	invName := control.BuildInvocationName(payload.Build.Id)
	rc, err := resultdb.NewClient(ctx, rdbHost, payload.Build.Project)
	if err != nil {
		return transient.Tag.Apply(err)
	}
	inv, err := rc.GetInvocation(ctx, invName)
	code := status.Code(err)
	if code == codes.NotFound {
		// Invocation not found, end the task gracefully.
		logging.Warningf(ctx, "Invocation %s for project %s not found.",
			invName, payload.Build.Project)
		taskCounter.Add(ctx, 1, payload.Build.Project, "ignored_invocation_not_found")
		return nil
	}
	if code == codes.PermissionDenied {
		// Invocation not found, end the task gracefully.
		logging.Warningf(ctx, "Permission denied to read invocation %s for project %s.",
			invName, payload.Build.Project)
		taskCounter.Add(ctx, 1, payload.Build.Project, "ignored_resultdb_permission_denied")
		return nil
	}
	if err != nil {
		logging.Warningf(ctx, "GetInvocation has error code %s.", code)
		return transient.Tag.Apply(errors.Annotate(err, "get invocation").Err())
	}

	ingestedInv, gitRef, err := extractIngestionContext(payload, inv)
	if err != nil {
		return err
	}

	if payload.TaskIndex == 0 {
		// The first task should create the ingested invocation record
		// and git reference record referenced from the invocation record
		// (if any).
		err = recordIngestionContext(ctx, ingestedInv, gitRef)
		if err != nil {
			return err
		}
	}

	// Query test variants from ResultDB.
	req := &rdbpb.QueryTestVariantsRequest{
		Invocations: []string{inv.Name},
		ResultLimit: 100,
		PageSize:    10000,
		ReadMask:    testVariantReadMask,
		PageToken:   payload.PageToken,
	}
	rsp, err := rc.QueryTestVariants(ctx, req)
	if err != nil {
		err = errors.Annotate(err, "query test variants").Err()
		return transient.Tag.Apply(err)
	}

	// Schedule a task to deal with the next page of results (if needed).
	// Do this immediately, so that task can commence while we are still
	// inserting the results for this page.
	if rsp.NextPageToken != "" {
		if err := scheduleNextTask(ctx, payload, rsp.NextPageToken); err != nil {
			err = errors.Annotate(err, "schedule next task").Err()
			return transient.Tag.Apply(err)
		}
	}

	// Record the test results for test history.
	err = recordTestResults(ctx, ingestedInv, rsp.TestVariants)
	if err != nil {
		// If any transaction failed, the task will be retried and the tables will be
		// eventual-consistent.
		return errors.Annotate(err, "record test results").Err()
	}

	// Clustering and test variant analysis currently don't support chromium
	// milestone projects.
	if chromiumMilestoneProjectRE.MatchString(payload.Build.Project) {
		return nil
	}

	failingTVs := filterToTestVariantsWithUnexpectedFailures(rsp.TestVariants)
	nextPageToken := rsp.NextPageToken

	// Insert the test results for clustering.
	err = ingestForClustering(ctx, i.clustering, payload, ingestedInv, failingTVs)
	if err != nil {
		return err
	}

	// Ingest for test variant analysis.
	realmCfg, err := config.Realm(ctx, inv.Realm)
	if err != nil && err != config.RealmNotExistsErr {
		return transient.Tag.Apply(err)
	}

	ingestForTestVariantAnalysis := realmCfg != nil &&
		shouldIngestForTestVariants(realmCfg, payload)

	if ingestForTestVariantAnalysis {
		builder := payload.Build.Builder
		if err := createOrUpdateAnalyzedTestVariants(ctx, inv.Realm, builder, failingTVs); err != nil {
			err = errors.Annotate(err, "ingesting for test variant analysis").Err()
			return transient.Tag.Apply(err)
		}

		if nextPageToken == "" {
			// In the last task, after all test variants ingested.
			isPreSubmit := payload.PresubmitRun != nil
			contributedToCLSubmission := payload.PresubmitRun != nil &&
				payload.PresubmitRun.Mode == pb.PresubmitRunMode_FULL_RUN &&
				payload.PresubmitRun.Status == pb.PresubmitRunStatus_PRESUBMIT_RUN_STATUS_SUCCEEDED

			task := &taskspb.CollectTestResults{
				Resultdb: &taskspb.ResultDB{
					Invocation: inv,
					Host:       rdbHost,
				},
				Builder:                   builder,
				Project:                   payload.Build.Project,
				IsPreSubmit:               isPreSubmit,
				ContributedToClSubmission: contributedToCLSubmission,
			}
			if err = resultcollector.Schedule(ctx, task); err != nil {
				return transient.Tag.Apply(err)
			}
		}
	}

	// Ingest for test variant analysis (change point analysis).
	// Note that this is different from the ingestForTestVariantAnalysis above
	// which should eventually be removed.
	// See go/luci-test-variant-analysis-design for details.
	err = ingestForChangePointAnalysis(ctx, i.testVariantBranchExporter, rsp, payload)
	if err != nil {
		// Only log the error for now, we will return error when everything is
		// working.
		err = errors.Annotate(err, "change point analysis").Err()
		logging.Errorf(ctx, err.Error())
		// return err
	}

	err = ingestForVerdictExport(ctx, i.verdictExporter, rsp, inv, payload)
	if err != nil {
		return errors.Annotate(err, "export verdicts").Err()
	}

	if nextPageToken == "" {
		// In the last task.
		taskCounter.Add(ctx, 1, payload.Build.Project, "success")
	}
	return nil
}

// isProjectEnabledForIngestion returns if the LUCI project is enabled for
// ingestion. By default, all LUCI projects are enabled for ingestion, but
// it is possible to limit ingestion to an allowlisted set in the
// service configuration.
func isProjectEnabledForIngestion(ctx context.Context, project string) (bool, error) {
	cfg, err := config.Get(ctx)
	if err != nil {
		return false, errors.Annotate(err, "get service config").Err()
	}
	if !cfg.Ingestion.GetProjectAllowlistEnabled() {
		return true, nil
	}
	allowList := cfg.Ingestion.GetProjectAllowlist()
	for _, entry := range allowList {
		if project == entry {
			return true, nil
		}
	}
	return false, nil
}

// filterToTestVariantsWithUnexpectedFailures filters the given list of
// test variants to only those with unexpected failures.
func filterToTestVariantsWithUnexpectedFailures(tvs []*rdbpb.TestVariant) []*rdbpb.TestVariant {
	var results []*rdbpb.TestVariant
	for _, tv := range tvs {
		if hasUnexpectedFailures(tv) {
			results = append(results, tv)
		}
	}
	return results
}

// scheduleNextTask schedules a task to continue the ingestion,
// starting at the given page token.
// If a continuation task for this task has been previously scheduled
// (e.g. in a previous try of this task), this method does nothing.
func scheduleNextTask(ctx context.Context, task *taskspb.IngestTestResults, nextPageToken string) error {
	if nextPageToken == "" {
		// If the next page token is "", it means ResultDB returned the
		// last page. We should not schedule a continuation task.
		panic("next page token cannot be the empty page token")
	}
	buildID := control.BuildID(task.Build.Host, task.Build.Id)

	// Schedule the task transactionally, conditioned on it not having been
	// scheduled before.
	_, err := span.ReadWriteTransaction(ctx, func(ctx context.Context) error {
		entries, err := control.Read(ctx, []string{buildID})
		if err != nil {
			return errors.Annotate(err, "read ingestion record").Err()
		}

		entry := entries[0]
		if entry == nil {
			return errors.Reason("build %v does not have ingestion record", buildID).Err()
		}
		if task.TaskIndex >= entry.TaskCount {
			// This should nver happen.
			panic("current ingestion task not recorded on ingestion control record")
		}
		nextTaskIndex := task.TaskIndex + 1
		if nextTaskIndex != entry.TaskCount {
			// Next task has already been created in the past. Do not create
			// it again.
			// This can happen if the ingestion task failed after
			// it scheduled the ingestion task for the next page,
			// and was subsequently retried.
			return nil
		}
		entry.TaskCount = entry.TaskCount + 1
		if err := control.InsertOrUpdate(ctx, entry); err != nil {
			return errors.Annotate(err, "update ingestion record").Err()
		}

		itrTask := &taskspb.IngestTestResults{
			PartitionTime: task.PartitionTime,
			Build:         task.Build,
			PresubmitRun:  task.PresubmitRun,
			PageToken:     nextPageToken,
			TaskIndex:     nextTaskIndex,
		}
		Schedule(ctx, itrTask)

		return nil
	})
	return err
}

func ingestForClustering(ctx context.Context, clustering *ingestion.Ingester, payload *taskspb.IngestTestResults, inv *testresults.IngestedInvocation, tvs []*rdbpb.TestVariant) (err error) {
	ctx, s := trace.StartSpan(ctx, "go.chromium.org/luci/analysis/internal/services/resultingester.ingestForClustering")
	defer func() { s.End(err) }()

	changelists := make([]*pb.Changelist, 0, len(inv.Changelists))
	for _, cl := range inv.Changelists {
		changelists = append(changelists, &pb.Changelist{
			Host:      cl.Host,
			Change:    cl.Change,
			Patchset:  int32(cl.Patchset),
			OwnerKind: cl.OwnerKind,
		})
	}

	// Setup clustering ingestion.
	opts := ingestion.Options{
		TaskIndex:     payload.TaskIndex,
		Project:       inv.Project,
		PartitionTime: inv.PartitionTime,
		Realm:         inv.Project + ":" + inv.SubRealm,
		InvocationID:  inv.IngestedInvocationID,
		BuildStatus:   inv.BuildStatus,
		Changelists:   changelists,
	}

	if payload.PresubmitRun != nil {
		opts.PresubmitRun = &ingestion.PresubmitRun{
			ID:     payload.PresubmitRun.PresubmitRunId,
			Owner:  payload.PresubmitRun.Owner,
			Mode:   payload.PresubmitRun.Mode,
			Status: payload.PresubmitRun.Status,
		}
		opts.BuildCritical = payload.PresubmitRun.Critical
		if payload.PresubmitRun.Critical && inv.BuildStatus == pb.BuildStatus_BUILD_STATUS_FAILURE &&
			payload.PresubmitRun.Status == pb.PresubmitRunStatus_PRESUBMIT_RUN_STATUS_SUCCEEDED {
			logging.Warningf(ctx, "Inconsistent data from LUCI CV: build %v/%v was critical to presubmit run %v/%v and failed, but presubmit run succeeded.",
				payload.Build.Host, payload.Build.Id, payload.PresubmitRun.PresubmitRunId.System, payload.PresubmitRun.PresubmitRunId.Id)
		}
	}
	// Clustering ingestion is designed to behave gracefully in case of
	// a task retry. Given the same options and same test variants (in
	// the same order), the IDs and content of the chunks it writes is
	// designed to be stable. If chunks already exist, it will skip them.
	if err := clustering.Ingest(ctx, opts, tvs); err != nil {
		err = errors.Annotate(err, "ingesting for clustering").Err()
		return transient.Tag.Apply(err)
	}
	return nil
}

func ingestForChangePointAnalysis(ctx context.Context, exporter *tvbexporter.Exporter, rsp *rdbpb.QueryTestVariantsResponse, payload *taskspb.IngestTestResults) (err error) {
	ctx, s := trace.StartSpan(ctx, "go.chromium.org/luci/analysis/internal/services/resultingester.ingestForChangePointAnalysis")
	defer func() { s.End(err) }()

	cfg, err := config.Get(ctx)
	if err != nil {
		return errors.Annotate(err, "read config").Err()
	}
	tvaEnabled := cfg.TestVariantAnalysis != nil && cfg.TestVariantAnalysis.Enabled
	if !tvaEnabled {
		return nil
	}
	err = changepoints.Analyze(ctx, rsp.TestVariants, payload, rsp.Sources, exporter)
	if err != nil {
		return errors.Annotate(err, "analyze test variants").Err()
	}
	return nil
}

func ingestForVerdictExport(ctx context.Context, verdictExporter *testverdicts.Exporter,
	rsp *rdbpb.QueryTestVariantsResponse, inv *rdbpb.Invocation, payload *taskspb.IngestTestResults) (err error) {

	ctx, s := trace.StartSpan(ctx, "go.chromium.org/luci/analysis/internal/services/resultingester.ingestForVerdictExport")
	defer func() { s.End(err) }()

	cfg, err := config.Get(ctx)
	if err != nil {
		return errors.Annotate(err, "read config").Err()
	}
	enabled := cfg.TestVerdictExport != nil && cfg.TestVerdictExport.Enabled
	if !enabled {
		return nil
	}
	// Export test verdicts.
	exportOptions := testverdicts.ExportOptions{
		Payload:    payload,
		Invocation: inv,
	}
	err = verdictExporter.Export(ctx, rsp, exportOptions)
	if err != nil {
		return errors.Annotate(err, "export").Err()
	}
	return nil
}

func validateRequest(ctx context.Context, payload *taskspb.IngestTestResults) error {
	if !payload.PartitionTime.IsValid() {
		return errors.New("partition time must be specified and valid")
	}
	t := payload.PartitionTime.AsTime()
	now := clock.Now(ctx)
	if t.Before(now.Add(ingestionEarliest)) {
		return fmt.Errorf("partition time (%v) is too long ago", t)
	} else if t.After(now.Add(ingestionLatest)) {
		return fmt.Errorf("partition time (%v) is too far in the future", t)
	}
	if payload.Build == nil {
		return errors.New("build must be specified")
	}
	if err := control.ValidateBuildResult(payload.Build); err != nil {
		return err
	}
	return nil
}
