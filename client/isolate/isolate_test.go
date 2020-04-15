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

package isolate

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"go.chromium.org/luci/client/archiver"
	isolateservice "go.chromium.org/luci/common/api/isolate/isolateservice/v1"
	"go.chromium.org/luci/common/data/text/units"
	"go.chromium.org/luci/common/flag/stringlistflag"
	"go.chromium.org/luci/common/isolated"
	"go.chromium.org/luci/common/isolatedclient"
	"go.chromium.org/luci/common/isolatedclient/isolatedfake"
	"go.chromium.org/luci/common/system/filesystem"
	"go.chromium.org/luci/common/testing/testfs"

	. "github.com/smartystreets/goconvey/convey"
)

func init() {
	log.SetOutput(ioutil.Discard)
}

func TestReplaceVars(t *testing.T) {
	t.Parallel()
	Convey(`Variables replacement should be supported in isolate files.`, t, func() {

		opts := &ArchiveOptions{PathVariables: map[string]string{"VAR": "wonderful"}}

		// Single replacement.
		r, err := ReplaceVariables("hello <(VAR) world", opts)
		So(err, ShouldBeNil)
		So(r, ShouldResemble, "hello wonderful world")

		// Multiple replacement.
		r, err = ReplaceVariables("hello <(VAR) <(VAR) world", opts)
		So(err, ShouldBeNil)
		So(r, ShouldResemble, "hello wonderful wonderful world")

		// Replacement of missing variable.
		r, err = ReplaceVariables("hello <(MISSING) world", opts)
		So(err.Error(), ShouldResemble, "no value for variable 'MISSING'")
	})
}

func TestArchive(t *testing.T) {
	// Create a .isolate file and archive it.
	t.Parallel()
	ctx := context.Background()

	Convey(`Tests the creation and archival of an isolate file.`, t, func() {
		server := isolatedfake.New()
		ts := httptest.NewServer(server)
		defer ts.Close()
		namespace := isolatedclient.DefaultNamespace
		a := archiver.New(ctx, isolatedclient.NewClient(ts.URL, isolatedclient.WithNamespace(namespace)), nil)

		// Setup temporary directory.
		//   /base/bar
		//   /base/ignored
		//   /foo/baz.isolate
		//   /link -> /base/bar
		//   /second/boz
		// Result:
		//   /baz.isolated
		tmpDir, err := ioutil.TempDir("", "isolate")
		So(err, ShouldBeNil)
		defer func() {
			if err := os.RemoveAll(tmpDir); err != nil {
				t.Fail()
			}
		}()
		baseDir := filepath.Join(tmpDir, "base")
		fooDir := filepath.Join(tmpDir, "foo")
		secondDir := filepath.Join(tmpDir, "second")
		So(os.Mkdir(baseDir, 0700), ShouldBeNil)
		So(os.Mkdir(fooDir, 0700), ShouldBeNil)
		So(os.Mkdir(secondDir, 0700), ShouldBeNil)
		So(ioutil.WriteFile(filepath.Join(baseDir, "bar"), []byte("foo"), 0600), ShouldBeNil)
		So(ioutil.WriteFile(filepath.Join(baseDir, "ignored"), []byte("ignored"), 0600), ShouldBeNil)
		So(ioutil.WriteFile(filepath.Join(secondDir, "boz"), []byte("foo2"), 0600), ShouldBeNil)
		isolate := `{
		'variables': {
			'files': [
				'../base/',
				'../second/',
				'../link',
			],
		},
		'conditions': [
			['OS=="amiga"', {
				'variables': {
					'command': ['amiga'],
				},
			}],
			['OS=="win"', {
				'variables': {
					'command': ['win'],
				},
			}],
		],
	}`
		isolatePath := filepath.Join(fooDir, "baz.isolate")
		So(ioutil.WriteFile(isolatePath, []byte(isolate), 0600), ShouldBeNil)
		if runtime.GOOS != "windows" {
			So(os.Symlink(filepath.Join("base", "bar"), filepath.Join(tmpDir, "link")), ShouldBeNil)
		} else {
			So(ioutil.WriteFile(filepath.Join(tmpDir, "link"), []byte("no link on Windows"), 0600), ShouldBeNil)
		}
		opts := &ArchiveOptions{
			Isolate:                    isolatePath,
			Isolated:                   filepath.Join(tmpDir, "baz.isolated"),
			Blacklist:                  stringlistflag.Flag{"ignored", "*.isolate"},
			PathVariables:              map[string]string{"VAR": "wonderful"},
			ConfigVariables:            map[string]string{"OS": "amiga"},
			AllowCommandAndRelativeCWD: true,
		}
		item := Archive(a, opts)
		So(item.DisplayName, ShouldResemble, "baz.isolated")
		item.WaitForHashed()
		So(item.Error(), ShouldBeNil)
		So(a.Close(), ShouldBeNil)

		mode := 0600
		if runtime.GOOS == "windows" {
			mode = 0666
		}

		//   /base/
		baseIsolatedData := isolated.Isolated{
			Algo: "sha-1",
			Files: map[string]isolated.File{
				filepath.Join("base", "bar"): isolated.BasicFile("0beec7b5ea3f0fdbc95d0dd47f3c5bc275da8a33", mode, 3),
			},
			Version: isolated.IsolatedFormatVersion,
		}
		encoded, err := json.Marshal(baseIsolatedData)
		So(err, ShouldBeNil)
		baseIsolatedEncoded := string(encoded) + "\n"
		h := isolated.GetHash(namespace)
		baseIsolatedHash := isolated.HashBytes(h, []byte(baseIsolatedEncoded))

		//   /second/
		secondIsolatedData := isolated.Isolated{
			Algo: "sha-1",
			Files: map[string]isolated.File{
				filepath.Join("second", "boz"): isolated.BasicFile("aaadd94977b8fbf3f6fb09fc3bbbc9edbdfa8427", mode, 4),
			},
			Version: isolated.IsolatedFormatVersion,
		}
		encoded, err = json.Marshal(secondIsolatedData)
		So(err, ShouldBeNil)
		secondIsolatedEncoded := string(encoded) + "\n"
		secondIsolatedHash := isolated.HashBytes(h, []byte(secondIsolatedEncoded))

		isolatedData := isolated.Isolated{
			Algo:    "sha-1",
			Command: []string{"amiga"},
			Files:   map[string]isolated.File{},
			// This list must be in deterministic order.
			Includes:    isolated.HexDigests{baseIsolatedHash, secondIsolatedHash},
			RelativeCwd: "foo",
			Version:     isolated.IsolatedFormatVersion,
		}
		if runtime.GOOS != "windows" {
			isolatedData.Files["link"] = isolated.SymLink(filepath.Join("base", "bar"))
		} else {
			isolatedData.Files["link"] = isolated.BasicFile("12339b9756c2994f85c310d560bc8c142a6b79a1", 0666, 18)
		}
		encoded, err = json.Marshal(isolatedData)
		So(err, ShouldBeNil)
		isolatedEncoded := string(encoded) + "\n"
		isolatedHash := isolated.HashBytes(h, []byte(isolatedEncoded))

		expected := map[string]map[isolated.HexDigest]string{
			namespace: {
				"0beec7b5ea3f0fdbc95d0dd47f3c5bc275da8a33": "foo",
				"aaadd94977b8fbf3f6fb09fc3bbbc9edbdfa8427": "foo2",
				baseIsolatedHash:   baseIsolatedEncoded,
				isolatedHash:       isolatedEncoded,
				secondIsolatedHash: secondIsolatedEncoded,
			},
		}
		if runtime.GOOS == "windows" {
			// TODO(maruel): Fix symlink support on Windows.
			expected[namespace]["12339b9756c2994f85c310d560bc8c142a6b79a1"] = "no link on Windows"
		}
		actual := map[string]map[isolated.HexDigest]string{}
		for n, c := range server.Contents() {
			actual[n] = map[isolated.HexDigest]string{}
			for k, v := range c {
				actual[n][k] = string(v)
			}
		}
		So(actual, ShouldResemble, expected)
		So(item.Digest(), ShouldResemble, isolatedHash)

		stats := a.Stats()
		So(stats.TotalHits(), ShouldBeZeroValue)
		So(stats.TotalBytesHits(), ShouldResemble, units.Size(0))
		size := 3 + 4 + len(baseIsolatedEncoded) + len(isolatedEncoded) + len(secondIsolatedEncoded)
		if runtime.GOOS != "windows" {
			So(stats.TotalMisses(), ShouldEqual, 5)
			So(stats.TotalBytesPushed(), ShouldResemble, units.Size(size))
		} else {
			So(stats.TotalMisses(), ShouldEqual, 6)
			// Includes the duplicate due to lack of symlink.
			So(stats.TotalBytesPushed(), ShouldResemble, units.Size(size+18))
		}

		So(server.Error(), ShouldBeNil)
		digest, err := isolated.HashFile(h, filepath.Join(tmpDir, "baz.isolated"))
		So(digest, ShouldResemble, isolateservice.HandlersEndpointsV1Digest{Digest: string(isolatedHash), IsIsolated: false, Size: int64(len(isolatedEncoded))})
		So(err, ShouldBeNil)
	})
}

// Test that if the isolate file is not found, the error is properly propagated.
func TestArchiveFileNotFoundReturnsError(t *testing.T) {
	t.Parallel()

	Convey(`The client should handle missing isolate files.`, t, func() {
		a := archiver.New(context.Background(), isolatedclient.NewClient("http://unused"), nil)
		opts := &ArchiveOptions{
			Isolate:  "/this-file-does-not-exist",
			Isolated: "/this-file-doesnt-either",
		}
		item := Archive(a, opts)
		item.WaitForHashed()
		err := item.Error()
		So(strings.HasPrefix(err.Error(), "open /this-file-does-not-exist: "), ShouldBeTrue)
		// The archiver itself hasn't failed, it's Archive() that did.
		So(a.Close(), ShouldBeNil)
	})
}

// Test archival of a single directory.
func TestArchiveDir(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	Convey(`Isolating a single directory should archive the contents, not the dir`, t, testfs.MustWithTempDir(t, "", func(tmpDir string) {
		server := isolatedfake.New()
		ts := httptest.NewServer(server)
		defer ts.Close()
		namespace := isolatedclient.DefaultNamespace
		a := archiver.New(ctx, isolatedclient.NewClient(ts.URL, isolatedclient.WithNamespace(namespace)), nil)

		// Setup temporary directory.
		//   /base/subdir/foo
		//   /base/subdir/bar
		//   /base/subdir/baz.isolate
		// Result:
		//   /baz.isolated
		subDir := filepath.Join(tmpDir, "base", "subdir")
		So(os.MkdirAll(subDir, 0700), ShouldBeNil)
		So(ioutil.WriteFile(filepath.Join(subDir, "foo"), []byte("foo"), 0600), ShouldBeNil)
		So(ioutil.WriteFile(filepath.Join(subDir, "bar"), []byte("bar"), 0600), ShouldBeNil)
		isolate := `{
                  'variables': {
                    'files': [
                      './',
                    ],
                  },
                }`
		isolatePath := filepath.Join(subDir, "baz.isolate")
		So(ioutil.WriteFile(isolatePath, []byte(isolate), 0600), ShouldBeNil)

		opts := &ArchiveOptions{
			Isolate:  isolatePath,
			Isolated: filepath.Join(tmpDir, "baz.isolated"),
		}
		item := Archive(a, opts)
		So(item.DisplayName, ShouldResemble, "baz.isolated")
		item.WaitForHashed()
		So(item.Error(), ShouldBeNil)
		So(a.Close(), ShouldBeNil)

		mode := 0600
		if runtime.GOOS == "windows" {
			mode = 0666
		}

		//   /subdir/
		fooHash := isolated.HexDigest("0beec7b5ea3f0fdbc95d0dd47f3c5bc275da8a33")
		barHash := isolated.HexDigest("62cdb7020ff920e5aa642c3d4066950dd1f01f4d")
		bazHash := isolated.HexDigest("dfe2138a5f964f254b334255217f719367cd1231")
		subdirIsolatedData := isolated.Isolated{
			Algo: "sha-1",
			Files: map[string]isolated.File{
				"foo":         isolated.BasicFile(fooHash, mode, 3),
				"bar":         isolated.BasicFile(barHash, mode, 3),
				"baz.isolate": isolated.BasicFile(bazHash, mode, 155),
			},
			Version: isolated.IsolatedFormatVersion,
		}
		encoded, err := json.Marshal(subdirIsolatedData)
		So(err, ShouldBeNil)
		subdirIsolatedEncoded := string(encoded) + "\n"
		h := isolated.GetHash(namespace)
		subdirIsolatedHash := isolated.HashBytes(h, []byte(subdirIsolatedEncoded))

		isolatedData := isolated.Isolated{
			Algo:     "sha-1",
			Files:    map[string]isolated.File{},
			Includes: isolated.HexDigests{subdirIsolatedHash},
			Version:  isolated.IsolatedFormatVersion,
		}
		encoded, err = json.Marshal(isolatedData)
		So(err, ShouldBeNil)
		isolatedEncoded := string(encoded) + "\n"
		isolatedHash := isolated.HashBytes(h, []byte(isolatedEncoded))

		expected := map[string]map[isolated.HexDigest]string{
			namespace: {
				fooHash:            "foo",
				barHash:            "bar",
				bazHash:            isolate,
				subdirIsolatedHash: subdirIsolatedEncoded,
				isolatedHash:       isolatedEncoded,
			},
		}

		actual := map[string]map[isolated.HexDigest]string{}
		for n, c := range server.Contents() {
			actual[n] = map[isolated.HexDigest]string{}
			for k, v := range c {
				actual[n][k] = string(v)
			}
		}
		So(actual, ShouldResemble, expected)
		So(item.Digest(), ShouldResemble, isolatedHash)
	}))
}

func TestProcessIsolateFile(t *testing.T) {
	t.Parallel()

	Convey(`Directory deps should end with osPathSeparator`, t, testfs.MustWithTempDir(t, "", func(tmpDir string) {
		baseDir := filepath.Join(tmpDir, "baseDir")
		secondDir := filepath.Join(tmpDir, "secondDir")
		So(os.Mkdir(baseDir, 0700), ShouldBeNil)
		So(os.Mkdir(secondDir, 0700), ShouldBeNil)
		So(ioutil.WriteFile(filepath.Join(baseDir, "foo"), []byte("foo"), 0600), ShouldBeNil)
		// Note that for "secondDir", its separator is omitted intentionally.
		isolate := `{
			'variables': {
				'files': [
					'../baseDir/',
					'../secondDir',
					'../baseDir/foo',
				],
			},
		}`

		outDir := filepath.Join(tmpDir, "out")
		So(os.Mkdir(outDir, 0700), ShouldBeNil)
		isolatePath := filepath.Join(outDir, "my.isolate")
		So(ioutil.WriteFile(isolatePath, []byte(isolate), 0600), ShouldBeNil)

		opts := &ArchiveOptions{
			Isolate: isolatePath,
		}

		deps, _, _, err := ProcessIsolate(opts)
		So(err, ShouldBeNil)
		for _, dep := range deps {
			isDir, err := filesystem.IsDir(dep)
			So(err, ShouldBeNil)
			So(strings.HasSuffix(dep, osPathSeparator), ShouldEqual, isDir)
		}
	}))
}
