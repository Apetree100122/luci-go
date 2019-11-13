// Copyright 2019 The LUCI Authors.
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

package pagination

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestCursor(t *testing.T) {
	t.Parallel()

	Convey(`Token works`, t, func() {
		So(Token("v1", "v2"), ShouldResemble, "CgJ2MQoCdjI=")

		pos, err := ParseToken("CgJ2MQoCdjI=")
		So(err, ShouldBeNil)
		So(pos, ShouldResemble, []string{"v1", "v2"})

		Convey(`For fresh cursor`, func() {
			So(Token(), ShouldResemble, "")

			pos, err := ParseToken("")
			So(err, ShouldBeNil)
			So(pos, ShouldBeNil)
		})
	})
}

func TestAdjustPageSize(t *testing.T) {
	t.Parallel()
	Convey(`AdjustPageSize`, t, func() {
		Convey(`OK`, func() {
			So(AdjustPageSize(50), ShouldEqual, 50)
		})
		Convey(`Too big`, func() {
			So(AdjustPageSize(1e6), ShouldEqual, pageSizeMax)
		})
		Convey(`Missing or 0`, func() {
			So(AdjustPageSize(0), ShouldEqual, pageSizeDefault)
		})
	})
}
