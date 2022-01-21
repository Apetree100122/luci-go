// Copyright 2021 The LUCI Authors.
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

package userhtml

import (
	"context"
	"fmt"
	"strings"
	"time"

	"google.golang.org/grpc/codes"

	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/grpc/appstatus"
	"go.chromium.org/luci/server/caching"
	"go.chromium.org/luci/server/caching/layered"
	"go.chromium.org/luci/server/router"
	"go.chromium.org/luci/server/templates"

	"go.chromium.org/luci/cv/internal/acls"
	"go.chromium.org/luci/cv/internal/changelist"
	"go.chromium.org/luci/cv/internal/common"
	"go.chromium.org/luci/cv/internal/rpc/pagination"
	"go.chromium.org/luci/cv/internal/run"
)

func recentsPage(c *router.Context) {
	project := c.Params.ByName("Project")
	params, err := parseFormParams(c)
	if err != nil {
		errPage(c, err)
		return
	}

	runs, prev, next, err := searchRuns(c.Context, project, params)
	if err != nil {
		errPage(c, err)
		return
	}
	runsWithCLs, err := resolveRunsCLs(c.Context, runs)
	if err != nil {
		errPage(c, err)
		return
	}

	templates.MustRender(c.Context, c.Writer, "pages/recent_runs.html", map[string]interface{}{
		"Runs":         runsWithCLs,
		"Project":      project,
		"PrevPage":     prev,
		"NextPage":     next,
		"FilterStatus": params.statusString(),
		"FilterMode":   params.modeString(),
		"Now":          startTime(c.Context),
	})
}

type recentRunsParams struct {
	status          run.Status
	mode            run.Mode
	pageTokenString string
}

func parseFormParams(c *router.Context) (recentRunsParams, error) {
	params := recentRunsParams{}
	if err := c.Request.ParseForm(); err != nil {
		return params, errors.Annotate(err, "failed to parse form").Err()
	}

	s := c.Request.Form.Get("status")
	switch val, ok := run.Status_value[strings.ToUpper(s)]; {
	case s == "":
	case ok:
		params.status = run.Status(val)
	default:
		return params, fmt.Errorf("invalid Run status %q", s)
	}

	switch m := run.Mode(c.Request.Form.Get("mode")); {
	case m == "":
	case m.Valid():
		params.mode = m
	default:
		return params, fmt.Errorf("invalid Run mode %q", params.mode)
	}

	params.pageTokenString = strings.TrimSpace(c.Request.Form.Get("page"))
	return params, nil
}

func (r *recentRunsParams) statusString() string {
	if r.status == run.Status_STATUS_UNSPECIFIED {
		return ""
	}
	return r.status.String()
}

func (r *recentRunsParams) modeString() string {
	return string(r.mode)
}

func searchRuns(ctx context.Context, project string, params recentRunsParams) (runs []*run.Run, prev, next string, err error) {
	var pageToken *run.PageToken
	if params.pageTokenString != "" {
		pageToken = &run.PageToken{}
		if err = pagination.DecryptPageToken(ctx, params.pageTokenString, pageToken); err != nil {
			// Log but don't return to the user entire error to avoid any accidental
			// leakage.
			logging.Warningf(ctx, "bad page token: %s", err)
			err = fmt.Errorf("bad page token")
			return
		}
	}

	var qb interface {
		LoadRuns(context.Context, ...run.LoadRunChecker) ([]*run.Run, *run.PageToken, error)
	}
	if project == "" {
		qb = run.RecentQueryBuilder{
			Limit:              50,
			CheckProjectAccess: acls.CheckProjectAccess,
			Status:             params.status,
		}.PageToken(pageToken)
	} else {
		switch ok, err := acls.CheckProjectAccess(ctx, project); {
		case err != nil:
			return nil, "", "", err
		case !ok:
			// Return NotFound error in the case of access denied.
			//
			// Rationale: the caller shouldn't be able to distinguish between
			// project not existing and not having access to the project, because
			// it may leak the existence of the project.
			return nil, "", "", appstatus.Errorf(codes.NotFound, "Project %q not found", project)
		}
		qb = run.ProjectQueryBuilder{
			Project: project,
			Limit:   50,
			Status:  params.status,
		}.PageToken(pageToken)
	}

	var nextPageToken *run.PageToken
	runs, nextPageToken, err = qb.LoadRuns(ctx, acls.NewRunReadChecker())
	if err != nil {
		return
	}
	logging.Debugf(ctx, "%d runs retrieved", len(runs))

	next, err = pagination.EncryptPageToken(ctx, nextPageToken)
	if err != nil {
		return
	}

	prev, err = pageTokens(ctx, params.pageTokenString, next)
	return
}

func resolveRunsCLs(ctx context.Context, runs []*run.Run) ([]runWithExternalCLs, error) {
	cls := make(map[common.CLID]*changelist.CL, len(runs))
	for _, r := range runs {
		for _, clid := range r.CLs {
			if cls[clid] == nil {
				cls[clid] = &changelist.CL{ID: clid}
			}
		}
	}
	if _, err := changelist.LoadCLsMap(ctx, cls); err != nil {
		return nil, err
	}

	out := make([]runWithExternalCLs, len(runs))
	for i, r := range runs {
		out[i].Run = r
		out[i].ExternalCLs = make([]changelist.ExternalID, len(r.CLs))
		for j, clid := range r.CLs {
			out[i].ExternalCLs[j] = cls[clid].ExternalID
		}
	}
	return out, nil
}

type runWithExternalCLs struct {
	*run.Run
	ExternalCLs []changelist.ExternalID
}

var tokenCache = layered.Cache{
	ProcessLRUCache: caching.RegisterLRUCache(1024),
	GlobalNamespace: "recent_cv_runs_page_token_cache",
	Marshal: func(item interface{}) ([]byte, error) {
		return []byte(item.(string)), nil
	},
	Unmarshal: func(blob []byte) (interface{}, error) {
		return string(blob), nil
	},
}

var tokenExp = 24 * time.Hour

// pageTokens caches the current pageToken associated to the next,
// so as to populate the previous page link when rendering the next page.
// Also returns a previously saved page token pointing to the previous page.
func pageTokens(ctx context.Context, pageToken, nextPageToken string) (prev string, err error) {

	// A whitespace will cause a 'Previous' link to render with no page token.
	// i.e. going to the first page.
	// An empty string will not render any 'Previous' link.

	// TODO(crbug.com/1249253): Consider redirecting to the first page of the
	// query when the given page token is valid, but we can't retrieve its
	// previous page.
	blankToken := " "
	var cachedV interface{}

	if pageToken == "" {
		pageToken = blankToken
	} else {
		cachedV, err = tokenCache.GetOrCreate(ctx, pageToken, func() (v interface{}, exp time.Duration, err error) {
			// We haven't seen this token yet, we don't know what its previous page is.
			return "", 0, nil
		})
		if err != nil {
			return
		}
		prev = cachedV.(string)
	}
	_, err = tokenCache.GetOrCreate(ctx, nextPageToken, func() (v interface{}, exp time.Duration, err error) {
		return pageToken, tokenExp, nil
	})
	return
}
