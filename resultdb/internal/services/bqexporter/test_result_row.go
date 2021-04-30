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

package bqexporter

import (
	"context"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/spanner"
	"github.com/golang/protobuf/descriptor"
	desc "github.com/golang/protobuf/protoc-gen-go/descriptor"
	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/runtime/protoiface"

	"go.chromium.org/luci/common/bq"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/server/span"

	"go.chromium.org/luci/resultdb/internal/invocations"
	"go.chromium.org/luci/resultdb/internal/spanutil"
	"go.chromium.org/luci/resultdb/internal/testresults"
	"go.chromium.org/luci/resultdb/pbutil"
	bqpb "go.chromium.org/luci/resultdb/proto/bq"
	pb "go.chromium.org/luci/resultdb/proto/v1"
)

var testResultRowSchema bigquery.Schema

const testResultRowMessage = "luci.resultdb.bq.TestResultRow"

func init() {
	var err error
	if testResultRowSchema, err = generateTestResultRowSchema(); err != nil {
		panic(err)
	}
}

func generateTestResultRowSchema() (schema bigquery.Schema, err error) {
	fd, _ := descriptor.MessageDescriptorProto(&bqpb.TestResultRow{})
	// We also need to get FileDescriptorProto for StringPair and TestMetadata
	// because they are defined in different files.
	fdsp, _ := descriptor.MessageDescriptorProto(&pb.StringPair{})
	fdtmd, _ := descriptor.MessageDescriptorProto(&pb.TestMetadata{})
	fdinv, _ := descriptor.MessageDescriptorProto(&bqpb.InvocationRecord{})
	fdset := &desc.FileDescriptorSet{File: []*desc.FileDescriptorProto{fd, fdsp, fdtmd, fdinv}}
	return generateSchema(fdset, testResultRowMessage)
}

// Row size limit is 5MB according to
// https://cloud.google.com/bigquery/quotas#streaming_inserts
// Cap the summaryHTML's length to 4MB to ensure the row size is under
// limit.
const maxSummaryLength = 4e6

func invocationProtoToRecord(inv *pb.Invocation) *bqpb.InvocationRecord {
	return &bqpb.InvocationRecord{
		Id:    string(invocations.MustParseName(inv.Name)),
		Tags:  inv.Tags,
		Realm: inv.Realm,
	}
}

// testResultRowInput is information required to generate a TestResult BigQuery row.
type testResultRowInput struct {
	exported   *pb.Invocation
	parent     *pb.Invocation
	tr         *pb.TestResult
	exonerated bool
}

func (i *testResultRowInput) row() protoiface.MessageV1 {
	tr := i.tr

	ret := &bqpb.TestResultRow{
		Exported:      invocationProtoToRecord(i.exported),
		Parent:        invocationProtoToRecord(i.parent),
		Name:          tr.Name,
		TestId:        tr.TestId,
		ResultId:      tr.ResultId,
		Variant:       pbutil.VariantToStringPairs(tr.Variant),
		VariantHash:   tr.VariantHash,
		Expected:      tr.Expected,
		Status:        tr.Status.String(),
		SummaryHtml:   tr.SummaryHtml,
		StartTime:     tr.StartTime,
		Duration:      tr.Duration,
		Tags:          tr.Tags,
		Exonerated:    i.exonerated,
		PartitionTime: i.exported.CreateTime,
		TestMetadata:  tr.TestMetadata,
	}

	if len(ret.SummaryHtml) > maxSummaryLength {
		ret.SummaryHtml = "[Trimmed] " + ret.SummaryHtml[:maxSummaryLength]
	}

	return ret
}

func (i *testResultRowInput) id() []byte {
	return []byte(i.tr.Name)
}

type testVariantKey struct {
	testID      string
	variantHash string
}

// queryExoneratedTestVariants reads exonerated test variants matching the predicate.
func queryExoneratedTestVariants(ctx context.Context, invIDs invocations.IDSet) (map[testVariantKey]struct{}, error) {
	st := spanner.NewStatement(`
		SELECT DISTINCT TestId, VariantHash,
		FROM TestExonerations
		WHERE InvocationId IN UNNEST(@invIDs)
	`)
	st.Params["invIDs"] = invIDs
	tvs := map[testVariantKey]struct{}{}
	var b spanutil.Buffer
	err := spanutil.Query(ctx, st, func(row *spanner.Row) error {
		var key testVariantKey
		if err := b.FromSpanner(row, &key.testID, &key.variantHash); err != nil {
			return err
		}
		tvs[key] = struct{}{}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return tvs, nil
}

func (b *bqExporter) queryTestResults(
	ctx context.Context,
	exportedID invocations.ID,
	q testresults.Query,
	exoneratedTestVariants map[testVariantKey]struct{},
	batchC chan []rowInput) error {

	invs, err := invocations.ReadBatch(ctx, q.InvocationIDs)
	if err != nil {
		return err
	}

	rows := make([]rowInput, 0, b.MaxBatchRowCount)
	batchSize := 0 // Estimated size of rows in bytes.
	rowCount := 0
	err = q.Run(ctx, func(tr *pb.TestResult) error {
		_, exonerated := exoneratedTestVariants[testVariantKey{testID: tr.TestId, variantHash: tr.VariantHash}]
		parentID, _, _ := testresults.MustParseName(tr.Name)
		rows = append(rows, &testResultRowInput{
			exported:   invs[exportedID],
			parent:     invs[parentID],
			tr:         tr,
			exonerated: exonerated,
		})
		batchSize += proto.Size(tr)
		rowCount++
		if len(rows) >= b.MaxBatchRowCount || batchSize >= b.MaxBatchSizeApprox {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case batchC <- rows:
			}
			rows = make([]rowInput, 0, b.MaxBatchRowCount)
			batchSize = 0
		}
		return nil
	})

	if err != nil {
		return err
	}

	if len(rows) > 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case batchC <- rows:
		}
	}

	// Log the number of fetched rows so that later we can compare it to
	// the value in QueryTestResultsStatistics. This is to help debugging
	// crbug.com/1090671.
	logging.Debugf(ctx, "fetched %d rows for invocations %q", rowCount, q.InvocationIDs)
	return nil
}

// exportTestResultsToBigQuery queries test results in Spanner then exports them to BigQuery.
func (b *bqExporter) exportTestResultsToBigQuery(ctx context.Context, ins inserter, invID invocations.ID, bqExport *pb.BigQueryExport) error {
	ctx, cancel := span.ReadOnlyTransaction(ctx)
	defer cancel()

	// Get the invocation set.
	invIDs, err := getInvocationIDSet(ctx, invID)
	if err != nil {
		return errors.Annotate(err, "invocation id set").Err()
	}

	exoneratedTestVariants, err := queryExoneratedTestVariants(ctx, invIDs)
	if err != nil {
		return errors.Annotate(err, "query exoneration").Err()
	}

	// Query test results and export to BigQuery.
	batchC := make(chan []rowInput)

	// Batch exports rows to BigQuery.
	eg, ctx := errgroup.WithContext(ctx)

	eg.Go(func() error {
		return b.batchExportRows(ctx, ins, batchC, func(ctx context.Context, err bigquery.PutMultiError, rows []*bq.Row) {
			// Print up to 10 errors.
			for i := 0; i < 10 && i < len(err); i++ {
				tr := rows[err[i].RowIndex].Message.(*bqpb.TestResultRow)
				logging.Errorf(ctx, "failed to insert row for %s: %s", pbutil.TestResultName(tr.Parent.Id, tr.TestId, tr.ResultId), err[i].Error())
			}
			if len(err) > 10 {
				logging.Errorf(ctx, "%d more row insertions failed", len(err)-10)
			}
		})
	})

	q := testresults.Query{
		Predicate:     bqExport.GetTestResults().GetPredicate(),
		InvocationIDs: invIDs,
		Mask:          testresults.AllFields,
	}
	eg.Go(func() error {
		defer close(batchC)
		return b.queryTestResults(ctx, invID, q, exoneratedTestVariants, batchC)
	})

	return eg.Wait()
}
