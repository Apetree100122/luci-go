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

package config

// Project represents the luci-notify configuration for a single project in the datastore.
type Project struct {
	// Name is the name of the project.
	//
	// This must be unique on this luci-notify instance.
	Name string `gae:"$id"`

	// Revision is the revision of this project's luci-notify configuration.
	Revision string

	// URL is the luci-config URL to this project's luci-notify configuration.
	URL string

	// TreeClosingEnabled determines whether we actually act on TreeClosers
	// for this project, and close/reopen the relevant tree. If false, we
	// still monitor builders, but just log what action we would have taken.
	TreeClosingEnabled bool
}
