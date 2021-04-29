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

package prjpb

import (
	"go.chromium.org/luci/cv/internal/prjmanager/copyonwrite"
)

// COWPCLs copy-on-write modifies PCLs.
func (p *PState) COWPCLs(m func(*PCL) *PCL, toAdd []*PCL) ([]*PCL, bool) {
	var mf copyonwrite.Modifier
	if m != nil {
		mf = func(v interface{}) interface{} {
			if v := m(v.(*PCL)); v != nil {
				return v
			}
			return copyonwrite.Deletion
		}
	}
	in := cowPCLs(p.GetPcls())
	out, updated := copyonwrite.Update(in, mf, cowPCLs(toAdd))
	return []*PCL(out.(cowPCLs)), updated
}

type cowPCLs []*PCL

func (c cowPCLs) CloneShallow(length int, capacity int) copyonwrite.Slice {
	r := make(cowPCLs, length, capacity)
	copy(r, c[:length])
	return r
}

func (c cowPCLs) Append(v interface{}) copyonwrite.Slice {
	return append(c, v.(*PCL))
}

func (c cowPCLs) Len() int {
	return len(c)
}

func (c cowPCLs) At(index int) interface{} {
	return c[index]
}
