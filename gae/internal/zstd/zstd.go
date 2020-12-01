// Copyright 2020 The LUCI Authors.
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

// Package zstd holds zstd encoders and decoders used by the GAE library.
package zstd

import (
	"github.com/klauspost/compress/zstd"
)

// Globally shared zstd encoder and decoder. We use only their EncodeAll and
// DecodeAll methods which are allowed to be used concurrently. Internally, both
// the encode and the decoder have worker pools (limited by GOMAXPROCS) and each
// concurrent EncodeAll/DecodeAll call temporary consumes one worker (so overall
// we do not run more jobs that we have cores for).
var (
	zstdEncoder *zstd.Encoder
	zstdDecoder *zstd.Decoder
)

func init() {
	var err error
	if zstdEncoder, err = zstd.NewWriter(nil); err != nil {
		panic(err) // this is impossible
	}
	if zstdDecoder, err = zstd.NewReader(nil); err != nil {
		panic(err) // this is impossible
	}
}

// EncodeAll will encode all input in src and append it to dst.
//
// This function can be called concurrently, but each call will only run on
// a single goroutine. If empty input is given, nothing is returned. Data
// compressed with EncodeAll can be decoded via DecodeAll.
func EncodeAll(src, dst []byte) []byte {
	return zstdEncoder.EncodeAll(src, dst)
}

// DecodeAll allows stateless decoding of a blob of bytes.
//
// Output will be appended to dst, so if the destination size is known you
// can pre-allocate the destination slice to avoid allocations. DecodeAll can
// be used concurrently.
func DecodeAll(input, dst []byte) ([]byte, error) {
	return zstdDecoder.DecodeAll(input, dst)
}
