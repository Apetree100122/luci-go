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

package secrets

import (
	"context"
	"flag"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"google.golang.org/api/option"

	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/server/auth"
	"go.chromium.org/luci/server/module"
)

// ModuleName can be used to refer to this module when declaring dependencies.
var ModuleName = module.RegisterName("go.chromium.org/luci/server/secrets")

// ModuleOptions contain configuration of the secrets server module.
type ModuleOptions struct {
	// RootSecret points to the root secret used to derive random secrets.
	//
	// In production it should be a reference to a Google Secret Manager secret
	// (in a form "sm://<project>/<secret>" or just "sm://<secret>" to fetch it
	// from the current project).
	//
	// In non-production environments it can be a literal secret value in a form
	// "devsecret://<base64-encoded secret>" or "devsecret-text://<secret>". If
	// omitted in a non-production environment, some phony hardcoded value is
	// used.
	//
	// When using Google Secret Manager, the secret version "latest" is used to
	// get the current value of the root secret, and a single immediately
	// preceding previous version (if it is still enabled) is used to get the
	// previous version of the root secret. This allows graceful rotation of
	// random secrets.
	RootSecret string
}

// Register registers the command line flags.
func (o *ModuleOptions) Register(f *flag.FlagSet) {
	f.StringVar(
		&o.RootSecret,
		"root-secret",
		o.RootSecret,
		`Either "sm://<project>/<secret>" or "sm://<secret>" to use Google Secret Manager, `+
			`or "devsecret://<base64-encoded value>" or "devsecret-text://<value>" `+
			`for a static development secret`,
	)
}

// NewModule returns a server module that adds a secret store backed by Google
// Secret Manager to the global server context.
func NewModule(opts *ModuleOptions) module.Module {
	if opts == nil {
		opts = &ModuleOptions{}
	}
	return &serverModule{opts: opts}
}

// NewModuleFromFlags is a variant of NewModule that initializes options through
// command line flags.
//
// Calling this function registers flags in flag.CommandLine. They are usually
// parsed in server.Main(...).
func NewModuleFromFlags() module.Module {
	opts := &ModuleOptions{}
	opts.Register(flag.CommandLine)
	return NewModule(opts)
}

// serverModule implements module.Module.
type serverModule struct {
	opts *ModuleOptions
}

// Name is part of module.Module interface.
func (*serverModule) Name() module.Name {
	return ModuleName
}

// Dependencies is part of module.Module interface.
func (*serverModule) Dependencies() []module.Dependency {
	return nil
}

// Initialize is part of module.Module interface.
func (m *serverModule) Initialize(ctx context.Context, host module.Host, opts module.HostOptions) (context.Context, error) {
	if !opts.Prod && m.opts.RootSecret == "" {
		m.opts.RootSecret = "devsecret-text://phony-root-secret-do-not-depend-on"
	}

	ts, err := auth.GetTokenSource(ctx, auth.AsSelf, auth.WithScopes(auth.CloudOAuthScopes...))
	if err != nil {
		return nil, errors.Annotate(err, "failed to initialize the token source").Err()
	}
	client, err := secretmanager.NewClient(ctx, option.WithTokenSource(ts))
	if err != nil {
		return nil, errors.Annotate(err, "failed to initialize the Secret Manager client").Err()
	}
	host.RegisterCleanup(func(context.Context) { client.Close() })

	store := &SecretManagerStore{
		CloudProject:        opts.CloudProject,
		AccessSecretVersion: client.AccessSecretVersion,
	}
	if m.opts.RootSecret != "" {
		if err := store.LoadRootSecret(ctx, m.opts.RootSecret); err != nil {
			return nil, errors.Annotate(err, "failed to initialize the secret store").Err()
		}
	}

	host.RunInBackground("luci.secrets", store.MaintenanceLoop)
	return Use(ctx, store), nil
}
