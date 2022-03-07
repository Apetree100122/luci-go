// Copyright 2018 The LUCI Authors.
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

package buildbucket

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	buildbucketpb "go.chromium.org/luci/buildbucket/proto"
	swarmingAPI "go.chromium.org/luci/common/api/swarming/swarming/v1"
	"go.chromium.org/luci/common/clock"
	"go.chromium.org/luci/common/data/stringset"
	"go.chromium.org/luci/common/data/strpair"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/common/sync/parallel"
	"go.chromium.org/luci/gae/service/datastore"
	"go.chromium.org/luci/milo/buildsource/swarming"
	"go.chromium.org/luci/milo/common"
	"go.chromium.org/luci/milo/common/model"
	"go.chromium.org/luci/milo/common/model/milostatus"
	"go.chromium.org/luci/milo/frontend/ui"
	"go.chromium.org/luci/server/auth"
)

func getPool(c context.Context, bid *buildbucketpb.BuilderID) (*ui.MachinePool, error) {
	// Get PoolKey
	builderPool := model.BuilderPool{
		BuilderID: datastore.MakeKey(c, model.BuilderSummaryKind, common.LegacyBuilderIDString(bid)),
	}
	// These are eventually consistent, so just log an error and pass if not found.
	switch err := datastore.Get(c, &builderPool); {
	case datastore.IsErrNoSuchEntity(err):
		logging.Warningf(c, "builder pool not found")
		return nil, nil
	case err != nil:
		return nil, err
	}
	// Get BotPool
	botPool := &model.BotPool{PoolID: builderPool.PoolKey.StringID()}
	switch err := datastore.Get(c, botPool); {
	case datastore.IsErrNoSuchEntity(err):
		logging.Warningf(c, "bot pool not found")
		return nil, nil
	case err != nil:
		return nil, err
	}
	return ui.NewMachinePool(c, botPool), nil
}

// stripDimensionExpiration removes dimension expiration if it exists.
// e.g. "60:key:value" -> "key:value".
func stripDimensionExpiration(dims []string) []string {
	result := strpair.Map{}
	for _, dim := range dims {
		splitted := strings.Split(dim, ":")

		key := splitted[0]
		value := splitted[1]
		if len(splitted) == 3 {
			key = splitted[1]
			value = splitted[2]
		}

		result.Add(key, value)
	}
	return result.Format()
}

// processBuilders saves the builder information into the datastore then returns
// a list of PoolDescriptors that needs to be fetched and saved.
func processBuilders(c context.Context, builders []*buildbucketpb.BuilderItem) ([]model.PoolDescriptor, error) {
	var builderPools []model.BuilderPool
	var descriptors []model.PoolDescriptor
	seen := stringset.New(0)

	for _, builder := range builders {

		id := common.LegacyBuilderIDString(builder.Id)
		dimensions := stripDimensionExpiration(builder.Config.Dimensions)
		descriptor := model.NewPoolDescriptor(builder.Config.SwarmingHost, dimensions)
		dID := descriptor.PoolID()
		builderPools = append(builderPools, model.BuilderPool{
			BuilderID: datastore.MakeKey(c, model.BuilderSummaryKind, id),
			PoolKey:   datastore.MakeKey(c, model.BotPoolKind, dID),
		})
		if added := seen.Add(dID); added {
			descriptors = append(descriptors, descriptor)
		}
	}

	return descriptors, datastore.Put(c, builderPools)
}

// parseBot parses a Swarming BotInfo response into the structure we will
// save into the datastore.  Since BotInfo doesn't have an explicit status
// field that matches Milo's abstraction of a Bot, the status is inferred:
// * A bot with TaskID is Busy
// * A bot that is dead or quarantined is Offline
// * Otherwise, it is implicitly connected and Idle.
func parseBot(c context.Context, swarmingHost string, botInfo *swarmingAPI.SwarmingRpcsBotInfo) (*model.Bot, error) {
	lastSeen, err := time.Parse(swarming.SwarmingTimeLayout, botInfo.LastSeenTs)
	if err != nil {
		return nil, err
	}
	result := &model.Bot{
		Name:     botInfo.BotId,
		URL:      fmt.Sprintf("https://%s/bot?id=%s", swarmingHost, botInfo.BotId),
		LastSeen: lastSeen,
	}

	switch {
	case botInfo.TaskId != "" || botInfo.MaintenanceMsg != "":
		result.Status = milostatus.Busy
	case botInfo.IsDead || botInfo.Quarantined:
		result.Status = milostatus.Offline
	default:
		// Defaults to idle.
	}
	return result, nil
}

// processBot retrieves the Bot pool details from Swarming for a given set of
// dimensions for its respective Swarming host, and saves the data into datastore.
func processBot(c context.Context, desc model.PoolDescriptor) error {
	t, err := auth.GetRPCTransport(c, auth.AsSelf)
	if err != nil {
		return err
	}
	sc, err := swarmingAPI.New(&http.Client{Transport: t})
	if err != nil {
		return err
	}
	sc.BasePath = fmt.Sprintf("https://%s/_ah/api/swarming/v1/", desc.Host())

	var bots []model.Bot
	bl := sc.Bots.List().Dimensions(desc.Dimensions().Format()...)
	// Keep fetching until the cursor is empty.
	for {
		botList, err := bl.Do()
		if err != nil {
			return err
		}
		for _, botInfo := range botList.Items {
			// Ignore deleted bots.
			if botInfo.Deleted {
				continue
			}
			bot, err := parseBot(c, desc.Host(), botInfo)
			if err != nil {
				return err
			}
			bots = append(bots, *bot)
		}

		if botList.Cursor == "" {
			break
		}
		bl = bl.Cursor(botList.Cursor)
	}
	// If there are too many bots, then it won't fit in datastore.
	// Only store a subset of the bots.
	// TODO(hinoka): This is inaccurate, but will only affect few builders.
	// Instead of chopping this list off, just store the statistics.
	if len(bots) > 1000 {
		bots = bots[:1000]
	}
	// This is a large RPC, don't try to batch it.
	return datastore.Put(c, &model.BotPool{
		PoolID:     desc.PoolID(),
		Descriptor: desc,
		Bots:       bots,
		LastUpdate: clock.Now(c),
	})
}

// fetchBotPools resolves the descriptors into actual BotPool information.
// The input is a list of descriptors to fetch from swarming.
// Basically this just runs processBot() a bunch of times.
func processBots(c context.Context, descriptors []model.PoolDescriptor) error {
	return parallel.WorkPool(8, func(ch chan<- func() error) {
		for _, desc := range descriptors {
			desc := desc
			ch <- func() error {
				return processBot(c, desc)
			}
		}
	})
}

// UpdatePools is a cron job endpoint that:
// 1. Fetches all the builders from our associated buildbucket instance.
// 2. Consolidates all known descriptors (host+dimensions), saves BuilderPool.
// 3. Fetches and saves BotPool data from swarming for all known descriptors.
func UpdatePools(c context.Context) error {
	host, err := getHost(c)
	if err != nil {
		return err
	}

	buildersClient, err := ProdBuildersClientFactory(c, host, auth.AsSelf)
	if err != nil {
		return err
	}

	// Get all the builders from buildbucket.
	builders := make([]*buildbucketpb.BuilderItem, 0)
	req := &buildbucketpb.ListBuildersRequest{PageSize: 1000}
	for {
		r, err := buildersClient.ListBuilders(c, req)
		if err != nil {
			return err
		}
		builders = append(builders, r.Builders...)
		if r.NextPageToken == "" {
			break
		}
		req.PageToken = r.NextPageToken
	}
	logging.Infof(c, "got %d builders from buildbucket", len(builders))

	// Process builders and save them.  We get back the descriptors that we have
	// to fetch next.
	descriptors, err := processBuilders(c, builders)
	if err != nil {
		return errors.Annotate(err, "processing builders").Err()
	}
	// And now also fetch and save the BotPools.
	return processBots(c, descriptors)
}
