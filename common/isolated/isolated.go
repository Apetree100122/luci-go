// Copyright 2015 The LUCI Authors.
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

package isolated

import "crypto"

// IsolatedFormatVersion is version of *.isolated file format. Put into JSON.
const IsolatedFormatVersion = "2.0"

// FileType describes the type of file being isolated.
type FileType string

const (
	// Basic represents normal files. It is the default type.
	Basic FileType = "basic"

	// ArArchive represents an ar archive containing a large number of small files.
	ArArchive FileType = "ar"

	// TarArchive represents a tar archive containing a large number of small files.
	TarArchive FileType = "tar"
)

// File describes a single file referenced by content in a .isolated file.
//
// For regular files, the Digest, Mode, and Size fields should be set, and the
// Type field should be set for non-basic files.
// For symbolic links, only the Link field should be set.
type File struct {
	Digest HexDigest `json:"h,omitempty"`
	Link   *string   `json:"l,omitempty"`
	Mode   *int      `json:"m,omitempty"`
	Size   *int64    `json:"s,omitempty"`
	Type   FileType  `json:"t,omitempty"`
}

// BasicFile returns a File populated for a basic file.
func BasicFile(d HexDigest, mode int, size int64) File {
	return File{
		Digest: d,
		Mode:   &mode,
		Size:   &size,
	}
}

// SymLink returns a File populated for a symbolic link.
func SymLink(link string) File {
	return File{
		Link: &link,
	}
}

// TarFile returns a file populated for a tar archive file.
func TarFile(d HexDigest, size int64) File {
	return File{
		Digest: d,
		Size:   &size,
		Type:   TarArchive,
	}
}

// Isolated is the data from a JSON serialized .isolated file.
type Isolated struct {
	Algo     string          `json:"algo"` // Must be "sha-1"
	Files    map[string]File `json:"files,omitempty"`
	Includes HexDigests      `json:"includes,omitempty"`
	Version  string          `json:"version"`
}

// New returns a new Isolated with the default Algo and Version.
func New(h crypto.Hash) *Isolated {
	a := ""
	switch h {
	case crypto.SHA1:
		a = "sha-1"
	case crypto.SHA256:
		a = "sha-256"
	case crypto.SHA512:
		a = "sha-512"
	}
	return &Isolated{
		Algo:    a,
		Version: IsolatedFormatVersion,
		Files:   map[string]File{},
	}
}
