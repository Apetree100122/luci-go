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

package coordinator

import (
	"context"
	"fmt"

	log "go.chromium.org/luci/common/logging"
	cfglib "go.chromium.org/luci/config"
	"go.chromium.org/luci/gae/service/info"
	"go.chromium.org/luci/grpc/grpcutil"
	"go.chromium.org/luci/logdog/api/config/svcconfig"
	"go.chromium.org/luci/logdog/server/config"

	"google.golang.org/grpc/codes"
)

// NamespaceAccessType specifies the type of namespace access that is being
// requested for WithProjectNamespace.
type NamespaceAccessType int

const (
	// NamespaceAccessNoAuth grants unconditional access to a project's namespace.
	// This bypasses all ACL checks, and must only be used by service endpoints
	// that explicitly apply ACLs elsewhere.
	NamespaceAccessNoAuth NamespaceAccessType = iota

	// NamespaceAccessAllTesting is an extension of NamespaceAccessNoAuth that,
	// in addition to doing no ACL checks, also does no project existence checks.
	//
	// This must ONLY be used for testing.
	NamespaceAccessAllTesting

	// NamespaceAccessREAD enforces READ permission access to a project's
	// namespace.
	NamespaceAccessREAD

	// NamespaceAccessWRITE enforces WRITE permission access to a project's
	// namespace.
	NamespaceAccessWRITE
)

// WithProjectNamespace sets the current namespace to the project name.
//
// It will return a user-facing wrapped gRPC error on failure:
//	- InvalidArgument if the project name is invalid.
//	- If the project exists, then
//	  - nil, if the user has the requested access.
//	  - Unauthenticated if the user does not have the requested access, but is
//	    also not authenticated. This lets them know they should try again after
//	    authenticating.
//	  - PermissionDenied if the user does not have the requested access.
//	- PermissionDenied if the project doesn't exist.
//	- Internal if an internal error occurred.
func WithProjectNamespace(c *context.Context, project string, at NamespaceAccessType) error {
	ctx := *c

	if err := cfglib.ValidateProjectName(project); err != nil {
		log.WithError(err).Errorf(ctx, "Project name is invalid.")
		return grpcutil.Errf(codes.InvalidArgument, "Project name is invalid: %s", err)
	}

	// Returns the project config, or "read denied" error if the project does not
	// exist.
	getProjectConfig := func() (*svcconfig.ProjectConfig, error) {
		pcfg, err := config.ProjectConfig(ctx, project)
		switch err {
		case nil:
			// Successfully loaded project config.
			return pcfg, nil

		case cfglib.ErrNoConfig, config.ErrInvalidConfig:
			// If the configuration request was valid, but no configuration could be
			// loaded, treat this as the user not having READ access to the project.
			// Otherwise, the user could use this error response to confirm a
			// project's existence.
			log.Fields{
				log.ErrorKey: err,
				"project":    project,
			}.Errorf(ctx, "Could not load config for project.")
			return nil, PermissionDeniedErr(ctx)

		default:
			// The configuration attempt failed to load. This is an internal error,
			// and is safe to return because it's not contingent on the existence (or
			// lack thereof) of the project.
			return nil, grpcutil.Internal
		}
	}

	// Validate that the current user has the requested access.
	switch at {
	case NamespaceAccessNoAuth:
		// Assert that the project exists and has a configuration.
		if _, err := getProjectConfig(); err != nil {
			return err
		}

	case NamespaceAccessAllTesting:
		// Sanity check: this should only be used on development instances.
		if !info.IsDevAppServer(ctx) {
			panic("Testing access requested on non-development instance.")
		}
		break

	case NamespaceAccessREAD:
		// Assert that the current user has READ access.
		pcfg, err := getProjectConfig()
		if err != nil {
			return err
		}
		switch yes, err := CheckProjectReader(ctx, pcfg); {
		case err != nil:
			return grpcutil.Internal
		case !yes:
			return PermissionDeniedErr(ctx)
		}

	case NamespaceAccessWRITE:
		// Assert that the current user has WRITE access.
		pcfg, err := getProjectConfig()
		if err != nil {
			return err
		}
		switch yes, err := CheckProjectWriter(ctx, pcfg); {
		case err != nil:
			return grpcutil.Internal
		case !yes:
			return PermissionDeniedErr(ctx)
		}

	default:
		panic(fmt.Errorf("unknown access type: %v", at))
	}

	pns := ProjectNamespace(project)
	nc, err := info.Namespace(ctx, pns)
	if err != nil {
		log.Fields{
			log.ErrorKey: err,
			"project":    project,
			"namespace":  pns,
		}.Errorf(ctx, "Failed to set namespace.")
		return grpcutil.Internal
	}

	*c = nc
	return nil
}

// Project returns the current project installed in the supplied Context's
// namespace.
//
// This function is called with the expectation that the Context is in a
// namespace conforming to ProjectNamespace. If this is not the case, this
// method will panic.
func Project(ctx context.Context) string {
	ns := info.GetNamespace(ctx)
	project := ProjectFromNamespace(ns)
	if project != "" {
		return project
	}
	panic(fmt.Errorf("current namespace %q does not begin with project namespace prefix (%q)", ns, ProjectNamespacePrefix))
}

// ProjectConfig returns the project-specific configuration for the
// current project.
//
// If there is no current project namespace, or if the current project has no
// configuration, config.ErrInvalidConfig will be returned.
func ProjectConfig(ctx context.Context) (*svcconfig.ProjectConfig, error) {
	if project := ProjectFromNamespace(info.GetNamespace(ctx)); project != "" {
		return config.ProjectConfig(ctx, project)
	}
	return nil, config.ErrInvalidConfig
}
