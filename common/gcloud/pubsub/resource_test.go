// Copyright 2015 The LUCI Authors.
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

package pubsub

import (
	"fmt"
	"strings"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestResource(t *testing.T) {
	Convey(`Resource validation`, t, func() {
		for _, tc := range []struct {
			v     string
			c     string
			valid bool
		}{
			{"projects/foo/topics/googResource", "topics", false},
			{"projects/foo/topics/yo", "topics", false},
			{fmt.Sprintf("projects/foo/topics/%s", strings.Repeat("a", 256)), "topics", false},
			{"projects/foo/topics/1ring", "topics", false},
			{"projects/foo/topics/allAboutThe$", "topics", false},
			{"projects/foo/topics/アップ", "topics", false},

			{"projects/foo/topics/AAA", "topics", true},
			{"projects/foo/topics/testResource", "topics", true},
			{fmt.Sprintf("projects/foo/topics/%s", strings.Repeat("a", 255)), "topics", true},
			{"projects/foo/topics/myResource-a_b.c~d+e%%f", "topics", true},
		} {
			if tc.valid {
				Convey(fmt.Sprintf(`A valid resource [%s] will validate.`, tc.v), func() {
					So(validateResource(tc.v, tc.c), ShouldBeNil)
				})
			} else {
				Convey(fmt.Sprintf(`An invalid resource [%s] will not validate.`, tc.v), func() {
					So(validateResource(tc.v, tc.c), ShouldNotBeNil)
				})
			}
		}
	})

	Convey(`Can create a new, valid Topic.`, t, func() {
		t := NewTopic("foo", "testTopic")
		So(t, ShouldEqual, Topic("projects/foo/topics/testTopic"))
		So(t.Validate(), ShouldBeNil)
	})

	Convey(`Can create a new, valid Subscription.`, t, func() {
		t := NewSubscription("foo", "testSubscription")
		So(t, ShouldEqual, Subscription("projects/foo/subscriptions/testSubscription"))
		So(t.Validate(), ShouldBeNil)
	})

	Convey(`Can extract a resource project.`, t, func() {
		p, n, err := resourceProjectName("projects/foo/topics/bar")
		So(err, ShouldBeNil)
		So(p, ShouldEqual, "foo")
		So(n, ShouldEqual, "bar")
	})
}
