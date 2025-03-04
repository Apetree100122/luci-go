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

// Package testverdicts handles export of test verdicts to BigQuery.
package testverdicts

import (
	"context"
	"encoding/hex"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"

	"go.chromium.org/luci/common/errors"
	rdbpbutil "go.chromium.org/luci/resultdb/pbutil"
	rdbpb "go.chromium.org/luci/resultdb/proto/v1"

	"go.chromium.org/luci/analysis/internal/analysis"
	controlpb "go.chromium.org/luci/analysis/internal/ingestion/control/proto"
	"go.chromium.org/luci/analysis/internal/ingestion/resultdb"
	"go.chromium.org/luci/analysis/internal/perms"
	"go.chromium.org/luci/analysis/internal/tasks/taskspb"
	"go.chromium.org/luci/analysis/pbutil"
	bqpb "go.chromium.org/luci/analysis/proto/bq"
	pb "go.chromium.org/luci/analysis/proto/v1"
)

// InsertClient defines an interface for inserting rows into BigQuery.
type InsertClient interface {
	// Insert inserts the given rows into BigQuery.
	Insert(ctx context.Context, rows []*bqpb.TestVerdictRow) error
}

// Exporter provides methods to stream test verdicts into BigQuery.
type Exporter struct {
	client InsertClient
}

// NewExporter instantiates a new Exporter. The given client is used
// to insert rows into BigQuery.
func NewExporter(client InsertClient) *Exporter {
	return &Exporter{client: client}
}

// ExportOptions captures context which will be exported
// alongside the test verdicts.
type ExportOptions struct {
	Payload    *taskspb.IngestTestResults
	Invocation *rdbpb.Invocation
}

// Export exports the given test verdicts to BigQuery.
func (e *Exporter) Export(ctx context.Context, tvs *rdbpb.QueryTestVariantsResponse, opts ExportOptions) error {
	rows := make([]*bqpb.TestVerdictRow, 0, len(tvs.TestVariants))
	for _, tv := range tvs.TestVariants {
		exportRow, err := prepareExportRow(tv, tvs.Sources, opts)
		if err != nil {
			return errors.Annotate(err, "prepare row").Err()
		}
		rows = append(rows, exportRow)
	}
	err := e.client.Insert(ctx, rows)
	if err != nil {
		return errors.Annotate(err, "insert rows").Err()
	}
	return nil
}

// prepareExportRow prepares a BigQuery export row for a
// ResultDB test verdict.
func prepareExportRow(tv *rdbpb.TestVariant, sourcesByID map[string]*rdbpb.Sources, opts ExportOptions) (*bqpb.TestVerdictRow, error) {
	project, _, err := perms.SplitRealm(opts.Invocation.Realm)
	if err != nil {
		return nil, errors.Annotate(err, "invalid realm").Err()
	}

	results := make([]*bqpb.TestVerdictRow_TestResult, 0, len(tv.Results))
	for _, r := range tv.Results {
		resultEntry, err := result(r.Result)
		if err != nil {
			return nil, errors.Annotate(err, "result entry").Err()
		}
		results = append(results, resultEntry)
	}

	exonerations := make([]*bqpb.TestVerdictRow_Exoneration, 0, len(tv.Exonerations))
	for _, e := range tv.Exonerations {
		exonerations = append(exonerations, exoneration(e))
	}

	var sources *pb.Sources
	var sourceRef *pb.SourceRef
	var sourceRefHash string
	if tv.SourcesId != "" {
		sources = pbutil.SourcesFromResultDB(sourcesByID[tv.SourcesId])
		sourceRef = pbutil.SourceRefFromSources(sources)
		sourceRefHash = hex.EncodeToString(pbutil.SourceRefHash(sourceRef))
	}

	var metadata *pb.TestMetadata
	if tv.TestMetadata != nil {
		metadata = pbutil.TestMetadataFromResultDB(tv.TestMetadata)
	}

	var cvRun *bqpb.TestVerdictRow_ChangeVerifierRun
	if opts.Payload.PresubmitRun != nil && opts.Payload.PresubmitRun.PresubmitRunId.System == "luci-cv" {
		cvRun = changeVerifierRun(opts.Payload.PresubmitRun)
	}

	var build *bqpb.TestVerdictRow_BuildbucketBuild
	if opts.Payload.Build != nil {
		build = buildbucketBuild(opts.Payload.Build)
	}

	inv, err := invocation(opts.Invocation)
	if err != nil {
		return nil, errors.Annotate(err, "invocation").Err()
	}

	variant, err := variantJSON(tv.Variant)
	if err != nil {
		return nil, errors.Annotate(err, "variant").Err()
	}

	return &bqpb.TestVerdictRow{
		Project:           project,
		TestId:            tv.TestId,
		Variant:           variant,
		VariantHash:       tv.VariantHash,
		Invocation:        inv,
		PartitionTime:     opts.Payload.PartitionTime,
		Status:            pbutil.TestVerdictStatusFromResultDB(tv.Status),
		Results:           results,
		Exonerations:      exonerations,
		Counts:            counts(results),
		BuildbucketBuild:  build,
		ChangeVerifierRun: cvRun,
		Sources:           sources,
		SourceRef:         sourceRef,
		SourceRefHash:     sourceRefHash,
		TestMetadata:      metadata,
	}, nil
}

func invocation(invocation *rdbpb.Invocation) (*bqpb.TestVerdictRow_InvocationRecord, error) {
	invocationID, err := rdbpbutil.ParseInvocationName(invocation.Name)
	if err != nil {
		return nil, errors.Annotate(err, "invalid invocation name %q", invocationID).Err()
	}
	propertiesJSON, err := MarshalStructPB(invocation.Properties)
	if err != nil {
		return nil, errors.Annotate(err, "marshal properties").Err()
	}

	return &bqpb.TestVerdictRow_InvocationRecord{
		Id:         invocationID,
		Tags:       pbutil.StringPairFromResultDB(invocation.Tags),
		Realm:      invocation.Realm,
		Properties: propertiesJSON,
	}, nil
}

func exoneration(exoneration *rdbpb.TestExoneration) *bqpb.TestVerdictRow_Exoneration {
	return &bqpb.TestVerdictRow_Exoneration{
		ExplanationHtml: exoneration.ExplanationHtml,
		Reason:          pbutil.ExonerationReasonFromResultDB(exoneration.Reason),
	}
}

func counts(results []*bqpb.TestVerdictRow_TestResult) *bqpb.TestVerdictRow_Counts {
	counts := &bqpb.TestVerdictRow_Counts{}
	for _, result := range results {
		counts.Total += 1
		if result.Status != pb.TestResultStatus_SKIP {
			counts.TotalNonSkipped += 1
		}
		if !result.Expected {
			counts.Unexpected += 1
			if result.Status != pb.TestResultStatus_SKIP {
				counts.UnexpectedNonSkipped += 1
				if result.Status != pb.TestResultStatus_PASS {
					counts.UnexpectedNonSkippedNonPassed += 1
				}
			}
		}
	}
	return counts
}

func changeVerifierRun(cv *controlpb.PresubmitResult) *bqpb.TestVerdictRow_ChangeVerifierRun {
	return &bqpb.TestVerdictRow_ChangeVerifierRun{
		Id:              cv.PresubmitRunId.Id,
		Mode:            cv.Mode,
		Status:          analysis.ToBQPresubmitRunStatus(cv.Status),
		IsBuildCritical: cv.Critical,
	}
}

func buildbucketBuild(build *controlpb.BuildResult) *bqpb.TestVerdictRow_BuildbucketBuild {
	return &bqpb.TestVerdictRow_BuildbucketBuild{
		Id: build.Id,
		Builder: &bqpb.TestVerdictRow_BuildbucketBuild_Builder{
			Project: build.Project,
			Bucket:  build.Bucket,
			Builder: build.Builder,
		},
		Status:            analysis.ToBQBuildStatus(build.Status),
		GardenerRotations: build.GardenerRotations,
	}
}

func result(result *rdbpb.TestResult) (*bqpb.TestVerdictRow_TestResult, error) {
	propertiesJSON, err := MarshalStructPB(result.Properties)
	if err != nil {
		return nil, errors.Annotate(err, "marshal properties").Err()
	}
	invID, err := resultdb.InvocationFromTestResultName(result.Name)
	if err != nil {
		return nil, errors.Annotate(err, "invocation from test result name").Err()
	}
	tr := &bqpb.TestVerdictRow_TestResult{
		Parent: &bqpb.TestVerdictRow_ParentInvocationRecord{
			Id: invID,
		},
		Name:        result.Name,
		ResultId:    result.ResultId,
		Expected:    result.Expected,
		Status:      pbutil.TestResultStatusFromResultDB(result.Status),
		SummaryHtml: result.SummaryHtml,
		StartTime:   result.StartTime,
		// Null durations are represented as zeroes in the export.
		// Unfortunately, BigQuery Write API does not offer a way for us
		// to write NULL to a NULLABLE FLOAT column.
		Duration:      result.Duration.AsDuration().Seconds(),
		Tags:          pbutil.StringPairFromResultDB(result.Tags),
		FailureReason: pbutil.FailureReasonFromResultDB(result.FailureReason),
		Properties:    propertiesJSON,
	}

	skipReason := pbutil.SkipReasonFromResultDB(result.SkipReason)
	if skipReason != pb.SkipReason_SKIP_REASON_UNSPECIFIED {
		tr.SkipReason = skipReason.String()
	}

	return tr, nil
}

// variantJSON returns the JSON equivalent for a variant.
// Each key in the variant is mapped to a top-level key in the
// JSON object.
// e.g. `{"builder":"linux-rel","os":"Ubuntu-18.04"}`
func variantJSON(variant *rdbpb.Variant) (string, error) {
	return pbutil.VariantToJSON(pbutil.VariantFromResultDB(variant))
}

// MarshalStructPB serialises a structpb.Struct as a JSONPB.
func MarshalStructPB(s *structpb.Struct) (string, error) {
	if s == nil {
		// There is no string value we can send to BigQuery that will
		// interpret as a NULL value for a JSON column:
		// - "" (empty string) is rejected as invalid JSON.
		// - "null" is interpreted as the JSON value null, not the
		//   absence of a value.
		// Consequently, the next best thing is to return an empty
		// JSON object.
		return pbutil.EmptyJSON, nil
	}
	// Structs are persisted as JSONPB strings.
	// See also https://bit.ly/chromium-bq-struct
	b, err := (&protojson.MarshalOptions{}).Marshal(s)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
