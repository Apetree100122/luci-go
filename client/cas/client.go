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

// Package cas provides remote-apis-sdks client with luci integration.
package cas

import (
	"context"
	"fmt"
	"runtime"
	"strings"

	"github.com/bazelbuild/remote-apis-sdks/go/pkg/client"
	"google.golang.org/grpc/credentials"

	"go.chromium.org/luci/auth"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/hardcoded/chromeinfra"
)

// NewClient returns luci auth configured Client for RBE-CAS.
func NewClient(ctx context.Context, instance string, opts auth.Options, readOnly bool) (*client.Client, error) {
	project := strings.Split(instance, "/")[1]
	var role string
	if readOnly {
		role = "cas-read-only"
	} else {
		role = "cas-read-write"
	}

	// Construct auth.Options.
	opts.ActAsServiceAccount = fmt.Sprintf("%s@%s.iam.gserviceaccount.com", role, project)
	opts.ActViaLUCIRealm = fmt.Sprintf("@internal:%s/%s", project, role)
	opts.Scopes = []string{"https://www.googleapis.com/auth/cloud-platform"}

	if strings.HasSuffix(project, "-dev") || strings.HasSuffix(project, "-staging") {
		// use dev token server for dev/staging projects.
		opts.TokenServerHost = chromeinfra.TokenServerDevHost
	}

	creds, err := auth.NewAuthenticator(ctx, auth.SilentLogin, opts).PerRPCCredentials()
	if err != nil {
		return nil, errors.Annotate(err, "failed to get PerRPCCredentials").Err()
	}
	dialParams := client.DialParams{
		Service:            "remotebuildexecution.googleapis.com:443",
		TransportCredsOnly: true,
	}

	cl, err := client.NewClient(ctx, instance, dialParams, ClientOptions(creds)...)
	if err != nil {
		return nil, errors.Annotate(err, "failed to create client").Err()
	}
	return cl, nil
}

// ClientOptions returns CAS client options.
func ClientOptions(creds credentials.PerRPCCredentials) []client.Opt {
	casConcurrency := runtime.NumCPU() * 2
	if runtime.GOOS == "windows" {
		// This is for better file write performance on Windows (http://b/171672371#comment6).
		casConcurrency = runtime.NumCPU()
	}

	return []client.Opt{
		&client.PerRPCCreds{Creds: creds},
		client.CASConcurrency(casConcurrency),
		client.UtilizeLocality(true),
		&client.TreeSymlinkOpts{Preserved: true, FollowsTarget: false},
		// Set restricted permission for written files.
		client.DirMode(0700),
		client.ExecutableMode(0700),
		client.RegularMode(0600),
		client.CompressedBytestreamThreshold(0),

		// Do not set per RPC timeout.
		client.RPCTimeouts{},
	}
}

// ContextWithMetadata attaches RBE related metadata with tool name to the
// given context.
func ContextWithMetadata(ctx context.Context, toolName string) (context.Context, error) {
	ctx, err := client.ContextWithMetadata(ctx, &client.ContextMetadata{
		ToolName: toolName,
	})
	if err != nil {
		return nil, errors.Annotate(err, "failed to attach metadata").Err()
	}

	m, err := client.GetContextMetadata(ctx)
	if err != nil {
		return nil, errors.Annotate(err, "failed to extract metadata").Err()
	}

	logging.Infof(ctx, "context metadata: %#+v", *m)

	return ctx, nil
}
