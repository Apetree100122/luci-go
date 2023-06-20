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

package rpc

import (
	"context"
	"fmt"

	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/grpc/appstatus"

	"go.chromium.org/luci/buildbucket/appengine/common"
	"go.chromium.org/luci/buildbucket/appengine/internal/perm"
	"go.chromium.org/luci/buildbucket/appengine/model"
	"go.chromium.org/luci/buildbucket/bbperms"
	pb "go.chromium.org/luci/buildbucket/proto"
	"go.chromium.org/luci/buildbucket/protoutil"
)

// validateGet validates the given request.
func validateGet(req *pb.GetBuildRequest) error {
	switch {
	case req.GetId() != 0:
		if req.Builder != nil || req.BuildNumber != 0 {
			return errors.Reason("id is mutually exclusive with (builder and build_number)").Err()
		}
	case req.GetBuilder() != nil && req.BuildNumber != 0:
		if err := protoutil.ValidateRequiredBuilderID(req.Builder); err != nil {
			return errors.Annotate(err, "builder").Err()
		}
	default:
		return errors.Reason("one of id or (builder and build_number) is required").Err()
	}
	return nil
}

func getBuildIDByBuildNumber(ctx context.Context, bldr *pb.BuilderID, nbr int32) (int64, error) {
	addr := fmt.Sprintf("luci.%s.%s/%s/%d", bldr.Project, bldr.Bucket, bldr.Builder, nbr)
	switch ents, err := model.SearchTagIndex(ctx, "build_address", addr); {
	case model.TagIndexIncomplete.In(err):
		// Shouldn't happen because build address is globally unique (exactly one entry in a complete index).
		return 0, errors.Reason("unexpected incomplete index for build address %q", addr).Err()
	case err != nil:
		return 0, err
	case len(ents) == 0:
		return 0, perm.NotFoundErr(ctx)
	case len(ents) == 1:
		return ents[0].BuildID, nil
	default:
		// Shouldn't happen because build address is globally unique and created before the build.
		return 0, errors.Reason("unexpected number of results for build address %q: %d", addr, len(ents)).Err()
	}
}

// GetBuild handles a request to retrieve a build. Implements pb.BuildsServer.
func (*Builds) GetBuild(ctx context.Context, req *pb.GetBuildRequest) (*pb.Build, error) {
	if err := validateGet(req); err != nil {
		return nil, appstatus.BadRequest(err)
	}
	m, err := model.NewBuildMask("", req.Fields, req.Mask)
	if err != nil {
		return nil, appstatus.BadRequest(errors.Annotate(err, "invalid mask").Err())
	}
	if req.Id == 0 {
		req.Id, err = getBuildIDByBuildNumber(ctx, req.Builder, req.BuildNumber)
		if err != nil {
			return nil, err
		}
	}

	bld, err := common.GetBuild(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	// User needs BuildsGet or BuildsGetLimited permission to call this endpoint.
	readPerm, err := perm.GetFirstAvailablePerm(ctx, bld.Proto.Builder, bbperms.BuildsGet, bbperms.BuildsGetLimited)
	if err != nil {
		return nil, err
	}

	bp, err := bld.ToProto(ctx, m, func(b *pb.Build) error {
		if readPerm == bbperms.BuildsGet {
			return nil
		}
		return perm.RedactBuild(ctx, nil, b)
	})
	if err != nil {
		return nil, err
	}

	return bp, nil

}
