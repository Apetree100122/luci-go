// Copyright 2016 The LUCI Authors.
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

// Package tokenminter implements TokenMinter API.
//
// This is main public API of The Token Server.
package tokenminter

import (
	"cloud.google.com/go/bigquery"

	"go.chromium.org/luci/server/auth"
	"go.chromium.org/luci/server/auth/signing"

	"go.chromium.org/luci/tokenserver/api/minter/v1"
	"go.chromium.org/luci/tokenserver/appengine/impl/certchecker"
	"go.chromium.org/luci/tokenserver/appengine/impl/delegation"
	"go.chromium.org/luci/tokenserver/appengine/impl/machinetoken"
	"go.chromium.org/luci/tokenserver/appengine/impl/projectscope"
	"go.chromium.org/luci/tokenserver/appengine/impl/serviceaccounts"
	"go.chromium.org/luci/tokenserver/appengine/impl/serviceaccountsv2"
	"go.chromium.org/luci/tokenserver/appengine/impl/utils/projectidentity"
)

// serverImpl implements minter.TokenMinterServer RPC interface.
type serverImpl struct {
	minter.UnsafeTokenMinterServer

	machinetoken.MintMachineTokenRPC
	delegation.MintDelegationTokenRPC
	serviceaccounts.MintOAuthTokenGrantRPC
	serviceaccounts.MintOAuthTokenViaGrantRPC
	projectscope.MintProjectTokenRPC
	serviceaccountsv2.MintServiceAccountTokenRPC
}

// NewServer returns prod TokenMinterServer implementation.
//
// It does all authorization checks inside.
func NewServer(signer signing.Signer, bq *bigquery.Client, prod bool) minter.TokenMinterServer {
	return &serverImpl{
		MintMachineTokenRPC: machinetoken.MintMachineTokenRPC{
			Signer:           signer,
			CheckCertificate: certchecker.CheckCertificate,
			LogToken:         machinetoken.NewTokenLogger(bq, !prod),
		},
		MintDelegationTokenRPC: delegation.MintDelegationTokenRPC{
			Signer:   signer,
			Rules:    delegation.GlobalRulesCache.Rules,
			LogToken: delegation.NewTokenLogger(bq, !prod),
		},
		MintOAuthTokenGrantRPC: serviceaccounts.MintOAuthTokenGrantRPC{
			Signer:   signer,
			Rules:    serviceaccounts.GlobalRulesCache.Rules,
			LogGrant: serviceaccounts.NewGrantLogger(bq, !prod),
		},
		MintOAuthTokenViaGrantRPC: serviceaccounts.MintOAuthTokenViaGrantRPC{
			Signer:          signer,
			Rules:           serviceaccounts.GlobalRulesCache.Rules,
			MintAccessToken: auth.MintAccessTokenForServiceAccount,
			LogOAuthToken:   serviceaccounts.NewOAuthTokenLogger(bq, !prod),
		},
		MintProjectTokenRPC: projectscope.MintProjectTokenRPC{
			Signer:            signer,
			MintAccessToken:   auth.MintAccessTokenForServiceAccount,
			ProjectIdentities: projectidentity.ProjectIdentities,
			LogToken:          projectscope.NewTokenLogger(bq, !prod),
		},
		MintServiceAccountTokenRPC: serviceaccountsv2.MintServiceAccountTokenRPC{
			Signer:          signer,
			Mapping:         serviceaccountsv2.GlobalMappingCache.Mapping,
			MintAccessToken: auth.MintAccessTokenForServiceAccount,
			MintIDToken:     auth.MintIDTokenForServiceAccount,
			LogToken:        serviceaccountsv2.NewTokenLogger(bq, !prod),
		},
	}
}
