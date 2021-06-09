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

package bq

import (
	"context"

	"google.golang.org/protobuf/proto"

	"go.chromium.org/luci/server/tq"

	cvbq "go.chromium.org/luci/cv/internal/bq"
	"go.chromium.org/luci/cv/internal/common"
)

const exportRunToBQTaskClass = "bq-export"

// Exporter sends finished Run data to BigQuery.
type Exporter struct {
	tqd *tq.Dispatcher
	bqc cvbq.Client
}

// NewExporter creates a new Exporter, registering it in the given TQ
// dispatcher.
func NewExporter(tqd *tq.Dispatcher, bqc cvbq.Client) *Exporter {
	exporter := &Exporter{tqd, bqc}
	tqd.RegisterTaskClass(tq.TaskClass{
		ID:        exportRunToBQTaskClass,
		Prototype: &ExportRunToBQTask{},
		Queue:     "bq-export",
		Quiet:     true,
		// BQ Export should be done in a transaction, because we want
		// BQ exported if and only if Run Status is changed in datastore.
		Kind: tq.Transactional,
		Handler: func(ctx context.Context, payload proto.Message) error {
			task := payload.(*ExportRunToBQTask)
			err := send(ctx, bqc, common.RunID(task.RunId))
			return common.TQifyError(ctx, err)
		},
	})
	return exporter
}

// Schedule enqueues a task to send a row to BQ for a Run.
func (s *Exporter) Schedule(ctx context.Context, runID common.RunID) error {
	return s.tqd.AddTask(ctx, &tq.Task{Payload: &ExportRunToBQTask{RunId: string(runID)}})
}
