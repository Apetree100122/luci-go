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

package monorail

import (
	"github.com/golang/protobuf/proto"

	mpb "go.chromium.org/luci/analysis/internal/bugs/monorail/api_proto"

	configpb "go.chromium.org/luci/analysis/proto/config"
)

// ChromiumTestPriorityField is the resource name of the priority field
// that is consistent with ChromiumTestConfig.
const ChromiumTestPriorityField = "projects/chromium/fieldDefs/11"

// ChromiumTestTypeField is the resource name of the type field
// that is consistent with ChromiumTestConfig.
const ChromiumTestTypeField = "projects/chromium/fieldDefs/10"

// ChromiumTestConfig provides chromium-like configuration for tests
// to use.
func ChromiumTestConfig() *configpb.MonorailProject {
	projectCfg := &configpb.MonorailProject{
		Project: "chromium",
		DefaultFieldValues: []*configpb.MonorailFieldValue{
			{
				FieldId: 10,
				Value:   "Bug",
			},
		},
		PriorityFieldId: 11,
		Priorities: []*configpb.MonorailPriority{
			{
				Priority: "0",
				Threshold: &configpb.ImpactThreshold{
					TestResultsFailed: &configpb.MetricThreshold{
						OneDay: proto.Int64(1000),
					},
					TestRunsFailed: &configpb.MetricThreshold{
						OneDay: proto.Int64(100),
					},
				},
			},
			{
				Priority: "1",
				Threshold: &configpb.ImpactThreshold{
					TestResultsFailed: &configpb.MetricThreshold{
						OneDay: proto.Int64(500),
					},
					TestRunsFailed: &configpb.MetricThreshold{
						OneDay: proto.Int64(50),
					},
				},
			},
			{
				Priority: "2",
				Threshold: &configpb.ImpactThreshold{
					TestResultsFailed: &configpb.MetricThreshold{
						OneDay: proto.Int64(100),
					},
					TestRunsFailed: &configpb.MetricThreshold{
						OneDay: proto.Int64(10),
					},
				},
			},
			{
				Priority: "3",
				// Should be less onerous than the bug-filing thresholds
				// used in BugUpdater tests, to avoid bugs that were filed
				// from being immediately closed.
				Threshold: &configpb.ImpactThreshold{
					TestResultsFailed: &configpb.MetricThreshold{
						OneDay:   proto.Int64(50),
						ThreeDay: proto.Int64(300),
						SevenDay: proto.Int64(1), // Set to 1 so that we check hysteresis never rounds down to 0 and prevents bugs from closing.
					},
				},
			},
		},
		PriorityHysteresisPercent: 10,
	}
	return projectCfg
}

// ChromiumTestIssuePriority returns the priority of an issue, assuming
// it has been created consistent with ChromiumTestConfig.
func ChromiumTestIssuePriority(issue *mpb.Issue) string {
	for _, fv := range issue.FieldValues {
		if fv.Field == ChromiumTestPriorityField {
			return fv.Value
		}
	}
	return ""
}
