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

package gobmap

import (
	"context"

	"google.golang.org/protobuf/types/known/timestamppb"

	"go.chromium.org/luci/common/clock"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/retry/transient"
	"go.chromium.org/luci/gae/service/datastore"

	"go.chromium.org/luci/cv/internal/changelist"
	"go.chromium.org/luci/cv/internal/config"
	"go.chromium.org/luci/cv/internal/gerrit/cfgmatcher"
)

const (
	mapKind    = "MapPart"
	parentKind = "MapPartParent"
)

// mapPart contains config groups for a particular LUCI project and host/repo
// combination.
//
// MapPart entities are stored with a parent key of the form (MapPartParent,
// host/repo), so that all mapPart entities with a particular host/repo can be
// fetched with an ancestor query; the goal is to have fast reads by host/repo.
//
// The MapPart entities as a whole store a mapping used to lookup which host,
// repo and ref maps to which config group; the map is updated when a project
// config is updated.
type mapPart struct {
	_kind string `gae:"$kind,MapPart"`

	// The ID of this MapPart, which is the LUCI project name.
	ID string `gae:"$id"`

	// LUCI project name.
	//
	// This field contains the same value as ID, and is included so
	// that we can index on it, and thus filter on it in queries.
	Project string

	// MapPartParent key. This parent has an ID of the form "host/repo".
	Parent *datastore.Key `gae:"$parent"`

	// Groups keeps config groups of a LUCI project applicable to this
	// host/repo.
	Groups *cfgmatcher.Groups

	// ConfigHash is the hash of latest CV project config imported from LUCI
	// Config; this is updated based on ProjectConfig entity.
	ConfigHash string `gae:",noindex"`
}

// Update loads the new config and updates the gob map entities
// accordingly.
//
// This may include adding, removing and modifying entities.
//
// TODO(qyearsley): Handle possible race condition; There may be 2 or more
// concurrent Update calls, which could lead to corrupted data.
func Update(ctx context.Context, project string) error {
	var toPut, toDelete []*mapPart

	// Fetch stored GWM entities
	mps := []*mapPart{}
	q := datastore.NewQuery(mapKind).Eq("Project", project)
	if err := datastore.GetAll(ctx, q, &mps); err != nil {
		return errors.Annotate(err, "failed to get MapPart entities for project %q", project).Tag(transient.Tag).Err()
	}

	switch meta, err := config.GetLatestMeta(ctx, project); {
	case err != nil:
		return err
	case meta.Status != config.StatusEnabled:
		// The project was disabled or removed, delete everything.
		toDelete = mps
	default:
		cgs, err := meta.GetConfigGroups(ctx)
		if err != nil {
			return err
		}
		toPut, toDelete = listUpdates(ctx, mps, cgs, meta.Hash(), project)
	}

	// TODO(qyearsle): split Delete/Put into batches of ~128 and execute in
	// parallel. Reason: some LUCI projects watch 100s of repos, while
	// Delete/Put are limited to ~500 entities.
	if err := datastore.Delete(ctx, toDelete); err != nil {
		return errors.Annotate(err, "failed to delete %d MapPart entities when updating project %q",
			len(toDelete), project).Tag(transient.Tag).Err()
	}
	if err := datastore.Put(ctx, toPut); err != nil {
		return errors.Annotate(err, "failed to put %d MapPart entities when updating project %q",
			len(toPut), project).Tag(transient.Tag).Err()
	}

	return nil
}

// listUpdates determines what needs to be updated.
//
// It computes which of the existing MapPart entities should be
// removed, and which MapPart entities should be put (added or updated).
func listUpdates(ctx context.Context, mps []*mapPart, latestConfigGroups []*config.ConfigGroup,
	latestHash, project string) (toPut, toDelete []*mapPart) {
	// Make a map of host/repo to config hashes for currently
	// existing MapPart entities; used below.
	existingHashes := make(map[string]string, len(mps))
	for _, mp := range mps {
		hostRepo := mp.Parent.StringID()
		existingHashes[hostRepo] = mp.ConfigHash
	}

	// List `internal.Groups` present in the latest config groups.
	hostRepoToGroups := internalGroups(latestConfigGroups)

	// List MapParts to put; these are those either have
	// no existing hash yet or a different existing hash.
	for hostRepo, groups := range hostRepoToGroups {
		if existingHashes[hostRepo] == latestHash {
			// Already up to date.
			continue
		}
		mp := &mapPart{
			ID:         project,
			Project:    project,
			Parent:     datastore.MakeKey(ctx, parentKind, hostRepo),
			Groups:     groups,
			ConfigHash: latestHash,
		}
		toPut = append(toPut, mp)
	}

	// List MapParts to delete; these are those that currently exist but
	// have no groups in the latest config.
	toDelete = []*mapPart{}
	for _, mp := range mps {
		hostRepo := mp.Parent.StringID()
		if _, ok := hostRepoToGroups[hostRepo]; !ok {
			toDelete = append(toDelete, mp)
		}
	}

	return toPut, toDelete
}

// internalGroups converts config.ConfigGroups to cfgmatcher.Groups.
//
// It returns a map of host/repo to cfgmatcher.Groups.
func internalGroups(configGroups []*config.ConfigGroup) map[string]*cfgmatcher.Groups {
	ret := make(map[string]*cfgmatcher.Groups)
	for _, g := range configGroups {
		for _, gerrit := range g.Content.Gerrit {
			host := config.GerritHost(gerrit)
			for _, p := range gerrit.Projects {
				hostRepo := host + "/" + p.Name
				group := cfgmatcher.MakeGroup(g, p)
				if groups, ok := ret[hostRepo]; ok {
					groups.Groups = append(groups.Groups, group)
				} else {
					ret[hostRepo] = &cfgmatcher.Groups{Groups: []*cfgmatcher.Group{group}}
				}
			}
		}
	}
	return ret
}

// Lookup returns config group IDs which watch the given combination of Gerrit
// host, repo and ref.
//
// For example: the input might be ("chromium-review.googlesource.com",
// "chromium/src", "refs/heads/main"), and the output might be 0 or 1 or
// multiple IDs which can be used to fetch config groups.
//
// Due to the ref_regexp[_exclude] options, CV can't ensure that each possible
// combination is watched by at most one ConfigGroup, which is why this may
// return multiple ConfigGroupIDs even for the same LUCI project.
func Lookup(ctx context.Context, host, repo, ref string) (*changelist.ApplicableConfig, error) {
	now := timestamppb.New(clock.Now(ctx).UTC())

	// Fetch all MapPart entities for the given host and repo.
	hostRepo := host + "/" + repo
	parentKey := datastore.MakeKey(ctx, parentKind, hostRepo)
	q := datastore.NewQuery(mapKind).Ancestor(parentKey)
	mps := []*mapPart{}
	if err := datastore.GetAll(ctx, q, &mps); err != nil {
		return nil, errors.Annotate(err, "failed to fetch MapParts for %s", hostRepo).Tag(transient.Tag).Err()
	}

	// For each MapPart entity, inspect the Groups to determine which configs
	// apply for the given ref.
	ac := &changelist.ApplicableConfig{UpdateTime: now}
	for _, mp := range mps {
		if groups := mp.Groups.Match(ref); len(groups) != 0 {
			ids := make([]string, len(groups))
			for i, g := range groups {
				ids[i] = g.GetId()
			}
			ac.Projects = append(ac.Projects, &changelist.ApplicableConfig_Project{
				Name:           mp.Project,
				ConfigGroupIds: ids,
			})
		}
	}
	return ac, nil
}
