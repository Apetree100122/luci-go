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

package bqexporter

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"

	"cloud.google.com/go/bigquery"
	"github.com/golang/protobuf/descriptor"
	desc "github.com/golang/protobuf/protoc-gen-go/descriptor"
	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/runtime/protoiface"

	"go.chromium.org/luci/common/bq"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/server/span"

	"go.chromium.org/luci/resultdb/internal/artifactcontent"
	"go.chromium.org/luci/resultdb/internal/artifacts"
	"go.chromium.org/luci/resultdb/internal/invocations"
	"go.chromium.org/luci/resultdb/pbutil"
	bqpb "go.chromium.org/luci/resultdb/proto/bq"
	pb "go.chromium.org/luci/resultdb/proto/v1"
)

var textArtifactRowSchema bigquery.Schema

var lineBreak = []byte("\n")

const (
	artifactRowMessage = "luci.resultdb.bq.TextArtifactRow"

	// Row size limit is 5MB according to
	// https://cloud.google.com/bigquery/quotas#streaming_inserts
	// Split artifact content into 4MB shards if it's too large.
	contentShardSize = 4e6

	// Number of workers to download artifact content.
	artifactWorkers = 10
)

func init() {
	var err error
	if textArtifactRowSchema, err = generateArtifactRowSchema(); err != nil {
		panic(err)
	}
}

func generateArtifactRowSchema() (schema bigquery.Schema, err error) {
	fd, _ := descriptor.MessageDescriptorProto(&bqpb.TextArtifactRow{})
	fdinv, _ := descriptor.MessageDescriptorProto(&bqpb.InvocationRecord{})
	fdsp, _ := descriptor.MessageDescriptorProto(&pb.StringPair{})
	fdset := &desc.FileDescriptorSet{File: []*desc.FileDescriptorProto{fd, fdinv, fdsp}}
	return generateSchema(fdset, artifactRowMessage)
}

// textArtifactRowInput is information required to generate a text artifact BigQuery row.
type textArtifactRowInput struct {
	exported *pb.Invocation
	parent   *pb.Invocation
	a        *pb.Artifact
	shardID  int32
	content  string
}

func (i *textArtifactRowInput) row() protoiface.MessageV1 {
	_, testID, resultID, artifactID := artifacts.MustParseName(i.a.Name)
	expRec := invocationProtoToRecord(i.exported)
	parRect := invocationProtoToRecord(i.parent)

	return &bqpb.TextArtifactRow{
		Exported:   expRec,
		Parent:     parRect,
		TestId:     testID,
		ResultId:   resultID,
		ArtifactId: artifactID,
		ShardId:    i.shardID,
		Content:    i.content,
	}
}

func (i *textArtifactRowInput) id() []byte {
	return []byte(fmt.Sprintf("%s/%d", i.a.Name, i.shardID))
}

func (b *bqExporter) downloadArtifactContent(ctx context.Context, a *artifact, rowC chan rowInput) error {
	ac := artifactcontent.Reader{
		RBEInstance: b.Options.ArtifactRBEInstance,
		Hash:        a.RBECASHash,
		Size:        a.SizeBytes,
	}

	var str strings.Builder
	shardId := 0
	input := func() *textArtifactRowInput {
		return &textArtifactRowInput{
			exported: a.exported,
			parent:   a.parent,
			a:        a.Artifact.Artifact,
			shardID:  int32(shardId),
			content:  str.String(),
		}
	}

	return ac.DownloadRBECASContent(ctx, b.rbecasClient, func(pr io.Reader) error {
		sc := bufio.NewScanner(pr)
		for sc.Scan() {
			// TODO(crbug.com/1149736): handle the case that a single line is 5MB+.
			// If such case happens, we should split the line.
			if str.Len()+len(sc.Bytes())+len(lineBreak) > contentShardSize {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case rowC <- input():
				}
				shardId++
				str.Reset()
			}
			str.Write(sc.Bytes())
			str.Write(lineBreak)
		}
		if str.Len() > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case rowC <- input():
			}
		}
		return nil
	})
}

type artifact struct {
	*artifacts.Artifact
	exported *pb.Invocation
	parent   *pb.Invocation
}

func (b *bqExporter) queryTextArtifacts(ctx context.Context, exportedID invocations.ID, bqExport *pb.BigQueryExport, artifactC chan *artifact) error {
	invIDs, err := getInvocationIDSet(ctx, exportedID)
	if err != nil {
		return err
	}

	q := artifacts.Query{
		InvocationIDs:       invIDs,
		TestResultPredicate: bqExport.GetTextArtifacts().GetPredicate().GetTestResultPredicate(),
		ContentTypeRegexp:   bqExport.GetTextArtifacts().GetPredicate().GetContentTypeRegexp(),
		WithRBECASHash:      true,
	}

	invs, err := invocations.ReadBatch(ctx, q.InvocationIDs)
	if err != nil {
		return err
	}

	return q.Run(ctx, func(a *artifacts.Artifact) error {
		invID, _, _, _ := artifacts.MustParseName(a.Name)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case artifactC <- &artifact{Artifact: a, exported: invs[exportedID], parent: invs[invID]}:
		}
		return nil
	})
}

func (b *bqExporter) artifactRowInputToBatch(ctx context.Context, rowC chan rowInput, batchC chan []rowInput) error {
	rows := make([]rowInput, 0, b.MaxBatchRowCount)
	batchSize := 0 // Estimated size of rows in bytes.
	for row := range rowC {
		contentLength := len(row.(*textArtifactRowInput).content)
		if len(rows)+1 >= b.MaxBatchRowCount || batchSize+contentLength >= b.MaxBatchSizeApprox {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case batchC <- rows:
			}
			rows = make([]rowInput, 0, b.MaxBatchRowCount)
			batchSize = 0
		}
		rows = append(rows, row)
		batchSize += contentLength
	}
	if len(rows) > 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case batchC <- rows:
		}
	}
	return nil
}

// exportTextArtifactsToBigQuery queries text artifacts in Spanner then exports them to BigQuery.
func (b *bqExporter) exportTextArtifactsToBigQuery(ctx context.Context, ins inserter, invID invocations.ID, bqExport *pb.BigQueryExport) error {
	ctx, cancel := span.ReadOnlyTransaction(ctx)
	defer cancel()

	// Query artifacts and export to BigQuery.
	batchC := make(chan []rowInput)
	rowC := make(chan rowInput)
	artifactC := make(chan *artifact, artifactWorkers)

	// Batch exports rows to BigQuery.
	eg, ctx := errgroup.WithContext(ctx)

	eg.Go(func() error {
		return b.batchExportRows(ctx, ins, batchC, func(ctx context.Context, err bigquery.PutMultiError, rows []*bq.Row) {
			// Print up to 10 errors.
			for i := 0; i < 10 && i < len(err); i++ {
				a := rows[err[i].RowIndex].Message.(*bqpb.TextArtifactRow)
				var artifactName string
				if a.TestId != "" {
					artifactName = pbutil.TestResultArtifactName(a.Parent.Id, a.TestId, a.ResultId, a.ArtifactId)
				} else {
					artifactName = pbutil.InvocationArtifactName(a.Parent.Id, a.ArtifactId)
				}
				logging.Errorf(ctx, "failed to insert row for %s: %s", artifactName, err[i].Error())
			}
			if len(err) > 10 {
				logging.Errorf(ctx, "%d more row insertions failed", len(err)-10)
			}
		})
	})

	eg.Go(func() error {
		defer close(batchC)
		return b.artifactRowInputToBatch(ctx, rowC, batchC)
	})

	eg.Go(func() error {
		defer close(rowC)

		subEg, ctx := errgroup.WithContext(ctx)
		for w := 0; w < artifactWorkers; w++ {
			subEg.Go(func() error {
				for a := range artifactC {
					if err := b.downloadArtifactContent(ctx, a, rowC); err != nil {
						return err
					}
				}
				return nil
			})
		}
		return subEg.Wait()
	})

	eg.Go(func() error {
		defer close(artifactC)
		return b.queryTextArtifacts(ctx, invID, bqExport, artifactC)
	})

	return eg.Wait()
}
