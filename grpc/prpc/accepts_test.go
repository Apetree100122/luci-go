// Copyright 2016 The LUCI Authors.
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

package prpc

import (
	"net/http"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"
)

func TestAccept(t *testing.T) {
	t.Parallel()

	Convey("parseAccept", t, func() {
		test := func(value string, expectedErr interface{}, expectedTypes ...acceptItem) {
			Convey("mediaType="+value, func() {
				actual, err := parseAccept(value)
				So(err, ShouldErrLike, expectedErr)
				So(actual, ShouldResemble, accept(expectedTypes))
			})
		}
		test("", nil)
		qf1 := func(value string) acceptItem {
			return acceptItem{
				Value:         value,
				QualityFactor: 1.0,
			}
		}

		test("text/html", nil,
			qf1("text/html"),
		)

		test("text/html; level=1", nil,
			qf1("text/html; level=1"),
		)
		test("TEXT/HTML; LEVEL=1", nil,
			qf1("TEXT/HTML; LEVEL=1"),
		)

		test("text/html, application/json", nil,
			qf1("text/html"),
			qf1("application/json"),
		)

		test("text/html; level=1, application/json", nil,
			qf1("text/html; level=1"),
			qf1("application/json"),
		)

		test("text/html; level=1, application/json; foo=bar", nil,
			qf1("text/html; level=1"),
			qf1("application/json; foo=bar"),
		)

		test("text/html; level=1; q=0.5", nil,
			acceptItem{
				Value:         "text/html; level=1",
				QualityFactor: 0.5,
			},
		)
		test("text/html; level=1; q=0.5; a=1", nil,
			acceptItem{
				Value:         "text/html; level=1",
				QualityFactor: 0.5,
			},
		)
		test("TEXT/HTML; LEVEL=1; Q=0.5", nil,
			acceptItem{
				Value:         "TEXT/HTML; LEVEL=1",
				QualityFactor: 0.5,
			},
		)
		test("text/html; level=1; q=0.5, application/json", nil,
			acceptItem{
				Value:         "text/html; level=1",
				QualityFactor: 0.5,
			},
			qf1("application/json"),
		)
		test("text/html; level=1; q=0.5, application/json; a=b", nil,
			acceptItem{
				Value:         "text/html; level=1",
				QualityFactor: 0.5,
			},
			qf1("application/json; a=b"),
		)
		test("text/html; level=1; q=0.5, */*; q=0.1", nil,
			acceptItem{
				Value:         "text/html; level=1",
				QualityFactor: 0.5,
			},
			acceptItem{
				Value:         "*/*",
				QualityFactor: 0.1,
			},
		)
		test("text/html; level=1; q=0.5, */*; x=3;q=0.1; y=5", nil,
			acceptItem{
				Value:         "text/html; level=1",
				QualityFactor: 0.5,
			},
			acceptItem{
				Value:         "*/*; x=3",
				QualityFactor: 0.1,
			},
		)
		test("text/html;q=q", "q parameter: expected a floating-point number")
	})

	Convey("qParamSplit", t, func() {
		type params map[string]string

		test := func(item, mediaType, qValue string) {
			Convey(item, func() {
				actualMediaType, actualQValue := qParamSplit(item)
				So(actualMediaType, ShouldEqual, mediaType)
				So(actualQValue, ShouldEqual, qValue)
			})
		}

		test("", "", "")
		test("a/b", "a/b", "")

		test("a/b;mtp1=1", "a/b;mtp1=1", "")
		test("a/b;q=1", "a/b", "1")
		test("a/b; q=1", "a/b", "1")
		test("a/b;mtp1=1;q=1", "a/b;mtp1=1", "1")
		test("a/b;mtp1=1;q=1;", "a/b;mtp1=1", "1")
		test("a/b;mtp1=1; q=1", "a/b;mtp1=1", "1")
		test("a/b;mtp1=1;q =1", "a/b;mtp1=1", "1")
		test("a/b;mtp1=1;q  =1", "a/b;mtp1=1", "1")
		test("a/b;mtp1=1;q= 1", "a/b;mtp1=1", "1")
		test("a/b;mtp1=1;q=1 ", "a/b;mtp1=1", "1")

		test("a/b;mtp1=1;Q=1", "a/b;mtp1=1", "1")

		test("a/b;mtp1=1;q=1.0", "a/b;mtp1=1", "1.0")
		test("a/b;mtp1=1;q=1.0;", "a/b;mtp1=1", "1.0")

		test("a/b;mtp1=1;  q  =  foo", "a/b;mtp1=1", "foo")

		test("a/b;c=q", "a/b;c=q", "")
		test("a/b;mtp1=1;  q", "a/b;mtp1=1;  q", "")
		test("a/b;mtp1=1;  q=", "a/b;mtp1=1;  q=", "")
	})
}

func TestAcceptContentEncoding(t *testing.T) {
	t.Parallel()
	Convey("Accept-Encoding", t, func() {
		h := http.Header{}
		Convey(`Empty`, func() {
			So(mayGZipResponse(h), ShouldBeFalse)
		})

		Convey(`gzip`, func() {
			h.Set("Accept-Encoding", "gzip")
			So(mayGZipResponse(h), ShouldBeTrue)
		})

		Convey(`multiple values`, func() {
			h.Add("Accept-Encoding", "foo")
			h.Add("Accept-Encoding", "gzip")
			So(mayGZipResponse(h), ShouldBeTrue)
		})
	})
}
