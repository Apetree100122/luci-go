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
	"time"

	"cloud.google.com/go/spanner"

	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/sync/parallel"
	rdbpbutil "go.chromium.org/luci/resultdb/pbutil"
	rdbpb "go.chromium.org/luci/resultdb/proto/v1"
	"go.chromium.org/luci/server/span"

	"go.chromium.org/luci/analysis/internal/ingestion/resultdb"
	"go.chromium.org/luci/analysis/internal/perms"
	"go.chromium.org/luci/analysis/internal/tasks/taskspb"
	"go.chromium.org/luci/analysis/internal/testresults"
	"go.chromium.org/luci/analysis/internal/tracing"
	"go.chromium.org/luci/analysis/pbutil"
	pb "go.chromium.org/luci/analysis/proto/v1"
)

// maximumCLs is the maximum number of changelists to capture from any
// buildbucket run, after which the CL list is truncated. This avoids builds
// with an excessive number of included CLs from storing an excessive amount
// of data per failure.
const maximumCLs = 10

// IngestionContext captures context for test results ingested for a build.
type IngestionContext struct {
	Project string
	// IngestedInvocationID is the ID of the (root) ResultDB invocation
	// being ingested, excluding "invocations/".
	IngestedInvocationID string
	SubRealm             string
	PartitionTime        time.Time
	BuildStatus          pb.BuildStatus

	// The unsubmitted changelists tested (if any). Limited to
	// at most 10 changelists.
	Changelists []testresults.Changelist
}

// extractIngestionContext extracts the ingested invocation and
// the git reference tested (if any).
func extractIngestionContext(task *taskspb.IngestTestResults, inv *rdbpb.Invocation) (*IngestionContext, error) {
	invID, err := rdbpbutil.ParseInvocationName(inv.Name)
	if err != nil {
		// This should never happen. Inv was originated from ResultDB.
		panic(err)
	}

	proj, subRealm, err := perms.SplitRealm(inv.Realm)
	if err != nil {
		return nil, errors.Annotate(err, "invocation has invalid realm: %q", inv.Realm).Err()
	}
	if proj != task.Build.Project {
		return nil, errors.Reason("invocation project (%q) does not match build project (%q) for build %s-%d",
			proj, task.Build.Project, task.Build.Host, task.Build.Id).Err()
	}

	gerritChanges := task.Build.Changelists
	changelists := make([]testresults.Changelist, 0, len(gerritChanges))
	for _, change := range gerritChanges {
		if err := testresults.ValidateGerritHostname(change.Host); err != nil {
			return nil, err
		}
		changelists = append(changelists, testresults.Changelist{
			Host:      change.Host,
			Change:    change.Change,
			Patchset:  change.Patchset,
			OwnerKind: change.OwnerKind,
		})
	}
	// TODO (b/258734241): This has been migrated to join-build task queue.
	// Remove when old build completions flushed out.
	// Store the tested changelists in sorted order. This ensures that for
	// the same combination of CLs tested, the arrays are identical.
	testresults.SortChangelists(changelists)

	// TODO (b/258734241): This has been migrated to join-build task queue.
	// Remove when old build completions flushed out.
	// Truncate the list of changelists to avoid storing an excessive number.
	// Apply truncation after sorting to ensure a stable set of changelists.
	if len(changelists) > maximumCLs {
		changelists = changelists[:maximumCLs]
	}

	ingestion := &IngestionContext{
		Project:              proj,
		IngestedInvocationID: invID,
		SubRealm:             subRealm,
		PartitionTime:        task.PartitionTime.AsTime(),
		BuildStatus:          task.Build.Status,
		Changelists:          changelists,
	}

	return ingestion, nil
}

type batch struct {
	// The test realms to insert/update.
	// Test realms should be inserted before any test variant realms.
	testRealms []*spanner.Mutation
	// The test variant realms to insert/update.
	// Test variant realms should be inserted before any test results.
	testVariantRealms []*spanner.Mutation
	// Test results to insert. Already prepared as Spanner mutations.
	testResults []*spanner.Mutation
}

func batchTestResults(ingestion *IngestionContext, tvs []*rdbpb.TestVariant, outputC chan batch) {
	// Must be selected such that no more than 20,000 mutations occur in
	// one transaction in the worst case.
	const batchSize = 900

	var trs []*spanner.Mutation
	var tvrs []*spanner.Mutation
	testRealmSet := make(map[testresults.TestRealm]bool, batchSize)
	startBatch := func() {
		trs = make([]*spanner.Mutation, 0, batchSize)
		tvrs = make([]*spanner.Mutation, 0, batchSize)
		testRealmSet = make(map[testresults.TestRealm]bool, batchSize)
	}
	outputBatch := func() {
		if len(trs) == 0 {
			// This should never happen.
			panic("Pushing empty batch")
		}
		testRealms := make([]*spanner.Mutation, 0, len(testRealmSet))
		for testRealm := range testRealmSet {
			testRealms = append(testRealms, testRealm.SaveUnverified())
		}

		outputC <- batch{
			testRealms:        testRealms,
			testVariantRealms: tvrs,
			testResults:       trs,
		}
	}

	startBatch()
	for _, tv := range tvs {
		// Limit batch size.
		// Keep all results for one test variant in one batch, so that the
		// TestVariantRealm record is kept together with the test results.
		if len(trs) > batchSize {
			outputBatch()
			startBatch()
		}

		testRealm := testresults.TestRealm{
			Project:           ingestion.Project,
			TestID:            tv.TestId,
			SubRealm:          ingestion.SubRealm,
			LastIngestionTime: spanner.CommitTimestamp,
		}
		testRealmSet[testRealm] = true

		tvr := testresults.TestVariantRealm{
			Project:           ingestion.Project,
			TestID:            tv.TestId,
			VariantHash:       tv.VariantHash,
			SubRealm:          ingestion.SubRealm,
			Variant:           pbutil.VariantFromResultDB(tv.Variant),
			LastIngestionTime: spanner.CommitTimestamp,
		}
		tvrs = append(tvrs, tvr.SaveUnverified())

		exonerationReasons := make([]pb.ExonerationReason, 0, len(tv.Exonerations))
		for _, ex := range tv.Exonerations {
			exonerationReasons = append(exonerationReasons, pbutil.ExonerationReasonFromResultDB(ex.Reason))
		}

		isFromBisection := isFromLuciBisection(tv)

		// Group results into test runs and order them by start time.
		resultsByRun := resultdb.GroupAndOrderTestResults(tv.Results)
		for runIndex, run := range resultsByRun {
			for resultIndex, inputTR := range run {
				tr := testresults.TestResult{
					Project:              ingestion.Project,
					TestID:               tv.TestId,
					PartitionTime:        ingestion.PartitionTime,
					VariantHash:          tv.VariantHash,
					IngestedInvocationID: ingestion.IngestedInvocationID,
					RunIndex:             int64(runIndex),
					ResultIndex:          int64(resultIndex),
					IsUnexpected:         !inputTR.Result.Expected,
					Status:               pbutil.TestResultStatusFromResultDB(inputTR.Result.Status),
					ExonerationReasons:   exonerationReasons,
					SubRealm:             ingestion.SubRealm,
					Changelists:          ingestion.Changelists,
					IsFromBisection:      isFromBisection,
				}
				if inputTR.Result.Duration != nil {
					d := new(time.Duration)
					*d = inputTR.Result.Duration.AsDuration()
					tr.RunDuration = d
				}
				// Convert the test result into a mutation immediately
				// to avoid storing both the TestResult object and
				// mutation object in memory until the transaction
				// commits.
				trs = append(trs, tr.SaveUnverified())
			}
		}
	}
	if len(trs) > 0 {
		outputBatch()
	}
}

// recordTestResults records test results from an test-verdict-ingestion task.
func recordTestResults(ctx context.Context, ingestion *IngestionContext, tvs []*rdbpb.TestVariant) (err error) {
	ctx, s := tracing.Start(ctx, "go.chromium.org/luci/analysis/internal/services/resultingester.recordTestResults")
	defer func() { tracing.End(s, err) }()

	const workerCount = 8

	return parallel.WorkPool(workerCount, func(c chan<- func() error) {
		batchC := make(chan batch)

		c <- func() error {
			defer close(batchC)
			batchTestResults(ingestion, tvs, batchC)
			return nil
		}

		for batch := range batchC {
			// Bind to a local variable so it can be used in a goroutine without being
			// overwritten. See https://go.dev/doc/faq#closures_and_goroutines
			batch := batch

			c <- func() error {
				// Write to different tables in different transactions to minimize the
				// number of splits involved in each transaction.
				_, err := span.ReadWriteTransaction(ctx, func(ctx context.Context) error {
					span.BufferWrite(ctx, batch.testRealms...)
					return nil
				})
				if err != nil {
					return errors.Annotate(err, "inserting test realms").Err()
				}
				_, err = span.ReadWriteTransaction(ctx, func(ctx context.Context) error {
					span.BufferWrite(ctx, batch.testVariantRealms...)
					return nil
				})
				if err != nil {
					return errors.Annotate(err, "inserting test variant realms").Err()
				}
				_, err = span.ReadWriteTransaction(ctx, func(ctx context.Context) error {
					span.BufferWrite(ctx, batch.testResults...)
					return nil
				})
				if err != nil {
					return errors.Annotate(err, "inserting test results").Err()
				}
				return nil
			}
		}
	})
}

// isFromLuciBisection checks if the test variant was from LUCI Bisection.
// LUCI Bisection test results will have the tag "is_luci_bisection" = "true".
func isFromLuciBisection(tv *rdbpb.TestVariant) bool {
	// This should not happen.
	if len(tv.Results) == 0 {
		panic("test variant has 0 results")
	}
	for _, tag := range tv.Results[0].Result.Tags {
		if tag.Key == "is_luci_bisection" && tag.Value == "true" {
			return true
		}
	}
	return false
}
