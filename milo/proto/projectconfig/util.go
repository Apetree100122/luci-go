// Copyright 2017 The LUCI Authors.
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

package projectconfigpb

import (
	"strings"

	"go.chromium.org/luci/milo/internal/utils"
)

// ParseCategory takes a Builder's Category and parses it into a list of
// subcategories. The top-level category is listed first.
func (b *Builder) ParseCategory() []string {
	return strings.Split(b.Category, "|")
}

// AllLegacyBuilderIDs returns all BuilderIDs in legacy format mentioned by this
// Console.
func (c *Console) AllLegacyBuilderIDs() ([]string, error) {
	builders := make([]string, 0, len(c.Builders))
	for _, b := range c.Builders {
		builders = append(builders, utils.LegacyBuilderIDString(b.Id))
	}
	return builders, nil
}
