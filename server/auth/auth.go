// Copyright 2015 The LUCI Authors.
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

package auth

import (
	"context"
	"fmt"
	"net/http"

	"golang.org/x/oauth2"

	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/common/retry/transient"
	"go.chromium.org/luci/common/trace"

	"go.chromium.org/luci/auth/identity"
	"go.chromium.org/luci/server/auth/delegation"
	"go.chromium.org/luci/server/auth/signing"
	"go.chromium.org/luci/server/router"
)

var (
	// ErrNotConfigured is returned by Authenticate and other functions if the
	// context wasn't previously initialized via 'Initialize'.
	ErrNotConfigured = errors.New("auth: the library is not properly configured")

	// ErrNoUsersAPI is returned by LoginURL and LogoutURL if none of
	// the authentication methods support UsersAPI.
	ErrNoUsersAPI = errors.New("auth: methods do not support login or logout URL")

	// ErrBadClientID is returned by Authenticate if caller is using
	// non-whitelisted OAuth2 client. More info is in the log.
	ErrBadClientID = errors.New("auth: OAuth client_id is not whitelisted")

	// ErrIPNotWhitelisted is returned when an account is restricted by an IP
	// whitelist and request's remote_addr is not in it.
	ErrIPNotWhitelisted = errors.New("auth: IP is not whitelisted")

	// ErrNoForwardableCreds is returned when attempting to forward credentials
	// (via AsCredentialsForwarder) that are not forwardable.
	ErrNoForwardableCreds = errors.New("auth: no forwardable credentials in the context")
)

// Method implements a particular kind of low-level authentication mechanism.
//
// It may also optionally implement a bunch of other interfaces:
//   UsersAPI: if the method supports login and logout URLs.
//   Warmable: if the method supports warm up.
//   HasHandlers: if the method needs to install HTTP handlers.
//
// Methods are not usually used directly, but passed to Authenticator{...} that
// knows how to apply them.
type Method interface {
	// Authenticate extracts user information from the incoming request.
	//
	// It returns:
	//   * (*User, nil) on success.
	//   * (nil, nil) if the method is not applicable.
	//   * (nil, error) if the method is applicable, but credentials are invalid.
	Authenticate(context.Context, *http.Request) (*User, error)
}

// UsersAPI may be additionally implemented by Method if it supports login and
// logout URLs.
type UsersAPI interface {
	// LoginURL returns a URL that, when visited, prompts the user to sign in,
	// then redirects the user to the URL specified by dest.
	LoginURL(c context.Context, dest string) (string, error)

	// LogoutURL returns a URL that, when visited, signs the user out,
	// then redirects the user to the URL specified by dest.
	LogoutURL(c context.Context, dest string) (string, error)
}

// Warmable may be additionally implemented by Method if it supports warm up.
type Warmable interface {
	// Warmup may be called to precache the data needed by the method.
	//
	// There's no guarantee when it will be called or if it will be called at all.
	// Should always do best-effort initialization. Errors are logged and ignored.
	Warmup(c context.Context) error
}

// HasHandlers may be additionally implemented by Method if it needs to
// install HTTP handlers.
type HasHandlers interface {
	// InstallHandlers installs necessary HTTP handlers into the router.
	InstallHandlers(r *router.Router, base router.MiddlewareChain)
}

// UserCredentialsGetter may be additionally implemented by Method if it knows
// how to extract end-user credentials from the incoming request. Currently
// understands only OAuth2 tokens.
type UserCredentialsGetter interface {
	// GetUserCredentials extracts an OAuth access token from the incoming request
	// or returns an error if it isn't possible.
	//
	// May omit token's expiration time if it isn't known.
	//
	// Guaranteed to be called only after the successful authentication, so it
	// doesn't have to recheck the validity of the token.
	GetUserCredentials(context.Context, *http.Request) (*oauth2.Token, error)
}

// User represents identity and profile of a user.
type User struct {
	// Identity is identity string of the user (may be AnonymousIdentity).
	// If User is returned by Authenticate(...), Identity string is always present
	// and valid.
	Identity identity.Identity `json:"identity,omitempty"`

	// Superuser is true if the user is site-level administrator. For example, on
	// GAE this bit is set for GAE-level administrators. Optional, default false.
	Superuser bool `json:"superuser,omitempty"`

	// Email is email of the user. Optional, default "". Don't use it as a key
	// in various structures. Prefer to use Identity() instead (it is always
	// available).
	Email string `json:"email,omitempty"`

	// Name is full name of the user. Optional, default "".
	Name string `json:"name,omitempty"`

	// Picture is URL of the user avatar. Optional, default "".
	Picture string `json:"picture,omitempty"`

	// ClientID is the ID of the pre-registered OAuth2 client so its identity can
	// be verified. Used only by authentication methods based on OAuth2.
	// See https://developers.google.com/console/help/#generatingoauth2 for more.
	ClientID string `json:"client_id,omitempty"`
}

// Authenticator performs authentication of incoming requests.
//
// It is a stateless object configured with a list of methods to try when
// authenticating incoming requests. It implements Authenticate method that
// performs high-level authentication logic using the provided list of low-level
// auth methods.
//
// Note that most likely you don't need to instantiate this object directly.
// Use Authenticate middleware instead. Authenticator is exposed publicly only
// to be used in advanced cases, when you need to fine-tune authentication
// behavior.
type Authenticator struct {
	Methods []Method // a list of authentication methods to try
}

// GetMiddleware returns a middleware that uses this Authenticator for
// authentication.
//
// It uses a.Authenticate internally and handles errors appropriately.
func (a *Authenticator) GetMiddleware() router.Middleware {
	return func(c *router.Context, next router.Handler) {
		ctx, err := a.Authenticate(c.Context, c.Request)
		switch {
		case transient.Tag.In(err):
			replyError(c.Context, c.Writer, 500, "Transient error during authentication", err)
		case err != nil:
			replyError(c.Context, c.Writer, 401, "Authentication error", err)
		default:
			c.Context = ctx
			next(c)
		}
	}
}

// Authenticate authenticates the requests and adds State into the context.
//
// Returns an error if credentials are provided, but invalid. If no credentials
// are provided (i.e. the request is anonymous), finishes successfully, but in
// that case State.Identity() returns AnonymousIdentity.
func (a *Authenticator) Authenticate(ctx context.Context, r *http.Request) (_ context.Context, err error) {
	tracedCtx, span := trace.StartSpan(ctx, "go.chromium.org/luci/server/auth.Authenticate")
	defer func() { span.End(err) }()

	report := durationReporter(tracedCtx, authenticateDuration)

	// We will need working DB factory below to check IP whitelist.
	cfg := getConfig(tracedCtx)
	if cfg == nil || cfg.DBProvider == nil || len(a.Methods) == 0 {
		report(ErrNotConfigured, "ERROR_NOT_CONFIGURED")
		return nil, ErrNotConfigured
	}

	// Pick first authentication method that applies.
	s := state{authenticator: a}
	for _, m := range a.Methods {
		var err error
		s.user, err = m.Authenticate(tracedCtx, r)
		if err != nil {
			if transient.Tag.In(err) {
				report(err, "ERROR_TRANSIENT_IN_AUTH")
			} else {
				report(err, "ERROR_BROKEN_CREDS") // e.g. malformed OAuth token
			}
			return nil, err
		}
		if s.user != nil {
			if err = s.user.Identity.Validate(); err != nil {
				report(err, "ERROR_BROKEN_IDENTITY") // a weird looking email address
				return nil, err
			}
			s.method = m
			break
		}
	}

	// If no authentication method is applicable, default to anonymous identity.
	if s.method == nil {
		s.user = &User{Identity: identity.AnonymousIdentity}
	}

	// Grab an end user IP as a string and convert it to net.IP to use in IP
	// whitelist check below.
	remoteAddr := r.RemoteAddr
	if cfg.EndUserIP != nil {
		remoteAddr = cfg.EndUserIP(r)
	}
	s.peerIP, err = parseRemoteIP(remoteAddr)
	if err != nil {
		panic(fmt.Errorf("auth: bad remote_addr: %v", err))
	}

	// Grab a snapshot of auth DB to use consistently for the duration of this
	// request.
	s.db, err = cfg.DBProvider(tracedCtx)
	if err != nil {
		report(ErrNotConfigured, "ERROR_NOT_CONFIGURED")
		return nil, ErrNotConfigured
	}

	// If using OAuth2, make sure ClientID is whitelisted.
	if s.user.ClientID != "" {
		valid, err := s.db.IsAllowedOAuthClientID(tracedCtx, s.user.Email, s.user.ClientID)
		if err != nil {
			report(err, "ERROR_TRANSIENT_IN_OAUTH_WHITELIST")
			return nil, err
		}
		if !valid {
			logging.Warningf(
				tracedCtx, "auth: %q is using client_id %q not in the whitelist",
				s.user.Email, s.user.ClientID)
			report(ErrBadClientID, "ERROR_FORBIDDEN_OAUTH_CLIENT")
			return nil, ErrBadClientID
		}
	}

	// Some callers may be constrained by an IP whitelist.
	switch ipWhitelist, err := s.db.GetWhitelistForIdentity(tracedCtx, s.user.Identity); {
	case err != nil:
		report(err, "ERROR_TRANSIENT_IN_IP_WHITELIST")
		return nil, err
	case ipWhitelist != "":
		switch whitelisted, err := s.db.IsInWhitelist(tracedCtx, s.peerIP, ipWhitelist); {
		case err != nil:
			report(err, "ERROR_TRANSIENT_IN_IP_WHITELIST")
			return nil, err
		case !whitelisted:
			report(ErrIPNotWhitelisted, "ERROR_FORBIDDEN_IP")
			return nil, ErrIPNotWhitelisted
		}
	}

	// peerIdent always matches the identity of a remote peer. It may be different
	// from s.user.Identity if the delegation is used (see below).
	s.peerIdent = s.user.Identity

	// TODO(vadimsh): Check X-Luci-Project header. If used, verify the caller is
	// a known LUCI service.

	// Check the delegation token. This is LUCI-specific authentication protocol.
	// Delegation tokens are generated by the central auth service (see luci-py's
	// auth_service) and validated by checking their RSA signature using auth
	// server's public keys.
	delegationTok := r.Header.Get(delegation.HTTPHeaderName)
	if delegationTok != "" {
		// Log the token fingerprint (even before parsing the token), it can be used
		// to grab the info about the token from the token server logs.
		logging.Fields{
			"fingerprint": tokenFingerprint(delegationTok),
		}.Debugf(tracedCtx, "auth: Received delegation token")

		// Need to grab our own identity to verify that the delegation token is
		// minted for consumption by us and not some other service.
		ownServiceIdentity, err := getOwnServiceIdentity(tracedCtx, cfg.Signer)
		if err != nil {
			report(err, "ERROR_TRANSIENT_IN_OWN_IDENTITY")
			return nil, err
		}
		delegatedIdentity, err := delegation.CheckToken(tracedCtx, delegation.CheckTokenParams{
			Token:                delegationTok,
			PeerID:               s.peerIdent,
			CertificatesProvider: s.db,
			GroupsChecker:        s.db,
			OwnServiceIdentity:   ownServiceIdentity,
		})
		if err != nil {
			if transient.Tag.In(err) {
				report(err, "ERROR_TRANSIENT_IN_TOKEN_CHECK")
			} else {
				report(err, "ERROR_BAD_DELEGATION_TOKEN")
			}
			return nil, err
		}

		// User profile information is not available when using delegation, so just
		// wipe it.
		s.user = &User{Identity: delegatedIdentity}

		// Log that 'peerIdent' is pretending to be 'delegatedIdentity'.
		logging.Fields{
			"peerID":      s.peerIdent,
			"delegatedID": delegatedIdentity,
		}.Debugf(tracedCtx, "auth: Using delegation")
	}

	// If not using the delegation, grab the end user creds in case we want to
	// forward them later in GetRPCTransport(AsCredentialsForwarder). Note that
	// generally delegation tokens are not forwardable, so we disable credentials
	// forwarding when delegation is used.
	s.endUserErr = ErrNoForwardableCreds
	if delegationTok == "" {
		if credsGetter, _ := s.method.(UserCredentialsGetter); credsGetter != nil {
			s.endUserTok, s.endUserErr = credsGetter.GetUserCredentials(tracedCtx, r)
		}
	}

	// Inject the auth state into the original context (not the traced one).
	report(nil, "SUCCESS")
	return WithState(ctx, &s), nil
}

// usersAPI returns implementation of UsersAPI by examining Methods.
//
// Returns nil if none of Methods implement UsersAPI.
func (a *Authenticator) usersAPI() UsersAPI {
	for _, m := range a.Methods {
		if api, ok := m.(UsersAPI); ok {
			return api
		}
	}
	return nil
}

// LoginURL returns a URL that, when visited, prompts the user to sign in,
// then redirects the user to the URL specified by dest.
//
// Returns ErrNoUsersAPI if none of the authentication methods support login
// URLs.
func (a *Authenticator) LoginURL(c context.Context, dest string) (string, error) {
	if api := a.usersAPI(); api != nil {
		return api.LoginURL(c, dest)
	}
	return "", ErrNoUsersAPI
}

// LogoutURL returns a URL that, when visited, signs the user out, then
// redirects the user to the URL specified by dest.
//
// Returns ErrNoUsersAPI if none of the authentication methods support login
// URLs.
func (a *Authenticator) LogoutURL(c context.Context, dest string) (string, error) {
	if api := a.usersAPI(); api != nil {
		return api.LogoutURL(c, dest)
	}
	return "", ErrNoUsersAPI
}

////

// replyError logs the error and writes it to ResponseWriter.
func replyError(c context.Context, rw http.ResponseWriter, code int, msg string, err error) {
	logging.WithError(err).Errorf(c, "HTTP %d: %s", code, msg)
	http.Error(rw, msg, code)
}

// getOwnServiceIdentity returns 'service:<appID>' identity of the current
// service.
func getOwnServiceIdentity(c context.Context, signer signing.Signer) (identity.Identity, error) {
	if signer == nil {
		return "", ErrNotConfigured
	}
	serviceInfo, err := signer.ServiceInfo(c)
	if err != nil {
		return "", err
	}
	return identity.MakeIdentity("service:" + serviceInfo.AppID)
}
