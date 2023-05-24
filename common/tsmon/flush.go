// Copyright 2015 The LUCI Authors.
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

package tsmon

import (
	"context"
	"time"

	"go.chromium.org/luci/common/clock"
	"go.chromium.org/luci/common/logging"
)

// Flush sends all the metrics that are registered in the application.
func Flush(ctx context.Context) error {
	return GetState(ctx).Flush(ctx, nil)
}

// autoFlusher knows how to periodically call 'Flush'.
type autoFlusher struct {
	killed chan struct{}
	cancel context.CancelFunc

	flush func(context.Context) error // mocked in unit tests
}

func (f *autoFlusher) start(ctx context.Context, interval time.Duration) {
	flush := f.flush
	if flush == nil {
		flush = Flush
	}

	// 'killed' is closed when timer goroutine exits.
	killed := make(chan struct{})
	f.killed = killed

	ctx, f.cancel = context.WithCancel(ctx)
	go func() {
		defer close(killed)

		for {
			if tr := <-clock.After(ctx, interval); tr.Incomplete() {
				return
			}
			if err := flush(ctx); err != nil && err != context.Canceled {
				logging.Warningf(ctx, "Failed to flush tsmon metrics: %v", err)
			}
		}
	}()
}

func (f *autoFlusher) stop() {
	f.cancel()
	<-f.killed
	f.cancel = nil
	f.killed = nil
}
