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

package config

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"

	"google.golang.org/protobuf/proto"

	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/retry/transient"
	"go.chromium.org/luci/gae/service/datastore"

	pb "go.chromium.org/luci/cv/api/config/v2"
)

const projectConfigKind string = "ProjectConfig"

// ProjectConfig is the root entity that keeps track of the latest version
// info of the CV config for a LUCI Project. It only contains high-level
// metadata about the config. The actual content of config is stored in the
// `ConfigGroup` entities which can be looked up by constructing IDs using
// `ConfigGroupNames` field.
type ProjectConfig struct {
	_kind string `gae:"$kind,ProjectConfig"`
	// Project is the name of this LUCI Project.
	Project string `gae:"$id"`
	// Enabled indicates whether CV is enabled for this LUCI Project.
	//
	// Project is disabled if it is de-registered in LUCI Config or it no longer
	// has CV config file.
	Enabled bool
	// UpdateTime is the timestamp when this ProjectConfig was last updated.
	UpdateTime time.Time `gae:",noindex"`
	// EVersion is the latest version number of this ProjectConfig.
	//
	// It increments by 1 every time a new config change is imported to CV for
	// this LUCI Project.
	EVersion int64 `gae:",noindex"`
	// Hash is a string computed from the content of latest imported CV Config
	// using `computeHash()`.
	Hash string `gae:",noindex"`
	// ExternalHash is the hash string of this CV config in the external source
	// of truth (currently, LUCI Config). Used to quickly decided whether the
	// Config has been updated without fetching the full content.
	ExternalHash string `gae:",noindex"`
	// ConfigGroupNames are the names of all ConfigGroups in the current version
	// of CV Config.
	ConfigGroupNames []string `gae:",noindex"`
}

// computeHash computes the hash string of given CV Config and prefixed with
// hash algorithm string. (e.g. sha256:deadbeefdeadbeef)
//
// The hash string is an hex-encoded string of the first 8 bytes (i.e. 16
// char in length) of sha256(deterministically binary serialized Config proto).
// Note that, deterministic marshalling does NOT guarantee the same output
// for the equal proto message  across different language or event builds.
// Therefore, in worst case scenario, when a newer version of proto lib is
// deployed, CV may re-ingest functionally equivalent config.
// See: https://godoc.org/google.golang.org/protobuf/proto#MarshalOptions
func computeHash(cfg *pb.Config) string {
	b, err := proto.MarshalOptions{Deterministic: true}.Marshal(cfg)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal config: %s", err))
	}
	sha := sha256.New()
	sha.Write(b)
	return fmt.Sprintf("sha256:%s", hex.EncodeToString(sha.Sum(nil)[:8]))
}

// getAllProjectIDs returns the names of all projects available in datastore.
func getAllProjectIDs(ctx context.Context, enabledOnly bool) ([]string, error) {
	var projects []*ProjectConfig
	query := datastore.NewQuery(projectConfigKind).Project("Enabled")
	if err := datastore.GetAll(ctx, query, &projects); err != nil {
		return nil, errors.Annotate(err, "failed to query all projects").Tag(transient.Tag).Err()
	}
	ret := make([]string, 0, len(projects))
	for _, p := range projects {
		if enabledOnly && !p.Enabled {
			continue
		}
		ret = append(ret, p.Project)
	}
	sort.Strings(ret)
	return ret, nil
}

// ConfigHashInfo stores high-level info about a ProjectConfig `Hash`.
//
// It is primarily used for cleanup purpose to decide which `Hash` and
// its corresponding `ConfigGroup`s can be safely deleted.
type ConfigHashInfo struct {
	_kind string `gae:"$kind,ProjectConfigHashInfo"`
	// Hash is the `Hash` of a `ProjectConfig` CV has imported.
	Hash    string         `gae:"$id"`
	Project *datastore.Key `gae:"$parent"`
	// ProjectEVersion is largest version of ProjectConfig that this `Hash`
	// maps to.
	//
	// It is possible for a ConfigHash maps to multiple EVersions (e.g. a CV
	// Config change is landed then reverted which results in two new EVersions
	// but only one new Hash). Only the largest EVersion matters when cleanup
	// job runs (i.e. CV will keep the last 5 EVersions).
	ProjectEVersion int64 `gae:",noindex"`
	// UpdateTime is the timestamp when this ConfigHashInfo was last updated.
	UpdateTime time.Time `gae:",noindex"`
	// ConfigGroupNames are the names of all ConfigGroups with this `Hash`.
	ConfigGroupNames []string `gae:",noindex"`
}

// ConfigGroupID is the ID for ConfigGroup Entity.
//
// It is in the format of "hash/name" where
//   - `hash` is the `Hash` field in the containing `ProjectConfig`.
//   - `name` is the value of `ConfigGroup.Name` if specified. If `name is
//      not provided (Name in ConfigGroup is optional as of Sep. 2020. See:
//      crbug/1063508), use "index#i" as name instead where `i` is the index
//      (0-based) of this ConfigGroup in the config.
type ConfigGroupID string

// Returns Hash of the corresponding project config.
func (c ConfigGroupID) Hash() string {
	s := string(c)
	if i := strings.IndexRune(s, '/'); i >= 0 {
		return s[:i]
	}
	panic(fmt.Errorf("invalid ConfigGroupID %q", c))
}

func makeConfigGroupID(hash, name string, index int) ConfigGroupID {
	return ConfigGroupID(fmt.Sprintf("%s/%s", hash, makeConfigGroupName(name, index)))
}

func makeConfigGroupName(name string, index int) string {
	if name == "" {
		return fmt.Sprintf("index#%d", index)
	}
	return name
}

// ConfigGroup is an entity that represents a ConfigGroup defined in CV config.
type ConfigGroup struct {
	_kind   string         `gae:"$kind,ProjectConfigGroup"`
	Project *datastore.Key `gae:"$parent"`
	ID      ConfigGroupID  `gae:"$id"`
	// DrainingStartTime represents `draining_start_time` field in the CV config.
	//
	// Note that this is a project-level field. Therefore, all ConfigGroups in a
	// single version of config should have the same value.
	DrainingStartTime string `gae:",noindex"`
	// SubmitOptions represents `submit_options` field in the CV config.
	//
	// Note that this is currently a project-level field. Therefore, all
	// ConfigGroups in a single version of Config should have the same value.
	SubmitOptions *pb.SubmitOptions
	// Content represents a `pb.ConfigGroup` proto message defined in the CV
	// config
	Content *pb.ConfigGroup
}

// putConfigGroups puts the ConfigGroups in the given CV config to datastore.
//
// It checks for existence of each ConfigGroup first to avoid unnecessary puts.
// It is also idempotent so it is safe to retry and can be called out of a
// transactional context.
func putConfigGroups(ctx context.Context, cfg *pb.Config, project, hash string) error {
	cgLen := len(cfg.GetConfigGroups())
	if cgLen == 0 {
		return nil
	}

	projKey := datastore.MakeKey(ctx, projectConfigKind, project)
	keys := make([]*datastore.Key, cgLen)
	for i, cg := range cfg.GetConfigGroups() {
		keys[i] = datastore.NewKey(ctx, "ProjectConfigGroup",
			string(makeConfigGroupID(hash, cg.GetName(), i)), 0, projKey)
	}

	res, err := datastore.Exists(ctx, keys)
	if err != nil {
		return errors.Annotate(err, "failed to check the existence of ConfigGroups").Tag(transient.Tag).Err()
	}
	entities := make([]ConfigGroup, 0, cgLen)
	for i, cg := range cfg.GetConfigGroups() {
		if res.Get(0, i) {
			continue // already exists
		}
		entities = append(entities, ConfigGroup{
			ID:                ConfigGroupID(keys[i].StringID()),
			Project:           projKey,
			DrainingStartTime: cfg.GetDrainingStartTime(),
			SubmitOptions:     cfg.GetSubmitOptions(),
			Content:           cg,
		})
	}

	// TODO(yiwzhang): batch up to ~10 entities to avoid hitting 10MiB limit of
	// request size: https://cloud.google.com/datastore/docs/concepts/limits
	if err := datastore.Put(ctx, entities); err != nil {
		return errors.Annotate(err, "failed to put ConfigGroups").Tag(transient.Tag).Err()
	}
	return nil
}
