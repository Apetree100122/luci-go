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

package iotools

import (
	"bytes"
	"errors"
	"io"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

type infiniteReader struct{}

func (r infiniteReader) Read(b []byte) (int, error) {
	for idx := 0; idx < len(b); idx++ {
		b[idx] = 0x55
	}
	return len(b), nil
}

type errorReader struct {
	error
}

func (e *errorReader) Read([]byte) (int, error) {
	return 0, e.error
}

func TestChainReader(t *testing.T) {
	Convey(`An empty ChainReader`, t, func() {
		cr := ChainReader{}

		Convey(`Should successfully read into a zero-byte array.`, func() {
			d := []byte{}
			count, err := cr.Read(d)
			So(count, ShouldEqual, 0)
			So(err, ShouldBeNil)
		})

		Convey(`Should fail with io.EOF during ReadByte.`, func() {
			b, err := cr.ReadByte()
			So(b, ShouldEqual, 0)
			So(err, ShouldEqual, io.EOF)
		})

		Convey(`Should have zero remaining bytes.`, func() {
			So(cr.Remaining(), ShouldEqual, 0)
		})
	})

	Convey(`A ChainReader with {{0x00, 0x01}, nil, nil, {0x02}, nil}`, t, func() {
		cr := ChainReader{bytes.NewReader([]byte{0x00, 0x01}), nil, nil, bytes.NewReader([]byte{0x02}), nil}

		Convey(`The ChainReader should have a Remaining count of 3.`, func() {
			So(cr.Remaining(), ShouldEqual, 3)
		})

		Convey(`The ChainReader should read: []byte{0x00, 0x01, 0x02} for buffer size 3.`, func() {
			data := make([]byte, 3)
			count, err := cr.Read(data)
			So(count, ShouldEqual, 3)
			So(err, ShouldBeNil)
			So(data, ShouldResemble, []byte{0x00, 0x01, 0x02})

			So(cr.Remaining(), ShouldEqual, 0)
		})

		Convey(`The ChainReader should read: []byte{0x00, 0x01} for buffer size 2.`, func() {
			data := make([]byte, 2)
			count, err := cr.Read(data)
			So(count, ShouldEqual, 2)
			So(err, ShouldBeNil)
			So(data, ShouldResemble, []byte{0x00, 0x01})

			So(cr.Remaining(), ShouldEqual, 1)
		})

		Convey(`The ChainReader should read bytes: 0x00, 0x01, 0x02, EOF.`, func() {
			b, err := cr.ReadByte()
			So(b, ShouldEqual, 0x00)
			So(err, ShouldBeNil)

			b, err = cr.ReadByte()
			So(b, ShouldEqual, 0x01)
			So(err, ShouldBeNil)

			b, err = cr.ReadByte()
			So(b, ShouldEqual, 0x02)
			So(err, ShouldBeNil)

			b, err = cr.ReadByte()
			So(b, ShouldEqual, 0x00)
			So(err, ShouldEqual, io.EOF)

			So(cr.Remaining(), ShouldEqual, 0)
		})
	})

	Convey(`A ChainReader with an infinite io.Reader`, t, func() {
		cr := ChainReader{&infiniteReader{}}

		Convey(`Should return an error on RemainingErr()`, func() {
			_, err := cr.RemainingErr()
			So(err, ShouldNotBeNil)
		})

		Convey(`Should panic on Remaining()`, func() {
			So(func() { cr.Remaining() }, ShouldPanic)
		})

		Convey(`Should fill a 1024-byte buffer`, func() {
			data := make([]byte, 1024)
			count, err := cr.Read(data)
			So(count, ShouldEqual, 1024)
			So(err, ShouldBeNil)
			So(data, ShouldResemble, bytes.Repeat([]byte{0x55}, 1024))
		})
	})

	Convey(`A ChainReader with {0x00, 0x01} and an error-returning io.Reader`, t, func() {
		e := errors.New("TEST ERROR")
		cr := ChainReader{bytes.NewReader([]byte{0x00, 0x01}), &errorReader{e}}

		Convey(`Should fill a 3-byte buffer with the first two bytes and return an error.`, func() {
			data := make([]byte, 3)
			count, err := cr.Read(data)
			So(count, ShouldEqual, 2)
			So(err, ShouldEqual, e)
			So(data[:2], ShouldResemble, []byte{0x00, 0x01})
		})
	})
}
