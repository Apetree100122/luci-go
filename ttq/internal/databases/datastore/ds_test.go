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

package datastore

import (
	"context"
	"testing"
	"time"

	"go.chromium.org/gae/impl/memory"
	ds "go.chromium.org/gae/service/datastore"

	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/common/logging/gologger"
	"go.chromium.org/luci/ttq/internal/reminder"
	ttqtesting "go.chromium.org/luci/ttq/internal/testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestAcceptance(t *testing.T) {
	ctx := memory.Use(context.Background())
	if testing.Verbose() {
		ctx = gologger.StdConfig.Use(ctx)
		ctx = logging.SetLevel(ctx, logging.Debug)
	}

	ds.GetTestable(ctx).Consistent(true)
	ttqtesting.RunDBAcceptance(ctx, &DB{}, t)
}

func TestAcceptablePrecision(t *testing.T) {
	t.Parallel()

	Convey("ds supports up to Microsecond precision", t, func() {
		So(reminder.FreshUntilPrecision, ShouldBeGreaterThanOrEqualTo, time.Microsecond)
	})
}
