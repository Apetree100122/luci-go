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

package coordinatorTest

import (
	"context"

	"go.chromium.org/luci/logdog/api/config/svcconfig"
	"go.chromium.org/luci/logdog/appengine/coordinator"
	"go.chromium.org/luci/logdog/appengine/coordinator/config"
	"go.chromium.org/luci/logdog/appengine/coordinator/endpoints"
	"go.chromium.org/luci/logdog/appengine/coordinator/flex"
)

// Services is a testing stub for a coordinator.Services instance that allows
// the user to configure the various services that are returned.
type Services struct {
	// C, if not nil, will be used to get the return values for Config, overriding
	// local static members.
	C func() (*svcconfig.Config, error)

	// PC, if not nil, will be used to get the return values for ProjectConfig,
	// overriding local static members.
	PC func() (*svcconfig.ProjectConfig, error)

	// Storage returns an intermediate storage instance for use by this service.
	//
	// The caller must close the returned instance if successful.
	//
	// By default, this will return a *BigTableStorage instance bound to the
	// Environment's BigTable instance if the stream is not archived, and an
	// *ArchivalStorage instance bound to this Environment's GSClient instance
	// if the stream is archived.
	ST func(*coordinator.LogStreamState) (coordinator.SigningStorage, error)
}

var _ endpoints.Services = (*Services)(nil)
var _ flex.Services = (*Services)(nil)

// Config implements coordinator.Services.
func (s *Services) Config(c context.Context) (*svcconfig.Config, error) {
	if s.C != nil {
		return s.C()
	}
	return config.Load(c)
}

// ProjectConfig implements coordinator.Services.
func (s *Services) ProjectConfig(c context.Context, project string) (*svcconfig.ProjectConfig, error) {
	if s.PC != nil {
		return s.PC()
	}
	return config.ProjectConfig(c, project)
}

// StorageForStream implements coordinator.Services.
func (s *Services) StorageForStream(c context.Context, lst *coordinator.LogStreamState, project string) (coordinator.SigningStorage, error) {
	if s.ST != nil {
		return s.ST(lst)
	}
	panic("not implemented")
}
