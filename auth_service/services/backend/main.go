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

// Package main is the main point of entry for the backend module.
//
// It handles task queue tasks and cron jobs.
package main

import (
	// Ensure registration of validation rules.
	// NOTE: this must go before anything that depends on validation globals,
	// e.g. cfgcache.Register in srvcfg files in allowlistcfg/ or oauthcfg/.
	"go.chromium.org/luci/auth_service/internal/configs/validation"

	"context"

	"go.chromium.org/luci/server"
	"go.chromium.org/luci/server/cron"
	"go.chromium.org/luci/server/module"

	"go.chromium.org/luci/auth_service/impl"
	"go.chromium.org/luci/auth_service/impl/model"
	"go.chromium.org/luci/auth_service/internal/configs/srvcfg/allowlistcfg"
	"go.chromium.org/luci/auth_service/internal/configs/srvcfg/oauthcfg"
	"go.chromium.org/luci/auth_service/internal/configs/srvcfg/securitycfg"
)

func main() {
	modules := []module.Module{
		cron.NewModuleFromFlags(),
	}

	impl.Main(modules, func(srv *server.Server) error {
		cron.RegisterHandler("update-config", func(ctx context.Context) error {
			// ip_allowlist.cfg handling.
			if err := allowlistcfg.Update(ctx); err != nil {
				return err
			}
			cfg, err := allowlistcfg.Get(ctx)
			if err != nil {
				return err
			}
			subnets, err := validation.GetSubnets(cfg.IpAllowlists)
			if err != nil {
				return err
			}
			if err := model.UpdateAllowlistEntities(ctx, subnets, true); err != nil {
				return err
			}

			// oauth.cfg handling.
			if err := oauthcfg.Update(ctx); err != nil {
				return err
			}
			oauthcfg, err := oauthcfg.Get(ctx)
			if err != nil {
				return err
			}

			// security.cfg handling.
			if err := securitycfg.Update(ctx); err != nil {
				return err
			}
			securitycfg, err := securitycfg.Get(ctx)
			if err != nil {
				return err
			}

			if err := model.UpdateAuthGlobalConfig(ctx, oauthcfg, securitycfg, true); err != nil {
				return err
			}
			return nil
		})
		return nil
	})
}
