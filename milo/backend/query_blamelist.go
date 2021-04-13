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

package backend

import (
	"context"
	"encoding/base64"
	"sync"

	"go.chromium.org/luci/auth/identity"
	buildbucketpb "go.chromium.org/luci/buildbucket/proto"
	"go.chromium.org/luci/buildbucket/protoutil"
	"go.chromium.org/luci/common/errors"
	gitpb "go.chromium.org/luci/common/proto/git"
	"go.chromium.org/luci/common/sync/parallel"
	"go.chromium.org/luci/gae/service/datastore"
	"go.chromium.org/luci/grpc/appstatus"
	milopb "go.chromium.org/luci/milo/api/service/v1"
	"go.chromium.org/luci/milo/common"
	"go.chromium.org/luci/milo/common/model"
	"go.chromium.org/luci/milo/git"
	"go.chromium.org/luci/server/auth"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/proto"
)

// QueryBlamelist implements milopb.MiloInternal service
func (s *MiloInternalService) QueryBlamelist(ctx context.Context, req *milopb.QueryBlamelistRequest) (_ *milopb.QueryBlamelistResponse, err error) {
	defer func() { err = appstatus.GRPCifyAndLog(ctx, err) }()

	allowed, err := common.IsAllowed(ctx, req.GetBuilder().GetProject())
	if err != nil {
		return nil, err
	}
	if !allowed {
		if auth.CurrentIdentity(ctx) == identity.AnonymousIdentity {
			return nil, appstatus.Error(codes.Unauthenticated, "not logged in ")
		}
		return nil, appstatus.Error(codes.PermissionDenied, "no access to the project")
	}

	startRev, err := prepareQueryBlamelistRequest(req)
	if err != nil {
		return nil, appstatus.BadRequest(err)
	}

	pageSize := adjustPageSize(req.PageSize)

	// Fetch one more commit to check whether there are more commits in the
	// blamelist.
	opts := &git.LogOptions{Limit: pageSize + 1, WithFiles: true}
	commits, err := git.Get(ctx).Log(ctx, req.GitilesCommit.Host, req.GitilesCommit.Project, startRev, opts)
	if err != nil {
		return nil, err
	}

	q := datastore.NewQuery("BuildSummary").Eq("BuilderID", common.LegacyBuilderIDString(req.Builder))
	commitColumn := "BuildSet"
	if req.MultiProjectSupport {
		commitColumn = "BlamelistPins"
	}
	blameLength := len(commits)
	m := sync.Mutex{}

	// Find the first other commit that has an associated build and update
	// blameLength.
	err = parallel.WorkPool(8, func(c chan<- func() error) {
		// Skip the first commit, it should always be included in the blamelist.
		for i, commit := range commits[1:] {
			newBlameLength := i + 1 // +1 since we skipped the first one.

			m.Lock()
			foundBuild := newBlameLength >= blameLength
			m.Unlock()

			// We have already found a build before this commit, no point looking
			// further.
			if foundBuild {
				break
			}

			curGC := &buildbucketpb.GitilesCommit{Host: req.GitilesCommit.Host, Project: req.GitilesCommit.Project, Id: commit.Id}
			c <- func() error {
				// Check whether this commit has an associated build.
				hasAssociatedBuild := false
				err := datastore.Run(ctx, q.Eq(commitColumn, protoutil.GitilesBuildSet(curGC)), func(build *model.BuildSummary) error {
					switch build.Summary.Status {
					case model.InfraFailure, model.Expired, model.Canceled:
						return nil
					default:
						hasAssociatedBuild = true
						return datastore.Stop
					}
				})
				if err != nil {
					return err
				}

				if hasAssociatedBuild {
					m.Lock()
					if newBlameLength < blameLength {
						blameLength = newBlameLength
					}
					m.Unlock()
				}
				return nil
			}
		}
	})
	if err != nil {
		return nil, err
	}

	// If there's more commits than needed, reserve the last commit as the pivot
	// for the next page.
	nextPageToken := ""
	if blameLength >= pageSize+1 {
		blameLength = pageSize
		nextPageToken, err = serializeQueryBlamelistPageToken(&milopb.QueryBlamelistPageToken{
			NextCommitId: commits[blameLength].Id,
		})
		if err != nil {
			return nil, err
		}
	}

	var precedingCommit *gitpb.Commit
	if blameLength < len(commits) {
		precedingCommit = commits[blameLength]
	}

	return &milopb.QueryBlamelistResponse{
		Commits:         commits[:blameLength],
		NextPageToken:   nextPageToken,
		PrecedingCommit: precedingCommit,
	}, nil
}

// prepareQueryBlamelistRequest
//  * validates the request params.
//  * extracts start startRev from page token or gittles commit.
func prepareQueryBlamelistRequest(req *milopb.QueryBlamelistRequest) (startRev string, err error) {
	switch {
	case req.PageSize < 0:
		return "", errors.Reason("page_size can not be negative").Err()
	case req.GitilesCommit == nil:
		return "", errors.Reason("gitiles_commit is required").Err()
	case req.GitilesCommit.Host == "":
		return "", errors.Reason("gitiles_commit.host is required").Err()
	case req.GitilesCommit.Project == "":
		return "", errors.Reason("gitiles_commit.project is required").Err()
	case req.GitilesCommit.Id == "" && req.GitilesCommit.Ref == "":
		return "", errors.Reason("either gitiles_commit.id or gitiles_commit.ref needs to be specified").Err()
	}

	if req.PageToken != "" {
		token, err := parseQueryBlamelistPageToken(req.PageToken)
		if err != nil {
			return "", errors.Annotate(err, "unable to parse page_token").Err()
		}
		return token.NextCommitId, nil
	}

	if req.GitilesCommit.Id == "" {
		return req.GitilesCommit.Ref, nil
	}

	return req.GitilesCommit.Id, nil
}

func parseQueryBlamelistPageToken(tokenStr string) (token *milopb.QueryBlamelistPageToken, err error) {
	bytes, err := base64.StdEncoding.DecodeString(tokenStr)
	if err != nil {
		return nil, err
	}
	token = &milopb.QueryBlamelistPageToken{}
	err = proto.Unmarshal(bytes, token)
	return
}

func serializeQueryBlamelistPageToken(token *milopb.QueryBlamelistPageToken) (string, error) {
	bytes, err := proto.Marshal(token)
	return base64.StdEncoding.EncodeToString(bytes), err
}

const (
	pageSizeMax     = 1000
	pageSizeDefault = 100
)

// adjustPageSize takes the given requested pageSize and adjusts as necessary.
func adjustPageSize(pageSize int32) int {
	switch {
	case pageSize >= pageSizeMax:
		return pageSizeMax
	case pageSize > 0:
		return int(pageSize)
	default:
		return pageSizeDefault
	}
}
