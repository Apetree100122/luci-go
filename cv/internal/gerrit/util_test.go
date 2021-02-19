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

package gerrit

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestTruncateMessage(t *testing.T) {
	t.Parallel()

	Convey("TruncateMessage", t, func() {

		Convey("Noop when message doesn't exceed max length", func() {
			const msg = "this is a message"
			So(truncate(msg, len(msg)), ShouldEqual, msg)
		})

		Convey("Truncate works", func() {
			const msg = "message is 45 characters long; max length 40."
			So(truncate(msg, 40), ShouldEqual, "message \n...[truncated too long message]")
		})

		Convey("Last rune is valid utf-8", func() {
			const msg = "😀😀😀😀😀😀😀😀😀😀😀😀"
			So(msg, ShouldHaveLength, 12*4) // 😀 is 4 bytes
			const expected = "😀\n...[truncated too long message]"
			// result should only keep (39-32)/4 = 1 emoji, where 32 is the
			// length of placeholder.
			So(truncate(msg, 39), ShouldEqual, expected)
		})
	})
}
