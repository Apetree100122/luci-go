// Copyright 2018 The LUCI Authors.
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

// Package main is the main entry point for the app.
package main

import (
	"net/http"

	"google.golang.org/appengine"

	gaeserver "go.chromium.org/luci/appengine/gaeauth/server"
	"go.chromium.org/luci/appengine/gaemiddleware/standard"
	"go.chromium.org/luci/common/data/rand/mathrand"
	"go.chromium.org/luci/grpc/discovery"
	"go.chromium.org/luci/grpc/grpcmon"
	"go.chromium.org/luci/grpc/prpc"
	"go.chromium.org/luci/server/auth"
	"go.chromium.org/luci/server/router"
	"go.chromium.org/luci/web/gowrappers/rpcexplorer"

	server "go.chromium.org/luci/gce/api/config/v1"
	"go.chromium.org/luci/gce/api/instances/v1"
	"go.chromium.org/luci/gce/api/projects/v1"
	"go.chromium.org/luci/gce/appengine/backend"
	"go.chromium.org/luci/gce/appengine/config"
	"go.chromium.org/luci/gce/appengine/rpc"
	"go.chromium.org/luci/gce/vmtoken"
)

func main() {
	mathrand.SeedRandomly()
	api := prpc.Server{
		UnaryServerInterceptor: grpcmon.UnaryServerInterceptor,
		Authenticator: &auth.Authenticator{
			Methods: []auth.Method{
				&gaeserver.OAuth2Method{Scopes: []string{gaeserver.EmailScope}},
			},
		},
	}
	server.RegisterConfigurationServer(&api, rpc.NewConfigurationServer())
	instances.RegisterInstancesServer(&api, rpc.NewInstancesServer())
	projects.RegisterProjectsServer(&api, rpc.NewProjectsServer())
	discovery.Enable(&api)

	r := router.New()

	standard.InstallHandlers(r)
	rpcexplorer.Install(r)

	mw := standard.Base()
	api.InstallHandlers(r, mw.Extend(vmtoken.Middleware))
	backend.InstallHandlers(r, mw)
	config.InstallHandlers(r, mw)

	http.DefaultServeMux.Handle("/", r)
	appengine.Main()
}
