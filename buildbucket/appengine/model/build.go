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

package model

import (
	"context"
	"fmt"
	"time"

	"github.com/golang/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"go.chromium.org/luci/auth/identity"
	"go.chromium.org/luci/common/clock"
	"go.chromium.org/luci/common/data/strpair"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/proto/mask"
	"go.chromium.org/luci/gae/service/datastore"

	bb "go.chromium.org/luci/buildbucket"
	pb "go.chromium.org/luci/buildbucket/proto"
	"go.chromium.org/luci/buildbucket/protoutil"
)

const (
	// BuildKind is a Build entity's kind in the datastore.
	BuildKind = "Build"
)

// isHiddenTag returns whether the given tag should be hidden by ToProto.
func isHiddenTag(key string) bool {
	// build_address is reserved by the server so that the TagIndex infrastructure
	// can be reused to fetch builds by builder + number (see tagindex.go and
	// rpc/get_build.go).
	// TODO(crbug/1042991): Unhide builder and gitiles_ref.
	// builder and gitiles_ref are allowed to be specified, are not internal,
	// and are only hidden here to match Python behavior.
	return key == "build_address" || key == "builder" || key == "gitiles_ref"
}

// PubSubCallback encapsulates parameters for a Pub/Sub callback.
type PubSubCallback struct {
	AuthToken string `gae:"auth_token,noindex"`
	Topic     string `gae:"topic,noindex"`
	UserData  string `gae:"user_data,noindex"`
}

// Build is a representation of a build in the datastore.
// Implements datastore.PropertyLoadSaver.
type Build struct {
	_     datastore.PropertyMap `gae:"-,extra"`
	_kind string                `gae:"$kind,Build"`
	ID    int64                 `gae:"$id"`

	// LegacyProperties are properties set for v1 legacy builds.
	LegacyProperties
	// UnusedProperties are properties set previously but currently unused.
	UnusedProperties

	// Proto is the pb.Build proto representation of the build.
	//
	// infra, input.properties, output.properties, and steps
	// are zeroed and stored in separate datastore entities
	// due to their potentially large size (see details.go).
	// tags are given their own field so they can be indexed.
	//
	// noindex is not respected here, it's set in pb.Build.ToProperty.
	Proto pb.Build `gae:"proto,noindex"`

	Project string `gae:"project"`
	// <project>/<bucket>. Bucket is in v2 format.
	// e.g. chromium/try (never chromium/luci.chromium.try).
	BucketID string `gae:"bucket_id"`
	// <project>/<bucket>/<builder>. Bucket is in v2 format.
	// e.g. chromium/try/linux-rel.
	BuilderID string `gae:"builder_id"`

	Canary bool `gae:"canary"`

	CreatedBy identity.Identity `gae:"created_by"`
	// TODO(nodir): Replace reliance on create_time indices with id.
	CreateTime time.Time `gae:"create_time"`
	// Experimental, if true, means to exclude from monitoring and search results
	// (unless specifically requested in search results).
	Experimental bool `gae:"experimental"`
	// Experiments is a slice of experiments enabled or disabled on this build.
	// Each element should look like "[-+]$experiment_name".
	Experiments []string `gae:"experiments"`
	Incomplete  bool     `gae:"incomplete"`

	// Deprecated; remove after v1 api turndown
	IsLuci bool `gae:"is_luci"`

	ResultDBUpdateToken string    `gae:"resultdb_update_token,noindex"`
	Status              pb.Status `gae:"status_v2"`
	StatusChangedTime   time.Time `gae:"status_changed_time"`
	// Tags is a slice of "<key>:<value>" strings taken from Proto.Tags.
	// Stored separately in order to index.
	Tags []string `gae:"tags"`

	// UpdateToken is set at the build creation time, and UpdateBuild requests are required
	// to have it in the header.
	UpdateToken string `gae:"update_token,noindex"`

	// PubSubCallback, if set, creates notifications for build status changes.
	PubSubCallback PubSubCallback `gae:"pubsub_callback,noindex"`
}

// Realm returns this build's auth realm, or an empty string if not opted into the
// realms experiment.
func (b *Build) Realm() string {
	if b.ExperimentStatus(bb.ExperimentUseRealms) == pb.Trinary_YES {
		return fmt.Sprintf("%s:%s", b.Proto.Builder.Project, b.Proto.Builder.Bucket)
	}
	return ""
}

// ExperimentStatus scans the experiments attached to this Build and returns:
//   * YES - The experiment was known at schedule time and enabled.
//   * NO - The experiment was known at schedule time and disabled.
//   * UNSET - The experiment was unknown at schedule time.
//
// Malformed Experiment filters are treated as UNSET.
func (b *Build) ExperimentStatus(expname string) (ret pb.Trinary) {
	b.IterExperiments(func(enabled bool, exp string) bool {
		if exp == expname {
			if enabled {
				ret = pb.Trinary_YES
			} else {
				ret = pb.Trinary_NO
			}
			return false
		}
		return true
	})
	return
}

// IterExperiments parses all experiments and calls `cb` for each.
//
// This will always include a call with bb.ExperimentNonProduction, even
// if '-'+bb.ExperimentNonProduction isn't recorded in the underlying
// Experiments field.
func (b *Build) IterExperiments(cb func(enabled bool, exp string) bool) {
	var hadNonProd bool

	for _, expFilter := range b.Experiments {
		if len(expFilter) == 0 {
			continue
		}
		plusMinus, exp := expFilter[0], expFilter[1:]
		hadNonProd = hadNonProd || exp == bb.ExperimentNonProduction

		keepGoing := true
		if plusMinus == '+' {
			keepGoing = cb(true, exp)
		} else if plusMinus == '-' {
			keepGoing = cb(false, exp)
		}
		if !keepGoing {
			return
		}
	}
	if !hadNonProd {
		cb(false, bb.ExperimentNonProduction)
	}
}

// Load overwrites this representation of a build by reading the given
// datastore.PropertyMap. Mutates this entity.
func (b *Build) Load(p datastore.PropertyMap) error {
	return datastore.GetPLS(b).Load(p)
}

var buildUpdateTimeClock = clock.GetSystemClock()

// OverrideGlobalBuildUpdateTimeClock allows you to override the clock used
// for setting the Build.Proto.UpdateTime field.
//
// This should only be used in tests, though use appropriate caution since this
// is a global variable.
//
// Overriding with `nil` will result in UpdateTime not being manipulated at all.
//
// Returns a function to undo the manipulation.
//
// Example Usage:
//
//    ctx, testClock := testclock.UseTime(ctx, ...)
//    defer OverrideGlobalBuildUpdateTimeClock(testClock)()
func OverrideGlobalBuildUpdateTimeClock(c clock.Clock) (undo func()) {
	oldVal := buildUpdateTimeClock
	buildUpdateTimeClock = c
	return func() {
		buildUpdateTimeClock = oldVal
	}
}

// Save returns the datastore.PropertyMap representation of this build. Mutates
// this entity to reflect computed datastore fields in the returned PropertyMap.
func (b *Build) Save(withMeta bool) (datastore.PropertyMap, error) {
	b.BucketID = protoutil.FormatBucketID(b.Proto.Builder.Project, b.Proto.Builder.Bucket)
	b.BuilderID = protoutil.FormatBuilderID(b.Proto.Builder)
	b.Canary = b.Proto.Canary
	b.Experimental = b.Proto.Input.GetExperimental()
	b.Incomplete = !protoutil.IsEnded(b.Proto.Status)
	b.Project = b.Proto.Builder.Project
	b.Status = b.Proto.Status
	if c := buildUpdateTimeClock; c != nil {
		b.Proto.UpdateTime = timestamppb.New(c.Now())
	}
	p, err := datastore.GetPLS(b).Save(withMeta)
	if err != nil {
		return nil, err
	}
	// ResultDetails is only set via v1 API. For builds only manipulated with the
	// v2 API, this column will be missing in the datastore in Python. Python
	// interprets this field as JSON, and a missing value is loaded as None which
	// Python loads as the empty JSON dict. However in Go, in order to preserve
	// the value of this field without having to interpret it, the type is set
	// to []byte. But if the build has no result details, Go will store the empty
	// value for this type (i.e. an empty string), which is not a valid JSON dict.
	// For the benefit of the Python service, default the value to null in the
	// datastore which Python will handle correctly.
	// TODO(crbug/1042991): Remove ResultDetails default once v1 API is removed.
	if len(b.LegacyProperties.ResultDetails) == 0 {
		p["result_details"] = datastore.MkProperty(nil)
	}
	// Writing a value for PubSubCallback confuses the Python implementation which
	// expects PubSubCallback to be a LocalStructuredProperty. See also unused.go.
	delete(p, "pubsub_callback")
	return p, nil
}

// ToProto returns the *pb.Build representation of this build.
func (b *Build) ToProto(ctx context.Context, m *mask.Mask) (*pb.Build, error) {
	build := b.ToSimpleBuildProto(ctx)
	if err := LoadBuildDetails(ctx, m, build); err != nil {
		return nil, err
	}
	return build, nil
}

// ToSimpleBuildProto returns the *pb.Build without loading steps, infra,
// input/output properties.
func (b *Build) ToSimpleBuildProto(ctx context.Context) *pb.Build {
	p := proto.Clone(&b.Proto).(*pb.Build)
	for _, t := range b.Tags {
		k, v := strpair.Parse(t)
		if !isHiddenTag(k) {
			p.Tags = append(p.Tags, &pb.StringPair{
				Key:   k,
				Value: v,
			})
		}
	}
	return p
}

// LoadBuildDetails loads the details of the given builds, trimming them
// according to the mask.
func LoadBuildDetails(ctx context.Context, m *mask.Mask, builds ...*pb.Build) error {
	l := len(builds)
	inf := make([]*BuildInfra, 0, l)
	inp := make([]*BuildInputProperties, 0, l)
	out := make([]*BuildOutputProperties, 0, l)
	stp := make([]*BuildSteps, 0, l)
	var dets []interface{}
	var err error
	isIncluded := func(path string) bool {
		switch inc, e := m.Includes(path); {
		case e != nil:
			err = errors.Annotate(err, "error checking %q field inclusiveness", path).Err()
		case inc != mask.Exclude:
			return true
		}
		return false
	}
	included := map[string]bool{
		"infra":             isIncluded("infra"),
		"input.properties":  isIncluded("input.properties"),
		"output.properties": isIncluded("output.properties"),
		"steps":             isIncluded("steps"),
	}
	if err != nil {
		return err
	}
	for i, p := range builds {
		if p.GetId() <= 0 {
			return errors.Reason("invalid build for %q", p).Err()
		}
		key := datastore.KeyForObj(ctx, &Build{ID: p.Id})
		inf = append(inf, &BuildInfra{Build: key})
		inp = append(inp, &BuildInputProperties{Build: key})
		out = append(out, &BuildOutputProperties{Build: key})
		stp = append(stp, &BuildSteps{Build: key})
		appendIfIncluded := func(path string, det interface{}) {
			if included[path] {
				dets = append(dets, det)
			}
		}
		appendIfIncluded("infra", inf[i])
		appendIfIncluded("input.properties", inp[i])
		appendIfIncluded("output.properties", out[i])
		appendIfIncluded("steps", stp[i])
		if err != nil {
			return err
		}
	}
	if err := GetIgnoreMissing(ctx, dets); err != nil {
		return errors.Annotate(err, "error fetching build details").Err()
	}

	for i, p := range builds {
		p.Infra = &inf[i].Proto.BuildInfra
		if p.Input == nil {
			p.Input = &pb.Build_Input{}
		}
		p.Input.Properties = &inp[i].Proto.Struct
		if p.Output == nil {
			p.Output = &pb.Build_Output{}
		}
		p.Output.Properties = &out[i].Proto.Struct
		p.Steps, err = stp[i].ToProto(ctx)
		if err != nil {
			return errors.Annotate(err, "error fetching steps for build %q", p).Err()
		}
		if err := m.Trim(p); err != nil {
			return errors.Annotate(err, "error trimming fields for %q", p).Err()
		}
	}
	return nil
}

// GetBuildAndBucket returns the build with the given ID as well as the bucket
// it belongs to. Returns datastore.ErrNoSuchEntity if either is not found.
func GetBuildAndBucket(ctx context.Context, id int64) (*Build, *Bucket, error) {
	bld := &Build{
		ID: id,
	}
	switch err := datastore.Get(ctx, bld); {
	case err == datastore.ErrNoSuchEntity:
		return nil, nil, err
	case err != nil:
		return nil, nil, errors.Annotate(err, "error fetching build with ID %d", id).Err()
	}
	bck := &Bucket{
		ID:     bld.Proto.Builder.Bucket,
		Parent: ProjectKey(ctx, bld.Proto.Builder.Project),
	}
	switch err := datastore.Get(ctx, bck); {
	case err == datastore.ErrNoSuchEntity:
		return nil, nil, err
	case err != nil:
		return nil, nil, errors.Annotate(err, "error fetching bucket %q", bld.BucketID).Err()
	}
	return bld, bck, nil
}
