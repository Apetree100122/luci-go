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

package run

import (
	"fmt"

	bqpb "go.chromium.org/luci/cv/api/bigquery/v1"
)

// Mode dictates the behavior of this Run.
//
// The end goal is to have arbitrary user-defined Mode names.
// For now, CQDaemon/LUCI CV operates with just 3 pre-defined modes,
// whose values are fixed based on legacy CQ BQ export.
type Mode string

const (
	// DryRun triggers configured Tryjobs, but doesn't submit.
	DryRun Mode = "DRY_RUN"
	// FullRun is DryRun followed by submit.
	FullRun Mode = "FULL_RUN"
	// QuickDryRun is like DryRun but different thus allowing either different or
	// faster yet less thorough Tryjobs.
	QuickDryRun Mode = "QUICK_DRY_RUN"
)

// BQAttemptMode returns corresponding value for legacy CQ BQ export.
func (m Mode) BQAttemptMode() bqpb.Mode {
	switch m {
	case DryRun:
		return bqpb.Mode_DRY_RUN
	case FullRun:
		return bqpb.Mode_FULL_RUN
	case QuickDryRun:
		return bqpb.Mode_QUICK_DRY_RUN
	default:
		panic(fmt.Sprintf("unknown RunMode %q", m))
	}
}

// ModeFromBQAttempt returns Mode from CQ BQ export.
func ModeFromBQAttempt(m bqpb.Mode) (Mode, error) {
	switch m {
	case bqpb.Mode_DRY_RUN:
		return DryRun, nil
	case bqpb.Mode_FULL_RUN:
		return FullRun, nil
	case bqpb.Mode_QUICK_DRY_RUN:
		return QuickDryRun, nil
	default:
		return "", fmt.Errorf("unknown Attempt mode %d %s", m, m)
	}
}
