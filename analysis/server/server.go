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

// Package server contains shared server initialisation logic for
// LUCI Analysis services.
package server

import (
	"context"

	"go.chromium.org/luci/analysis/app"
	"go.chromium.org/luci/analysis/internal/admin"
	adminpb "go.chromium.org/luci/analysis/internal/admin/proto"
	"go.chromium.org/luci/analysis/internal/analysis"
	"go.chromium.org/luci/analysis/internal/analyzedtestvariants"
	"go.chromium.org/luci/analysis/internal/bugs/updater"
	"go.chromium.org/luci/analysis/internal/clustering/reclustering/orchestrator"
	"go.chromium.org/luci/analysis/internal/config"
	"go.chromium.org/luci/analysis/internal/legacydb"
	"go.chromium.org/luci/analysis/internal/metrics"
	"go.chromium.org/luci/analysis/internal/services/reclustering"
	"go.chromium.org/luci/analysis/internal/services/resultcollector"
	"go.chromium.org/luci/analysis/internal/services/resultingester"
	"go.chromium.org/luci/analysis/internal/services/testvariantbqexporter"
	"go.chromium.org/luci/analysis/internal/services/testvariantupdator"
	"go.chromium.org/luci/analysis/internal/span"
	analysispb "go.chromium.org/luci/analysis/proto/v1"
	"go.chromium.org/luci/analysis/rpc"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/config/server/cfgmodule"
	"go.chromium.org/luci/grpc/prpc"
	luciserver "go.chromium.org/luci/server"
	"go.chromium.org/luci/server/auth"
	"go.chromium.org/luci/server/cron"
	"go.chromium.org/luci/server/encryptedcookies"
	"go.chromium.org/luci/server/gaeemulation"
	"go.chromium.org/luci/server/module"
	"go.chromium.org/luci/server/secrets"
	spanmodule "go.chromium.org/luci/server/span"
	"go.chromium.org/luci/server/tq"
)

// Main implements the common entrypoint for all LUCI Analysis GAE services.
// All LUCI Analysis GAE services have the code necessary to serve all pRPCs,
// crons and task queues. The only thing that is not shared is frontend
// handling, due to the fact this service requires other assets (javascript,
// files) to be deployed.
//
// Allowing all services to serve everything (except frontend) minimises
// the need to keep server code in sync with changes with dispatch.yaml.
// Moreover, dispatch.yaml changes are not deployed atomically
// with service changes, so this avoids transient traffic rejection during
// rollout of new LUCI Analysis versions that switch handling of endpoints
// between services.
func Main(init func(srv *luciserver.Server) error) {
	// Use the same modules for all LUCI Analysis services.
	modules := []module.Module{
		cfgmodule.NewModuleFromFlags(),
		cron.NewModuleFromFlags(),
		encryptedcookies.NewModuleFromFlags(), // Required for auth sessions.
		gaeemulation.NewModuleFromFlags(),     // Needed by cfgmodule.
		secrets.NewModuleFromFlags(),          // Needed by encryptedcookies.
		spanmodule.NewModuleFromFlags(),
		legacydb.NewModuleFromFlags(),
		tq.NewModuleFromFlags(),
	}
	luciserver.Main(nil, modules, func(srv *luciserver.Server) error {
		// Register pPRC servers.
		srv.PRPC.AccessControl = prpc.AllowOriginAll
		srv.PRPC.Authenticator = &auth.Authenticator{
			Methods: []auth.Method{
				&auth.GoogleOAuth2Method{
					Scopes: []string{"https://www.googleapis.com/auth/userinfo.email"},
				},
			},
		}
		// TODO(crbug/1082369): Remove this workaround once field masks can be decoded.
		srv.PRPC.HackFixFieldMasksForJSON = true
		srv.RegisterUnaryServerInterceptor(span.SpannerDefaultsInterceptor())

		ac, err := analysis.NewClient(srv.Context, srv.Options.CloudProject)
		if err != nil {
			return errors.Annotate(err, "creating analysis client").Err()
		}

		analysispb.RegisterClustersServer(srv.PRPC, rpc.NewClustersServer(ac))
		analysispb.RegisterRulesServer(srv.PRPC, rpc.NewRulesSever())
		analysispb.RegisterProjectsServer(srv.PRPC, rpc.NewProjectsServer())
		analysispb.RegisterInitDataGeneratorServer(srv.PRPC, rpc.NewInitDataGeneratorServer())
		analysispb.RegisterTestVariantsServer(srv.PRPC, rpc.NewTestVariantsServer())
		adminpb.RegisterAdminServer(srv.PRPC, admin.CreateServer())

		// Test History service needs to connect back to an old Spanner
		// database to service some queries.
		legacyCl := legacydb.LegacyClient(srv.Context)
		installOldDatabase := func(ctx context.Context) context.Context {
			if legacyCl != nil {
				// Route queries to the old database to the old database.
				return spanmodule.UseClient(ctx, legacyCl)
			}
			// Route queries in the time range of the old database
			// to the old database.
			return ctx
		}
		analysispb.RegisterTestHistoryServer(srv.PRPC, rpc.NewTestHistoryServer(installOldDatabase))

		// GAE crons.
		updateAnalysisAndBugsHandler := updater.NewHandler(srv.Options.CloudProject, srv.Options.Prod)
		cron.RegisterHandler("update-analysis-and-bugs", updateAnalysisAndBugsHandler.CronHandler)
		cron.RegisterHandler("read-config", config.Update)
		cron.RegisterHandler("export-test-variants", testvariantbqexporter.ScheduleTasks)
		cron.RegisterHandler("purge-test-variants", analyzedtestvariants.Purge)
		cron.RegisterHandler("reclustering", orchestrator.CronHandler)
		cron.RegisterHandler("global-metrics", metrics.GlobalMetrics)

		// Pub/Sub subscription endpoints.
		srv.Routes.POST("/_ah/push-handlers/buildbucket", nil, app.BuildbucketPubSubHandler)
		srv.Routes.POST("/_ah/push-handlers/cvrun", nil, app.CVRunPubSubHandler)

		// Register task queue tasks.
		if err := reclustering.RegisterTaskHandler(srv); err != nil {
			return errors.Annotate(err, "register reclustering").Err()
		}
		if err := resultingester.RegisterTaskHandler(srv); err != nil {
			return errors.Annotate(err, "register result ingester").Err()
		}
		resultcollector.RegisterTaskClass()
		testvariantbqexporter.RegisterTaskClass()
		testvariantupdator.RegisterTaskClass()

		return init(srv)
	})
}
