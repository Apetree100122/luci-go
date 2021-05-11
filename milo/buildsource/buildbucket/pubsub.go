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

package buildbucket

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/golang/protobuf/ptypes"
	"google.golang.org/genproto/protobuf/field_mask"
	"google.golang.org/protobuf/types/known/timestamppb"

	"go.chromium.org/luci/buildbucket/deprecated"
	buildbucketpb "go.chromium.org/luci/buildbucket/proto"
	"go.chromium.org/luci/buildbucket/protoutil"
	bbv1 "go.chromium.org/luci/common/api/buildbucket/buildbucket/v1"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/common/retry/transient"
	"go.chromium.org/luci/common/tsmon/field"
	"go.chromium.org/luci/common/tsmon/metric"
	"go.chromium.org/luci/gae/service/datastore"
	"go.chromium.org/luci/milo/common"
	"go.chromium.org/luci/milo/common/model"
	"go.chromium.org/luci/server/auth"
	"go.chromium.org/luci/server/router"
)

var (
	buildCounter = metric.NewCounter(
		"luci/milo/buildbucket_pubsub/builds",
		"The number of buildbucket builds received by Milo from PubSub",
		nil,
		field.String("bucket"),
		// True for luci builds; should always be true.
		field.Bool("luci"),
		// Status can be "COMPLETED", "SCHEDULED", or "STARTED"
		field.String("status"),
		// Action can be one of 3 options.
		//   * "Created" - This is the first time Milo heard about this build
		//   * "Modified" - Milo updated some information about this build vs. what
		//     it knew before.
		//   * "Rejected" - Milo was unable to accept this build.
		field.String("action"))
)

// PubSubHandler is a webhook that stores the builds coming in from pubsub.
func PubSubHandler(ctx *router.Context) {
	err := pubSubHandlerImpl(ctx.Context, ctx.Request)
	if err != nil {
		logging.Errorf(ctx.Context, "error while handling pubsub event")
		errors.Log(ctx.Context, err)
	}
	if transient.Tag.In(err) {
		// Transient errors are 500 so that PubSub retries them.
		ctx.Writer.WriteHeader(http.StatusInternalServerError)
		return
	}
	// No errors or non-transient errors are 200s so that PubSub does not retry
	// them.
	ctx.Writer.WriteHeader(http.StatusOK)
}

func mustTimestamp(ts *timestamppb.Timestamp) time.Time {
	if t, err := ptypes.Timestamp(ts); err == nil {
		return t
	}
	return time.Time{}
}

var summaryBuildMask = &field_mask.FieldMask{
	Paths: []string{
		"id",
		"builder",
		"number",
		"create_time",
		"start_time",
		"end_time",
		"update_time",
		"status",
		"summary_markdown",
		"tags",
		"infra.swarming",
		"input.experimental",
		"input.gitiles_commit",
		"output.properties",
		"critical",
	},
}

// getSummary returns a model.BuildSummary representing a buildbucket build.
func getSummary(c context.Context, host string, project string, id int64) (*model.BuildSummary, error) {
	client, err := buildbucketBuildsClient(c, host, auth.AsProject, auth.WithProject(project))
	if err != nil {
		return nil, err
	}
	b, err := client.GetBuild(c, &buildbucketpb.GetBuildRequest{
		Id:     id,
		Fields: summaryBuildMask,
	})
	if err != nil {
		return nil, err
	}
	buildAddress := fmt.Sprintf("%d", b.Id)
	if b.Number != 0 {
		buildAddress = fmt.Sprintf("luci.%s.%s/%s/%d", b.Builder.Project, b.Builder.Bucket, b.Builder.Builder, b.Number)
	}

	// Note: The parent for buildbucket build summaries is currently a fake entity.
	// In the future, builds can be cached here, but we currently don't do that.
	buildKey := datastore.MakeKey(c, "buildbucket.Build", fmt.Sprintf("%s:%s", host, buildAddress))
	swarming := b.GetInfra().GetSwarming()

	type OutputProperties struct {
		BlamelistPins []*buildbucketpb.GitilesCommit `json:"$recipe_engine/milo/blamelist_pins"`
	}
	var outputProperties OutputProperties
	propertiesJSON, err := b.GetOutput().GetProperties().MarshalJSON()
	err = json.Unmarshal(propertiesJSON, &outputProperties)
	if err != nil {
		logging.Warningf(c, "failed to decode test build set")
		return nil, err
	}

	var blamelistPins []string
	if len(outputProperties.BlamelistPins) > 0 {
		blamelistPins = make([]string, len(outputProperties.BlamelistPins))
		for i, gc := range outputProperties.BlamelistPins {
			blamelistPins[i] = protoutil.GitilesBuildSet(gc)
		}
	} else if gc := b.GetInput().GetGitilesCommit(); gc != nil {
		// Fallback to Input.GitilesCommit when there are no blamelist pins.
		blamelistPins = []string{protoutil.GitilesBuildSet(gc)}
	}

	bs := &model.BuildSummary{
		ProjectID:     b.Builder.Project,
		BuildKey:      buildKey,
		BuilderID:     common.LegacyBuilderIDString(b.Builder),
		BuildID:       "buildbucket/" + buildAddress,
		BuildSet:      protoutil.BuildSets(b),
		BlamelistPins: blamelistPins,
		ContextURI:    []string{fmt.Sprintf("buildbucket://%s/build/%d", host, id)},
		Created:       mustTimestamp(b.CreateTime),
		Summary: model.Summary{
			Start:  mustTimestamp(b.StartTime),
			End:    mustTimestamp(b.EndTime),
			Status: statusMap[b.Status],
		},
		Version:      mustTimestamp(b.UpdateTime).UnixNano(),
		Experimental: b.GetInput().GetExperimental(),
		Critical:     b.GetCritical(),
	}
	if task := swarming.GetTaskId(); task != "" {
		bs.ContextURI = append(
			bs.ContextURI,
			fmt.Sprintf("swarming://%s/task/%s", swarming.GetHostname(), swarming.GetTaskId()))
	}
	return bs, nil
}

// pubSubHandlerImpl takes the http.Request, expects to find
// a common.PubSubSubscription JSON object in the Body, containing a bbPSEvent,
// and handles the contents with generateSummary.
func pubSubHandlerImpl(c context.Context, r *http.Request) error {
	// This is the default action. The code below will modify the values of some
	// or all of these parameters.
	isLUCI, bucket, status, action := false, "UNKNOWN", "UNKNOWN", "Rejected"

	defer func() {
		// closure for late binding
		buildCounter.Add(c, 1, bucket, isLUCI, status, action)
	}()

	msg := common.PubSubSubscription{}
	if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
		// This might be a transient error, e.g. when the json format changes
		// and Milo isn't updated yet.
		return errors.Annotate(err, "could not decode message").Tag(transient.Tag).Err()
	}
	if v, ok := msg.Message.Attributes["version"].(string); ok && v != "v1" {
		// TODO(nodir): switch to v2, crbug.com/826006
		logging.Debugf(c, "unsupported pubsub message version %q. Ignoring", v)
		return nil
	}
	bData, err := msg.GetData()
	if err != nil {
		return errors.Annotate(err, "could not parse pubsub message string").Err()
	}

	event := struct {
		Build    bbv1.LegacyApiCommonBuildMessage `json:"build"`
		Hostname string                           `json:"hostname"`
	}{}
	if err := json.Unmarshal(bData, &event); err != nil {
		return errors.Annotate(err, "could not parse pubsub message data").Err()
	}

	build, err := deprecated.BuildToV2(&event.Build)
	if err != nil {
		return errors.Annotate(err, "could not parse deprecated.Build").Err()
	}

	status = build.Status.String()

	logging.Debugf(c, "Received from %s: build %s/%s/%s/%d (%s)\n%v",
		event.Hostname, build.Builder.Project, build.Builder.Bucket, build.Builder.Builder, build.Id, status, build)

	if build.Builder.Bucket == "" {
		logging.Infof(c, "This is not an ingestable build, ignoring")
		return nil
	}

	proj, err := common.GetProject(c, build.Builder.Project)
	if err != nil {
		return errors.Annotate(err, "could not get project configuration").Err()
	}
	if proj.BuilderIsIgnored(build.Builder) {
		logging.Infof(c, "This is from an ignored builder, ignoring")
		return nil
	}

	// TODO(iannucci,nodir): get the bot context too
	// TODO(iannucci,nodir): support manifests/got_revision
	bs, err := getSummary(c, event.Hostname, build.Builder.Project, build.Id)
	if err != nil {
		return err
	}
	if err := bs.AddManifestKeysFromBuildSets(c); err != nil {
		return err
	}

	return transient.Tag.Apply(datastore.RunInTransaction(c, func(c context.Context) error {
		curBS := &model.BuildSummary{BuildKey: bs.BuildKey}
		switch err := datastore.Get(c, curBS); err {
		case datastore.ErrNoSuchEntity:
			action = "Created"
		case nil:
			action = "Modified"
		default:
			return errors.Annotate(err, "reading current BuildSummary").Err()
		}

		if bs.Version <= curBS.Version {
			logging.Warningf(c, "current BuildSummary is newer: %d <= %d",
				bs.Version, curBS.Version)
			return nil
		}

		if err := datastore.Put(c, bs); err != nil {
			return err
		}

		return model.UpdateBuilderForBuild(c, bs)
	}, nil))
}

// MakeBuildKey returns a new datastore Key for a buildbucket.Build.
//
// There's currently no model associated with this key, but it's used as
// a parent for a model.BuildSummary.
func MakeBuildKey(c context.Context, host, buildAddress string) *datastore.Key {
	return datastore.MakeKey(c,
		"buildbucket.Build", fmt.Sprintf("%s:%s", host, buildAddress))
}
