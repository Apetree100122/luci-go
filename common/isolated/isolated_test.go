// Copyright 2019 The LUCI Authors.
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

import (
	"crypto"
	"strings"
	"testing"
)

func TestFile(t *testing.T) {
	BasicFile(HexDigest(strings.Repeat("1", 64)), 0600, 64)
	SymLink("l")
	TarFile(HexDigest(strings.Repeat("1", 64)), 64)
}

func TestNew(t *testing.T) {
	i := New(crypto.SHA1)
	if i.Algo != "sha-1" {
		t.Fatal(i.Algo)
	}
	if i.Version != "2.0" {
		t.Fatal(i.Version)
	}
	if len(i.Files) != 0 {
		t.Fatal(i.Files)
	}

	i = New(crypto.SHA256)
	if i.Algo != "sha-256" {
		t.Fatal(i.Algo)
	}
	if i.Version != "2.0" {
		t.Fatal(i.Version)
	}
	if len(i.Files) != 0 {
		t.Fatal(i.Files)
	}

	i = New(crypto.SHA512)
	if i.Algo != "sha-512" {
		t.Fatal(i.Algo)
	}
	if i.Version != "2.0" {
		t.Fatal(i.Version)
	}
	if len(i.Files) != 0 {
		t.Fatal(i.Files)
	}
}

func TestNew_bad(t *testing.T) {
	if a := New(crypto.MD5).Algo; a != "" {
		t.Fatalf("Expected empty invalid algo, got %v", a)
	}
	if a := New(crypto.SHA224).Algo; a != "" {
		t.Fatalf("Expected empty invalid algo, got %v", a)
	}
}
