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

package main

import (
	"context"

	"go.chromium.org/luci/common/gcloud/gs"
	log "go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/logdog/api/config/svcconfig"
	"go.chromium.org/luci/logdog/server/archivist"
	"go.chromium.org/luci/logdog/server/config"
)

// GetSettingsLoader is an archivist.SettingsLoader implementation that merges
// global and project-specific settings.
//
// The resulting settings object will be verified by the Archivist.
func GetSettingsLoader(serviceID string, acfg *svcconfig.Archivist) archivist.SettingsLoader {
	return func(c context.Context, project string) (*archivist.Settings, error) {
		// If the project config of a task no longer exists, this function is expected
		// to return config.ErrNoConfig. Then, the archivist task handler will discard
		// the task w/o returning an error.
		//
		// Please ensure that config.ProjectConfig() is called first to prevent other
		// errors from hiding config.ErrNoConfig.
		pcfg, err := config.ProjectConfig(c, project)
		if err != nil {
			log.Fields{
				log.ErrorKey: err,
				"project":    project,
			}.Errorf(c, "Failed to fetch project configuration.")
			return nil, err
		}

		indexParam := func(get func(ic *svcconfig.ArchiveIndexConfig) int32) int {
			if ic := pcfg.ArchiveIndexConfig; ic != nil {
				if v := get(ic); v > 0 {
					return int(v)
				}
			}

			if ic := acfg.ArchiveIndexConfig; ic != nil {
				if v := get(ic); v > 0 {
					return int(v)
				}
			}

			return 0
		}

		// Load our base settings.
		//
		// Archival bases are:
		// Staging: gs://<services:gs_staging_bucket>/<project-id>/...
		// Archive: gs://<project:archive_gs_bucket>/<project-id>/...
		st := archivist.Settings{
			GSBase:        gs.MakePath(pcfg.ArchiveGsBucket, "").Concat(serviceID),
			GSStagingBase: gs.MakePath(acfg.GsStagingBucket, "").Concat(serviceID),

			IndexStreamRange: indexParam(func(ic *svcconfig.ArchiveIndexConfig) int32 { return ic.StreamRange }),
			IndexPrefixRange: indexParam(func(ic *svcconfig.ArchiveIndexConfig) int32 { return ic.PrefixRange }),
			IndexByteRange:   indexParam(func(ic *svcconfig.ArchiveIndexConfig) int32 { return ic.ByteRange }),

			CloudLoggingProjectID:        pcfg.CloudLoggingConfig.GetDestination(),
			CloudLoggingWithProjectScope: !pcfg.CloudLoggingConfig.GetUseGlobalLogdogAccount(),
		}

		// Fold project settings into loaded ones.
		return &st, nil
	}
}
