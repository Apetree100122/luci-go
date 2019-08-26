// Copyright 2019 The LUCI Authors.
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
	"flag"

	"go.chromium.org/luci/common/logging"

	"go.chromium.org/luci/server"
	"go.chromium.org/luci/server/router"

	"go.chromium.org/luci/hardcoded/chromeinfra"

	resultspb "go.chromium.org/luci/results/proto/v1"
)

func main() {
	opts := server.Options{
		ClientAuth: chromeinfra.DefaultAuthOptions(),
	}
	opts.Register(flag.CommandLine)
	flag.Parse()

	srv, err := server.New(opts)
	if err != nil {
		srv.Fatal(err)
	}

	srv.Routes.GET("/", router.MiddlewareChain{}, func(c *router.Context) {
		logging.Debugf(c.Context, "DEV Hello debug world")
		logging.Infof(c.Context, "DEV Hello info world")
		logging.Warningf(c.Context, "DEV Hello warning world")
		c.Writer.Write([]byte("DEV Hello, world"))
	})

	// Register pRPC services.
	resultspb.RegisterRecorderServer(srv.PRPC, &RecorderServer{})

	if err := srv.ListenAndServe(); err != nil {
		srv.Fatal(err)
	}
}
