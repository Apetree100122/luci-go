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

package cipd

import (
	"context"
	"os"
)

// deleteOnClose is os.File that self-deletes once it closes.
//
// Implements pkg.Source interface. Used by fetchInstanceNoCache.
type deleteOnClose struct {
	*os.File
	size int64
}

// Size returns the file size.
func (d *deleteOnClose) Size() int64 {
	return d.size
}

// Close closes the underlying file and then deletes it.
func (d *deleteOnClose) Close(ctx context.Context, corrupt bool) (err error) {
	name := d.File.Name()
	defer func() {
		if rmErr := os.Remove(name); err == nil && rmErr != nil && !os.IsNotExist(rmErr) {
			err = rmErr
		}
	}()
	return d.File.Close()
}

// UnderlyingFile is only used by tests and shouldn't be used directly.
func (d *deleteOnClose) UnderlyingFile() *os.File {
	return d.File
}
