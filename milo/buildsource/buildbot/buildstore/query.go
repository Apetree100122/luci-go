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

package buildstore

import (
	"context"
	"sort"
	"strconv"

	"go.chromium.org/gae/service/datastore"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"

	"go.chromium.org/luci/milo/buildsource/buildbot/buildbotapi"
)

// Ternary has 3 defined values: either (zero), yes and no.
type Ternary int

const (
	Either Ternary = iota
	Yes
	No
)

func (t Ternary) filter(q *datastore.Query, fieldName string) *datastore.Query {
	switch t {
	case Yes:
		return q.Eq(fieldName, true)
	case No:
		return q.Eq(fieldName, false)
	default:
		return q
	}
}

// Query is a build query.
type Query struct {
	Master   string
	Builder  string
	Limit    int
	Finished Ternary
	Cursor   string

	// The following fields are tuning parameters specific to a buildstore
	// implementation. Their usage implies understanding of how emulation
	// works.

	// KeyOnly, if true, makes the datastore query keys-only.
	// Loaded Buildbot builds will have only master, builder and number.
	KeyOnly bool // make the data

	// NoAnnotationFetch, if true, will not fetch annotation proto from LogDog.
	// Loaded LUCI builds will not have properties, steps, logs or text.
	NoAnnotationFetch bool

	// NoChangeFetch, if true, will not load change history from Gitiles.
	// Loaded LUCI builds will not have Blame or SourceStamp.Changes.
	NoChangeFetch bool
}

func (q *Query) dsQuery() *datastore.Query {
	dsq := datastore.NewQuery(buildKind)
	if q.Master != "" {
		dsq = dsq.Eq("master", q.Master)
	}
	if q.Builder != "" {
		dsq = dsq.Eq("builder", q.Builder)
	}
	dsq = q.Finished.filter(dsq, "finished")
	if q.Limit > 0 {
		dsq = dsq.Limit(int32(q.Limit))
	}
	if q.KeyOnly {
		dsq = dsq.KeysOnly(true)
	}
	return dsq
}

// QueryResult is a result of running a Query.
type QueryResult struct {
	Builds     []*buildbotapi.Build // ordered from greater-number to lower-number
	NextCursor string
	PrevCursor string
}

// GetBuilds executes a build query and returns results.
// Does not check access.
func GetBuilds(c context.Context, q Query) (*QueryResult, error) {
	switch {
	case q.Master == "":
		return nil, errors.New("master is required")
	case q.Builder == "":
		return nil, errors.New("builder is required")
	}
	return getDatastoreBuilds(c, q, true)
}

// mergeBuilds merges builds from a and b to one slice.
// The returned builds are ordered by build numbers, descending.
//
// If a build number is present in both a and b, b's build is ignored.
func mergeBuilds(a, b []*buildbotapi.Build) []*buildbotapi.Build {
	ret := make([]*buildbotapi.Build, len(a), len(a)+len(b))
	copy(ret, a)

	// add builds from b that have unique build numbers.
	aNumbers := make(map[int]struct{}, len(a))
	for _, build := range a {
		aNumbers[build.Number] = struct{}{}
	}
	for _, build := range b {
		if _, ok := aNumbers[build.Number]; !ok {
			ret = append(ret, build)
		}
	}
	sort.Slice(ret, func(i, j int) bool {
		return ret[i].Number > ret[j].Number
	})
	return ret
}

func getDatastoreBuilds(c context.Context, q Query, includeExperimental bool) (*QueryResult, error) {
	var builds []*buildEntity
	if q.Limit > 0 {
		builds = make([]*buildEntity, 0, q.Limit)
	}

	dsq := q.dsQuery()

	if !includeExperimental {
		dsq = dsq.Eq("is_experimental", false)
	}

	// CUSTOM CURSOR.
	// This function uses a custom cursor based on build numbers.
	// A cursor is a build number that defines a page boundary.
	// If >=0, it is the inclusive lower boundary.
	//   Example: cursor="10", means return builds ...12, 11, 10.
	// If <0, it is the exclusive upper boundary, negated.
	//   Example: -10, means return builds 9, 8, 7...
	cursorNumber := 0
	order := "-number"
	reverse := false
	hasCursor := false
	if q.Cursor != "" {
		var err error
		if cursorNumber, err = strconv.Atoi(q.Cursor); err == nil {
			hasCursor = true
			if cursorNumber >= 0 {
				dsq = dsq.Gte("number", cursorNumber)
				order = "number"
				reverse = true
			} else {
				dsq = dsq.Lt("number", -cursorNumber)
			}
		} else {
			return nil, errors.New("bad cursor")
		}
	}
	dsq = dsq.Order(order)

	logging.Debugf(c, "running datastore query: %s", dsq)
	err := datastore.GetAll(c, dsq, &builds)
	if err != nil {
		return nil, err
	}
	if reverse {
		for i, j := 0, len(builds)-1; i < j; i, j = i+1, j-1 {
			builds[i], builds[j] = builds[j], builds[i]
		}
	}
	res := &QueryResult{
		Builds: make([]*buildbotapi.Build, len(builds)),
	}
	for i, b := range builds {
		b.addViewPath()
		res.Builds[i] = (*buildbotapi.Build)(b)
	}

	// Compute prev and next cursors.
	switch {
	case len(res.Builds) > 0:
		// res.Builds are ordered by numbers descending.

		// previous page must display builds with higher numbers.
		if !hasCursor {
			// do not generate a prev cursor for a non-cursor query
		} else {
			// positive cursors are inclusive
			res.PrevCursor = strconv.Itoa(res.Builds[0].Number + 1)
		}

		// next page must display builds with lower numbers.

		if lastNum := res.Builds[len(res.Builds)-1].Number; lastNum == 0 {
			// this is the first ever build, 0, do not generate a cursor
		} else {
			// negative cursors are exclusive.
			res.NextCursor = strconv.Itoa(-lastNum)
		}

	case cursorNumber > 0:
		// no builds and cursor is the inclusive lower boundary
		// e.g. cursor asks for builds after 10,
		// but there are only 0..5 builds.
		// Make the next cursor for builds <10.
		res.NextCursor = strconv.Itoa(-cursorNumber)

	default:
		// there can't be any builds.
	}

	return res, nil
}
