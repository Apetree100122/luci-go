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

package grpcutil

import (
	"context"
	"errors"
	"testing"

	"google.golang.org/grpc"

	. "github.com/smartystreets/goconvey/convey"
)

func TestChainUnaryServerInterceptors(t *testing.T) {
	t.Parallel()

	Convey("With interceptors", t, func() {
		testCtxKey := "testing"
		testInfo := &grpc.UnaryServerInfo{} // constant address for assertions
		testResponse := new(int)            // constant address for assertions
		testError := errors.New("boom")     // constant address for assertions

		calls := []string{}
		record := func(fn string) func() {
			calls = append(calls, "-> "+fn)
			return func() { calls = append(calls, "<- "+fn) }
		}

		callChain := func(intr grpc.UnaryServerInterceptor, h grpc.UnaryHandler) (interface{}, error) {
			return intr(context.Background(), "request", testInfo, h)
		}

		// A "library" of interceptors used below.

		doNothing := func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
			defer record("doNothing")()
			return handler(ctx, req)
		}

		populateContext := func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
			defer record("populateContext")()
			return handler(context.WithValue(ctx, &testCtxKey, "value"), req)
		}

		checkContext := func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
			defer record("checkContext")()
			So(ctx.Value(&testCtxKey), ShouldEqual, "value")
			return handler(ctx, req)
		}

		modifyReq := func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
			defer record("modifyReq")()
			return handler(ctx, "modified request")
		}

		checkReq := func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
			defer record("checkReq")()
			So(req.(string), ShouldEqual, "modified request")
			return handler(ctx, req)
		}

		checkErr := func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
			defer record("checkErr")()
			resp, err := handler(ctx, req)
			So(err, ShouldEqual, testError)
			return resp, err
		}

		abortChain := func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
			defer record("abortChain")()
			return nil, testError
		}

		successHandler := func(ctx context.Context, req interface{}) (interface{}, error) {
			defer record("successHandler")()
			return testResponse, nil
		}

		errorHandler := func(ctx context.Context, req interface{}) (interface{}, error) {
			defer record("errorHandler")()
			return nil, testError
		}

		Convey("Noop chain", func() {
			resp, err := callChain(ChainUnaryServerInterceptors(), successHandler)
			So(err, ShouldBeNil)
			So(resp, ShouldEqual, testResponse)
			So(calls, ShouldResemble, []string{
				"-> successHandler",
				"<- successHandler",
			})
		})

		Convey("One link chain", func() {
			resp, err := callChain(ChainUnaryServerInterceptors(doNothing), successHandler)
			So(err, ShouldBeNil)
			So(resp, ShouldEqual, testResponse)
			So(calls, ShouldResemble, []string{
				"-> doNothing",
				"-> successHandler",
				"<- successHandler",
				"<- doNothing",
			})
		})

		Convey("Changes propagate", func() {
			chain := ChainUnaryServerInterceptors(
				populateContext,
				modifyReq,
				doNothing,
				checkContext,
				checkReq,
			)
			resp, err := callChain(chain, successHandler)
			So(err, ShouldBeNil)
			So(resp, ShouldEqual, testResponse)
			So(calls, ShouldResemble, []string{
				"-> populateContext",
				"-> modifyReq",
				"-> doNothing",
				"-> checkContext",
				"-> checkReq",
				"-> successHandler",
				"<- successHandler",
				"<- checkReq",
				"<- checkContext",
				"<- doNothing",
				"<- modifyReq",
				"<- populateContext",
			})
		})

		Convey("Request error propagates", func() {
			chain := ChainUnaryServerInterceptors(
				doNothing,
				checkErr,
			)
			_, err := callChain(chain, errorHandler)
			So(err, ShouldEqual, testError)
			So(calls, ShouldResemble, []string{
				"-> doNothing",
				"-> checkErr",
				"-> errorHandler",
				"<- errorHandler",
				"<- checkErr",
				"<- doNothing",
			})
		})

		Convey("Interceptor can abort the chain", func() {
			chain := ChainUnaryServerInterceptors(
				doNothing,
				abortChain,
				doNothing,
				doNothing,
				doNothing,
				doNothing,
			)
			_, err := callChain(chain, successHandler)
			So(err, ShouldEqual, testError)
			So(calls, ShouldResemble, []string{
				"-> doNothing",
				"-> abortChain",
				"<- abortChain",
				"<- doNothing",
			})
		})
	})
}
