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

package router

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestRouter(t *testing.T) {
	t.Parallel()

	outputKey := "output"
	client := &http.Client{}
	appendValue := func(c context.Context, key string, val string) context.Context {
		var current []string
		if v := c.Value(key); v != nil {
			current = v.([]string)
		}
		return context.WithValue(c, key, append(current, val))
	}
	a := func(c *Context, next Handler) {
		c.Request = c.Request.WithContext(appendValue(c.Request.Context(), outputKey, "a:before"))
		next(c)
		c.Request = c.Request.WithContext(appendValue(c.Request.Context(), outputKey, "a:after"))
	}
	b := func(c *Context, next Handler) {
		c.Request = c.Request.WithContext(appendValue(c.Request.Context(), outputKey, "b:before"))
		next(c)
		c.Request = c.Request.WithContext(appendValue(c.Request.Context(), outputKey, "b:after"))
	}
	c := func(c *Context, next Handler) {
		c.Request = c.Request.WithContext(appendValue(c.Request.Context(), outputKey, "c"))
		next(c)
	}
	d := func(c *Context, next Handler) {
		next(c)
		c.Request = c.Request.WithContext(appendValue(c.Request.Context(), outputKey, "d"))
	}
	stop := func(_ *Context, _ Handler) {}
	handler := func(c *Context) {
		c.Request = c.Request.WithContext(appendValue(c.Request.Context(), outputKey, "handler"))
	}

	Convey("Router", t, func() {
		r := New()

		Convey("New", func() {
			Convey("Should initialize non-nil httprouter.Router", func() {
				So(r.hrouter, ShouldNotBeNil)
			})
		})

		Convey("Use", func() {
			Convey("Should append middleware", func() {
				So(len(r.middleware), ShouldEqual, 0)
				r.Use(NewMiddlewareChain(a, b))
				So(len(r.middleware), ShouldEqual, 2)
			})
		})

		Convey("Subrouter", func() {
			Convey("Should create new router with values from original router", func() {
				r.BasePath = "/foo"
				r.Use(NewMiddlewareChain(a, b))
				r2 := r.Subrouter("bar")
				So(r.hrouter, ShouldPointTo, r2.hrouter)
				So(r2.BasePath, ShouldEqual, "/foo/bar")
				So(r.middleware, ShouldResemble, r2.middleware)
			})
		})

		Convey("Handle", func() {
			Convey("Should not modify existing empty r.middleware slice", func() {
				So(len(r.middleware), ShouldEqual, 0)
				r.Handle("GET", "/bar", NewMiddlewareChain(b, c), handler)
				So(len(r.middleware), ShouldEqual, 0)
			})

			Convey("Should not modify existing r.middleware slice", func() {
				r.Use(NewMiddlewareChain(a))
				So(len(r.middleware), ShouldEqual, 1)
				r.Handle("GET", "/bar", NewMiddlewareChain(b, c), handler)
				So(len(r.middleware), ShouldEqual, 1)
			})
		})

		Convey("run", func() {
			ctx := &Context{
				Request: &http.Request{},
			}

			Convey("Should execute handler when using nil middlewares", func() {
				run(ctx, nil, nil, handler)
				So(ctx.Request.Context().Value(outputKey), ShouldResemble, []string{"handler"})
			})

			Convey("Should execute middlewares and handler in order", func() {
				m := NewMiddlewareChain(a, b, c)
				n := NewMiddlewareChain(d)
				run(ctx, m, n, handler)
				So(ctx.Request.Context().Value(outputKey), ShouldResemble,
					[]string{"a:before", "b:before", "c", "handler", "d", "b:after", "a:after"},
				)
			})

			Convey("Should not execute upcoming middleware/handlers if next is not called", func() {
				mc := NewMiddlewareChain(a, stop, b)
				run(ctx, mc, NewMiddlewareChain(), handler)
				So(ctx.Request.Context().Value(outputKey), ShouldResemble, []string{"a:before", "a:after"})
			})

			Convey("Should execute next middleware when it encounters nil middleware", func() {
				Convey("At start of first chain", func() {
					run(ctx, NewMiddlewareChain(nil, a), NewMiddlewareChain(b), handler)
					So(ctx.Request.Context().Value(outputKey), ShouldResemble, []string{"a:before", "b:before", "handler", "b:after", "a:after"})
				})
				Convey("At start of second chain", func() {
					run(ctx, NewMiddlewareChain(a), NewMiddlewareChain(nil, b), handler)
					So(ctx.Request.Context().Value(outputKey), ShouldResemble, []string{"a:before", "b:before", "handler", "b:after", "a:after"})
				})
				Convey("At end of first chain", func() {
					run(ctx, NewMiddlewareChain(a, nil), NewMiddlewareChain(b), handler)
					So(ctx.Request.Context().Value(outputKey), ShouldResemble, []string{"a:before", "b:before", "handler", "b:after", "a:after"})
				})
				Convey("At end of second chain", func() {
					run(ctx, NewMiddlewareChain(a), NewMiddlewareChain(b, nil), handler)
					So(ctx.Request.Context().Value(outputKey), ShouldResemble, []string{"a:before", "b:before", "handler", "b:after", "a:after"})
				})
			})
		})

		Convey("ServeHTTP", func() {
			ts := httptest.NewServer(r)
			a := func(c *Context, next Handler) {
				c.Request = c.Request.WithContext(appendValue(c.Request.Context(), outputKey, "a:before"))
				next(c)
				c.Request = c.Request.WithContext(appendValue(c.Request.Context(), outputKey, "a:after"))
				io.WriteString(c.Writer, strings.Join(c.Request.Context().Value(outputKey).([]string), ","))
			}

			Convey("Should execute middleware registered in Use and Handle in order", func() {
				r.Use(NewMiddlewareChain(a))
				r.GET("/ab", NewMiddlewareChain(b), handler)
				res, err := client.Get(ts.URL + "/ab")
				So(err, ShouldBeNil)
				defer res.Body.Close()
				So(res.StatusCode, ShouldEqual, http.StatusOK)
				p, err := io.ReadAll(res.Body)
				So(err, ShouldBeNil)
				So(string(p), ShouldEqual, strings.Join(
					[]string{"a:before", "b:before", "handler", "b:after", "a:after"},
					",",
				))
			})

			Convey("Should return method not allowed for existing path but wrong method in request", func() {
				r.POST("/comment", NewMiddlewareChain(), handler)
				res, err := client.Get(ts.URL + "/comment")
				So(err, ShouldBeNil)
				defer res.Body.Close()
				So(res.StatusCode, ShouldEqual, http.StatusMethodNotAllowed)
			})

			Convey("Should return expected response from handler", func() {
				handler := func(c *Context) {
					c.Writer.Write([]byte("Hello, " + c.Params[0].Value))
				}
				r.GET("/hello/:name", NewMiddlewareChain(c, d), handler)
				res, err := client.Get(ts.URL + "/hello/世界")
				So(err, ShouldBeNil)
				defer res.Body.Close()
				So(res.StatusCode, ShouldEqual, http.StatusOK)
				p, err := io.ReadAll(res.Body)
				So(err, ShouldBeNil)
				So(string(p), ShouldEqual, "Hello, 世界")
			})

			Reset(func() {
				ts.Close()
			})
		})

		Convey("makeBasePath", func() {
			cases := []struct{ base, relative, result string }{
				{"/", "", "/"},
				{"", "", "/"},
				{"foo", "", "/foo"},
				{"foo/", "", "/foo/"},
				{"foo", "/", "/foo/"},
				{"foo", "bar", "/foo/bar"},
				{"foo/", "bar", "/foo/bar"},
				{"foo", "bar/", "/foo/bar/"},
				{"/foo", "/bar", "/foo/bar"},
				{"/foo/", "/bar", "/foo/bar"},
				{"foo//", "///bar", "/foo/bar"},
				{"foo", "bar///baz/qux", "/foo/bar/baz/qux"},
				{"//foo//", "///bar///baz/qux/", "/foo/bar/baz/qux/"},
			}
			for _, c := range cases {
				So(makeBasePath(c.base, c.relative), ShouldEqual, c.result)
			}
		})
	})
}
