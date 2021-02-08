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

package changelist

import (
	"testing"

	"go.chromium.org/luci/auth/identity"

	"go.chromium.org/luci/cv/internal/gerrit/gerritfake"

	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"
)

func TestOwnerIdentity(t *testing.T) {
	t.Parallel()

	Convey("Snapshot.OwnerIdentity works", t, func() {
		s := &Snapshot{}
		_, err := s.OwnerIdentity()
		So(err, ShouldErrLike, "non-Gerrit CL")

		ci := gerritfake.CI(101, gerritfake.Owner("owner-1"))
		s.Kind = &Snapshot_Gerrit{Gerrit: &Gerrit{
			Host: "x-review.example.com",
			Info: ci,
		}}
		i, err := s.OwnerIdentity()
		So(err, ShouldBeNil)
		So(i, ShouldEqual, identity.Identity("user:owner-1@example.com"))

		Convey("no preferred email set", func() {
			// Yes, this happens if no preferred email is set. See crbug/1175771.
			ci.Owner.Email = ""
			_, err = s.OwnerIdentity()
			So(err, ShouldErrLike, "CL x-review.example.com/101 owner email of account 1 is unknown")
		})
	})
}
