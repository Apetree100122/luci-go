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

package environ

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func newImpl(caseInsensitive bool, entries []string) Env {
	e := Env{CaseInsensitive: caseInsensitive}
	e.LoadSlice(entries)
	return e
}

func TestEnvironmentConversion(t *testing.T) {
	t.Parallel()

	Convey(`Source environment slice translates correctly to/from an Env.`, t, func() {
		Convey(`Case insensitive (e.g., Windows)`, func() {
			env := newImpl(true, []string{
				"",
				"FOO",
				"BAR=BAZ",
				"bar=baz",
				"qux=quux=quuuuuuux",
			})
			So(env, ShouldResemble, Env{
				CaseInsensitive: true,
				env: map[string]string{
					"FOO": "FOO=",
					"BAR": "bar=baz",
					"QUX": "qux=quux=quuuuuuux",
				},
			})

			So(env.Sorted(), ShouldResemble, []string{
				"FOO=",
				"bar=baz",
				"qux=quux=quuuuuuux",
			})

			So(env.GetEmpty(""), ShouldEqual, "")
			So(env.GetEmpty("FOO"), ShouldEqual, "")
			So(env.GetEmpty("BAR"), ShouldEqual, "baz")
			So(env.GetEmpty("bar"), ShouldEqual, "baz")
			So(env.GetEmpty("qux"), ShouldEqual, "quux=quuuuuuux")
			So(env.GetEmpty("QuX"), ShouldEqual, "quux=quuuuuuux")
		})

		Convey(`Case sensitive (e.g., POSIX)`, func() {
			env := newImpl(false, []string{
				"",
				"FOO",
				"BAR=BAZ",
				"bar=baz",
				"qux=quux=quuuuuuux",
			})
			So(env, ShouldResemble, Env{
				env: map[string]string{
					"FOO": "FOO=",
					"BAR": "BAR=BAZ",
					"bar": "bar=baz",
					"qux": "qux=quux=quuuuuuux",
				},
			})

			So(env.Sorted(), ShouldResemble, []string{
				"BAR=BAZ",
				"FOO=",
				"bar=baz",
				"qux=quux=quuuuuuux",
			})

			So(env.GetEmpty(""), ShouldEqual, "")
			So(env.GetEmpty("FOO"), ShouldEqual, "")
			So(env.GetEmpty("BAR"), ShouldEqual, "BAZ")
			So(env.GetEmpty("bar"), ShouldEqual, "baz")
			So(env.GetEmpty("qux"), ShouldEqual, "quux=quuuuuuux")
			So(env.GetEmpty("QuX"), ShouldEqual, "")
		})
	})
}

func TestEnvironmentManipulation(t *testing.T) {
	t.Parallel()

	Convey(`A zero-valued Env`, t, func() {
		var env Env
		So(env.Len(), ShouldEqual, 0)

		Convey(`Can be sorted.`, func() {
			So(env.Sorted(), ShouldEqual, nil)
		})

		Convey(`Can call Get`, func() {
			v, ok := env.Get("foo")
			So(ok, ShouldBeFalse)
			So(v, ShouldEqual, "")
		})

		Convey(`Can call Set`, func() {
			env.Set("foo", "bar")
			So(env.Len(), ShouldEqual, 1)
			So(env.Sorted(), ShouldResemble, []string{"foo=bar"})
		})

		Convey(`Can be cloned, and propagates case insensitivity.`, func() {
			env.CaseInsensitive = true
			So(env.Clone(), ShouldResemble, env)
		})
	})

	Convey(`A testing Env`, t, func() {
		env := newImpl(true, []string{
			"PYTHONPATH=/foo:/bar:/baz",
			"http_proxy=wiped-out-by-next",
			"http_proxy=http://example.com",
			"novalue",
		})
		So(env.Len(), ShouldEqual, 3)

		Convey(`Can Get values.`, func() {
			v, ok := env.Get("PYTHONPATH")
			So(ok, ShouldBeTrue)
			So(v, ShouldEqual, "/foo:/bar:/baz")

			v, ok = env.Get("http_proxy")
			So(ok, ShouldBeTrue)
			So(v, ShouldEqual, "http://example.com")

			v, ok = env.Get("novalue")
			So(ok, ShouldBeTrue)
			So(v, ShouldEqual, "")
		})

		Convey(`Will note missing values.`, func() {
			_, ok := env.Get("missing")
			So(ok, ShouldBeFalse)

			_, ok = env.Get("")
			So(ok, ShouldBeFalse)
		})

		Convey(`Can be converted into a map and enumerated`, func() {
			So(env.Map(), ShouldResemble, map[string]string{
				"PYTHONPATH": "/foo:/bar:/baz",
				"http_proxy": "http://example.com",
				"novalue":    "",
			})

			Convey(`Can perform iteration`, func() {
				buildMap := make(map[string]string)
				So(env.Iter(func(k, v string) error {
					buildMap[k] = v
					return nil
				}), ShouldBeNil)
				So(env.Map(), ShouldResemble, buildMap)
			})

			Convey(`Can have elements removed through iteration`, func() {
				env.RemoveMatch(func(k, v string) bool {
					switch k {
					case "PYTHONPATH", "novalue":
						return true
					default:
						return false
					}
				})
				So(env.Map(), ShouldResemble, map[string]string{
					"http_proxy": "http://example.com",
				})
			})
		})

		Convey(`Can update its values.`, func() {
			orig := env.Clone()

			// Update PYTHONPATH, confirm that it updated correctly.
			v, _ := env.Get("PYTHONPATH")
			env.Set("PYTHONPATH", "/override:"+v)
			So(env.Sorted(), ShouldResemble, []string{
				"PYTHONPATH=/override:/foo:/bar:/baz",
				"http_proxy=http://example.com",
				"novalue=",
			})

			// Use a different-case key, and confirm that it still updated correctly.
			Convey(`When case insensitive, will update common keys.`, func() {
				env.CaseInsensitive = true

				env.Set("pYtHoNpAtH", "/override:"+v)
				So(env.Sorted(), ShouldResemble, []string{
					"http_proxy=http://example.com",
					"novalue=",
					"pYtHoNpAtH=/override:/foo:/bar:/baz",
				})
				So(env.GetEmpty("PYTHONPATH"), ShouldEqual, "/override:/foo:/bar:/baz")

				So(env.Remove("HTTP_PROXY"), ShouldBeTrue)
				So(env.Remove("nonexistent"), ShouldBeFalse)
				So(env.Sorted(), ShouldResemble, []string{
					"novalue=",
					"pYtHoNpAtH=/override:/foo:/bar:/baz",
				})

				// Test that the clone didn't change.
				So(orig.Sorted(), ShouldResemble, []string{
					"PYTHONPATH=/foo:/bar:/baz",
					"http_proxy=http://example.com",
					"novalue=",
				})

				var mergeMe Env
				mergeMe.LoadSlice([]string{
					"http_PROXY=foo",
					"HTTP_PROXY=FOO",
					"newkey=value",
				})
				orig.Update(mergeMe)
				So(orig.Sorted(), ShouldResemble, []string{
					"PYTHONPATH=/foo:/bar:/baz",
					"http_PROXY=foo",
					"newkey=value",
					"novalue=",
				})
			})
		})
	})
}

func TestEnvironmentConstruction(t *testing.T) {
	t.Parallel()

	Convey(`Testing Load* with an empty environment`, t, func() {
		var env Env

		Convey(`Can load an initial set of values from a map`, func() {
			env.Load(map[string]string{
				"FOO": "BAR",
				"foo": "bar",
			})

			So(env, ShouldResemble, Env{
				env: map[string]string{
					"FOO": "FOO=BAR",
					"foo": "foo=bar",
				},
			})
		})

		Convey(`Load with an nil/empty environment returns a zero value.`, func() {
			env.Load(nil)
			So(env.env, ShouldBeNil)

			env.Load(map[string]string{})
			So(env.env, ShouldBeNil)
		})
	})

	Convey(`New with a nil/empty slice returns a zero value.`, t, func() {
		So(New(nil).env, ShouldBeNil)
		So(New([]string{}).env, ShouldBeNil)
	})
}
