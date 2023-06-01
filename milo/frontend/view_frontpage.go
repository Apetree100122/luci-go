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

package frontend

import (
	"go.chromium.org/luci/server/router"
	"go.chromium.org/luci/server/templates"

	"go.chromium.org/luci/milo/frontend/ui"
	"go.chromium.org/luci/milo/internal/projectconfig"
)

func frontpageHandler(c *router.Context) {
	projs, err := projectconfig.GetVisibleProjects(c.Request.Context())
	if err != nil {
		ErrorHandler(c, err)
		return
	}
	templates.MustRender(c.Request.Context(), c.Writer, "pages/frontpage.html", templates.Args{
		"frontpage": ui.Frontpage{Projects: projs},
	})
}
