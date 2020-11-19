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

package gerrit

import (
	"context"
	"net/http"
	"strings"
	"time"

	"golang.org/x/oauth2"

	luciauth "go.chromium.org/luci/auth"
	"go.chromium.org/luci/common/api/gerrit"
	"go.chromium.org/luci/common/data/caching/lru"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/common/retry/transient"
	"go.chromium.org/luci/gae/service/datastore"
	"go.chromium.org/luci/server/auth"

	"go.chromium.org/luci/cv/internal/servicecfg"
)

// factory knows how to construct Gerrit Clients.
//
// CV must use project-scoped credentials, but not every project has configured
// project-scoped service account (PSSA). Worse, some projects configured PSSA
// before CQDaemon supported PSSA. Hence, when CQDaemon gained support for PSSA,
// CQDaemon had to blocklist these projects to avoid breakage.  The alternative
// and legacy authentication is based on per GerritHost auth tokens from
// ~/.netrc shared by all LUCI projects. CQDaemon logic is roughly:
//
//   if project in blocklist:
//     return use_legacy_netrc
//   try:
//     token = token_server.MintToken(project)
//   except 404: # not configured
//     return use_legacy_netrc
//   return use_pssa(token)
//
// For smooth migration from CQDaemon to CV, CV re-implements the same logic.
//
// Caveat: for smooth migration of other LUCI services in Go to PSSA,
// auth.GetRPCTransport(ctx, auth.AsProject, ...) helpfully and transparently
// defaults to auth.AsSelf if LUCI project doesn't have PSSA configured.
// Thus CV can't rely on the above method as is.
type factory struct {
	legacyCache *lru.Cache // caches legacy tokens and lack thereof per gerritHost.

	mockMintProjectToken func(context.Context, auth.ProjectTokenParams) (*auth.Token, error)
}

func newFactory() *factory {
	return &factory{
		// CV supports <20 legacy hosts. New ones shouldn't be added.
		legacyCache: lru.New(20),
	}
}

func (f *factory) makeClient(ctx context.Context, gerritHost, luciProject string) (Client, error) {
	if strings.ContainsRune(luciProject, '.') {
		panic(errors.Reason("swapped host %q with luciProject %q", gerritHost, luciProject).Err())
	}
	t, err := f.transport(ctx, gerritHost, luciProject)
	if err != nil {
		return nil, err
	}
	return gerrit.NewRESTClient(&http.Client{Transport: t}, gerritHost, true)
}

func (f *factory) transport(ctx context.Context, gerritHost, luciProject string) (http.RoundTripper, error) {
	// Do what auth.GetRPCTransport(ctx, auth.AsProject, ...) would do,
	// except obey pssa blocklist and default to legacy ~/.netrc creds.
	// See factory doc for more details.
	baseTransport, err := auth.GetRPCTransport(ctx, auth.NoAuth)
	if err != nil {
		return nil, err
	}
	return luciauth.NewModifyingTransport(baseTransport, func(req *http.Request) error {
		tok, err := f.token(ctx, gerritHost, luciProject)
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", tok.TokenType+" "+tok.AccessToken)
		return nil
	}), nil
}

func (f *factory) token(ctx context.Context, gerritHost, luciProject string) (*oauth2.Token, error) {
	cfg, err := servicecfg.GetMigrationConfig(ctx)
	if err != nil {
		return nil, err
	}
	allowed := true
	for _, project := range cfg.PssaMigration.ProjectsBlocklist {
		if luciProject == project {
			allowed = false
			logging.Debugf(ctx, "Project %q is not allowed to use project scoped service account", luciProject)
		}
	}
	if allowed {
		req := auth.ProjectTokenParams{
			MinTTL:      2 * time.Minute,
			LuciProject: luciProject,
			OAuthScopes: []string{gerrit.OAuthScope},
		}
		mintToken := auth.MintProjectToken
		if f.mockMintProjectToken != nil {
			mintToken = f.mockMintProjectToken
		}
		switch token, err := mintToken(ctx, req); {
		case err != nil:
			return nil, err
		case token != nil:
			return &oauth2.Token{
				AccessToken: token.Token,
				TokenType:   "Bearer",
			}, nil
		}
	}

	value, err := f.legacyCache.GetOrCreate(ctx, gerritHost, func() (value interface{}, ttl time.Duration, err error) {
		nt := netrcToken{GerritHost: gerritHost}
		switch err = datastore.Get(ctx, &nt); {
		case err == datastore.ErrNoSuchEntity:
			// While not expected in practice, speed up rollout of a fix by caching
			// for a short time only.
			ttl = 1 * time.Minute
			value = ""
			err = nil
		case err != nil:
			err = errors.Annotate(err, "failed to get legacy creds").Tag(transient.Tag).Err()
		default:
			value = nt.AccessToken
			ttl = 10 * time.Minute
		}
		return
	})

	switch {
	case err != nil:
		return nil, err
	case value.(string) == "":
		return nil, errors.Reason("No legacy credentials for host %q", gerritHost).Err()
	default:
		return &oauth2.Token{
			AccessToken: value.(string),
			TokenType:   "Basic",
		}, nil
	}
}

// netrcToken stores ~/.netrc access tokens of CQDaemon.
type netrcToken struct {
	GerritHost  string `gae:"$id"`
	AccessToken string `gae:",noindex"`
}

// SaveLegacyNetrcToken creates or updates legacy netrc token.
func SaveLegacyNetrcToken(ctx context.Context, host, token string) error {
	err := datastore.Put(ctx, &netrcToken{host, token})
	return errors.Annotate(err, "failed to save legacy netrc token").Err()
}
