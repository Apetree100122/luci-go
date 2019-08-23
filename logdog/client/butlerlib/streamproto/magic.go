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

package streamproto

var (
	// ProtocolFrameHeaderMagic is the number at the beginning of streams that
	// identifies the stream handshake version.
	//
	// This serves two purposes:
	//   - To disambiguate a Butler stream from some happenstance string of bytes
	//     (which probably won't start with these characters).
	//   - To allow an upgrade to the wire format, if one is ever needed. e.g.,
	//     a switch to something other than recordio/JSON.
	ProtocolFrameHeaderMagic = []byte("BTLR1\x1E")
)

// LocalNamedPipePath returns the path to a local Windows named pipe named
// `base`. This is used with the 'net.pipe' butler protocol.
func LocalNamedPipePath(base string) string {
	return `\\.\pipe\` + base
}
