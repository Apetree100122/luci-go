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

package state

import (
	"testing"

	"go.chromium.org/luci/cv/internal/changelist"
	"go.chromium.org/luci/cv/internal/prjmanager/prjpb"

	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"
)

func TestUpgradeIfNecessary(t *testing.T) {
	t.Parallel()

	Convey("UpgradeIfNecessary works", t, func() {
		s0 := &State{PB: &prjpb.PState{
			Pcls: []*prjpb.PCL{
				{Clid: 1},
				{Clid: 2},
			},
		}}

		Convey("not necessary", func() {
			pb := backupPB(s0)
			s1 := s0.UpgradeIfNecessary()
			So(s1, ShouldEqual, s0)
			So(s0.PB, ShouldResembleProto, pb)
		})

		Convey("necessary", func() {
			s0.PB.Pcls[0].OwnerLacksEmail = true
			pb := backupPB(s0)
			s1 := s0.UpgradeIfNecessary()
			So(s0.PB, ShouldResembleProto, pb)

			So(s1, ShouldNotEqual, s0)
			pb.Pcls[0].OwnerLacksEmail = false
			pb.Pcls[0].Errors = []*changelist.CLError{
				{Kind: &changelist.CLError_OwnerLacksEmail{OwnerLacksEmail: true}},
			}
			So(s1.PB, ShouldResembleProto, pb)
		})
	})
}
