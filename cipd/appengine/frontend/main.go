// Copyright 2017 The LUCI Authors.
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

// Binary frontend implements HTTP server that handles requests to 'default'
// module.
package main

import (
	"net/http"

	"google.golang.org/appengine"

	"go.chromium.org/luci/appengine/gaeauth/server"
	"go.chromium.org/luci/appengine/gaemiddleware/standard"
	"go.chromium.org/luci/grpc/discovery"
	"go.chromium.org/luci/grpc/grpcmon"
	"go.chromium.org/luci/grpc/grpcutil"
	"go.chromium.org/luci/grpc/prpc"
	"go.chromium.org/luci/server/auth"
	"go.chromium.org/luci/server/router"
	"go.chromium.org/luci/web/gowrappers/rpcexplorer"

	adminapi "go.chromium.org/luci/cipd/api/admin/v1"
	pubapi "go.chromium.org/luci/cipd/api/cipd/v1"
	"go.chromium.org/luci/cipd/appengine/impl"
	"go.chromium.org/luci/cipd/appengine/ui"
)

func main() {
	r := router.New()

	// Install auth, config and tsmon handlers.
	server.SwitchToEncryptedCookies()
	standard.InstallHandlers(r)
	impl.InitForGAE1(nil, router.MiddlewareChain{}) // don't install HTTP routes

	// RPC Explorer UI.
	rpcexplorer.Install(r)

	// Register non-pRPC routes, such as the client bootstrap handler and routes
	// to support minimal subset of legacy API required to let old CIPD clients
	// fetch packages and self-update.
	impl.PublicRepo.InstallHandlers(r, standard.Base().Extend(
		auth.Authenticate(&server.OAuth2Method{
			Scopes: []string{server.EmailScope},
		}),
	))

	// UI pages.
	ui.InstallHandlers(r, standard.Base(), "templates")

	// Install all RPC servers. Catch panics, report metrics to tsmon (including
	// panics themselves, as Internal errors).
	srv := &prpc.Server{
		UnaryServerInterceptor: grpcutil.ChainUnaryServerInterceptors(
			grpcmon.UnaryServerInterceptor,
			grpcutil.UnaryServerPanicCatcherInterceptor,
		),
	}
	adminapi.RegisterAdminServer(srv, impl.AdminAPI)
	pubapi.RegisterStorageServer(srv, impl.PublicCAS)
	pubapi.RegisterRepositoryServer(srv, impl.PublicRepo)
	discovery.Enable(srv)

	srv.InstallHandlers(r, standard.Base())
	http.DefaultServeMux.Handle("/", r)
	appengine.Main()
}
