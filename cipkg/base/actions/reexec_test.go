// Copyright 2023 The LUCI Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package actions

import (
	"context"
	"errors"
	"testing"

	"go.chromium.org/luci/cipkg/core"
	"go.chromium.org/luci/common/system/environ"

	. "github.com/smartystreets/goconvey/convey"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

func TestSetExecutor(t *testing.T) {
	Convey("Test set executor", t, func() {
		reg := NewReexecRegistry()

		Convey("ok", func() {
			err := SetExecutor[*anypb.Any](reg, func(ctx context.Context, msg *anypb.Any, out string) error { return nil })
			So(err, ShouldBeNil)
		})

		Convey("duplicated", func() {
			err := SetExecutor[*anypb.Any](reg, func(ctx context.Context, msg *anypb.Any, out string) error { return nil })
			So(err, ShouldBeNil)
			err = SetExecutor[*anypb.Any](reg, func(ctx context.Context, msg *anypb.Any, out string) error { return nil })
			So(errors.Is(err, ErrExecutorExisted), ShouldBeTrue)
		})

		Convey("sealed", func() {
			reg.interceptWithArgs(context.Background(), environ.New(nil), []string{}, func(i int) {})
			err := SetExecutor[*anypb.Any](reg, func(ctx context.Context, msg *anypb.Any, out string) error { return nil })
			So(errors.Is(err, ErrReexecRegistrySealed), ShouldBeTrue)
		})
	})
}

func runWithDrv(ctx context.Context, drv *core.Derivation, out string) {
	env := environ.New(drv.Env)
	env.Set("out", out)
	code := -1
	NewReexecRegistry().interceptWithArgs(ctx, env, drv.Args, func(i int) {
		code = i
	})
	So(code, ShouldEqual, 0)
}

func checkReexecArg(args []string, m proto.Message) {
	m, err := anypb.New(m)
	So(err, ShouldBeNil)
	b, err := protojson.Marshal(m)
	So(err, ShouldBeNil)
	So(args, ShouldContain, string(b))
}
