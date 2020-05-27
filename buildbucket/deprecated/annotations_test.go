// Copyright 2018 The LUCI Authors.
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

package deprecated

import (
	"context"
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/golang/protobuf/proto"

	pb "go.chromium.org/luci/buildbucket/proto"
	luci "go.chromium.org/luci/common/proto"
	annotpb "go.chromium.org/luci/common/proto/milo"

	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"
)

func TestAnnotations(t *testing.T) {
	t.Parallel()

	Convey("convert", t, func() {
		basePath := filepath.Join("testdata", "annotations")
		inputPath := filepath.Join(basePath, "annotations.pb.txt")
		wantPath := filepath.Join(basePath, "expected_steps.pb.txt")

		inputFile, err := ioutil.ReadFile(inputPath)
		So(err, ShouldBeNil)

		var ann annotpb.Step
		err = proto.UnmarshalText(string(inputFile), &ann)
		So(err, ShouldBeNil)

		wantFile, err := ioutil.ReadFile(wantPath)
		So(err, ShouldBeNil)

		var want pb.Build
		err = luci.UnmarshalTextML(string(wantFile), &want)
		So(err, ShouldBeNil)

		c := context.Background()
		p := stepConverter{
			defaultLogdogHost:   "logdog.example.com",
			defaultLogdogPrefix: "project/prefix",

			steps: map[string]*pb.Step{},
		}
		var got []*pb.Step
		_, err = p.convertSubsteps(c, &got, ann.Substep, "")
		So(err, ShouldBeNil)

		So(len(got), ShouldEqual, len(want.Steps))
		for i := range got {
			So(got[i], ShouldResembleProto, want.Steps[i])
		}

		Convey("e2e", func() {
			got, err := ConvertBuildSteps(c, ann.Substep, "logdog.example.com", "project/prefix")
			So(err, ShouldBeNil)

			So(len(got), ShouldEqual, len(want.Steps))
			for i := range got {
				So(got[i], ShouldResembleProto, want.Steps[i])
			}
		})
	})
}
