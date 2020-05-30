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

package lucictx

import (
	"context"
)

// GetLUCIExe calls Lookup and returns a copy of the current LUCIExe from
// LUCI_CONTEXT if it was present. If no LUCIExe is in the context, this
// returns nil.
func GetLUCIExe(ctx context.Context) *LUCIExe {
	ret := LUCIExe{}
	ok, err := Lookup(ctx, "luciexe", &ret)
	if err != nil {
		panic(err)
	}
	if !ok {
		return nil
	}
	return &ret
}

// SetLUCIExe sets the LUCIExe in the LUCI_CONTEXT.
func SetLUCIExe(ctx context.Context, le *LUCIExe) context.Context {
	return Set(ctx, "luciexe", le)
}
