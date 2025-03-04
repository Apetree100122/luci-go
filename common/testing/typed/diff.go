// Copyright 2023 The LUCI Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package typed is a strictly typed wrapper around cmp.Diff.
package typed

import (
	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/testing/protocmp"
)

// Diff is just like cmp.Diff but it forces got and want to have the same type
// and includes protocmp.Transform().
//
// This will result in more informative compile-time errors.
// protocmp.Transform() is necessary for any protocol buffer comparison, and it
// correctly does nothing if the arguments that we pass in are hereditarily
// non-protobufs. So, for developer convenience, let's just always add it.
func Diff[T any](got T, want T, opts ...cmp.Option) string {
	opts = append(opts, protocmp.Transform())
	return cmp.Diff(got, want, opts...)
}
