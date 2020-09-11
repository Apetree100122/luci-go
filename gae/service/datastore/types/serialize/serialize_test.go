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

package serialize

import (
	"bytes"
	"fmt"
	"io"
	"testing"
	"time"

	"go.chromium.org/luci/common/data/cmpbin"
	"go.chromium.org/luci/gae/service/blobstore"
	ds "go.chromium.org/luci/gae/service/datastore"

	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"
)

func init() {
	WritePropertyMapDeterministic = true
}

var (
	mp   = ds.MkProperty
	mpNI = ds.MkPropertyNI
)

type dspmapTC struct {
	name  string
	props ds.PropertyMap
}

func mkKey(appID, namespace string, elems ...interface{}) *ds.Key {
	return ds.MkKeyContext(appID, namespace).MakeKey(elems...)
}

func mkBuf(data []byte) WriteableBytesBuffer {
	return Invertible(bytes.NewBuffer(data))
}

// TODO(riannucci): dedup with datastore/key testing file
func ShouldEqualKey(actual interface{}, expected ...interface{}) string {
	if len(expected) != 1 {
		return fmt.Sprintf("Assertion requires 1 expected value, got %d", len(expected))
	}
	if actual.(*ds.Key).Equal(expected[0].(*ds.Key)) {
		return ""
	}
	return fmt.Sprintf("Expected: %q\nActual: %q", actual, expected[0])
}

func TestPropertyMapSerialization(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	tests := []dspmapTC{
		{
			"basic",
			ds.PropertyMap{
				"R": ds.PropertySlice{mp(false), mp(2.1), mpNI(3)},
				"S": ds.PropertySlice{mp("hello"), mp("world")},
			},
		},
		{
			"keys",
			ds.PropertyMap{
				"DS": ds.PropertySlice{
					mp(mkKey("appy", "ns", "Foo", 7)),
					mp(mkKey("other", "", "Yot", "wheeep")),
					mp((*ds.Key)(nil)),
				},
				"blobstore": ds.PropertySlice{mp(blobstore.Key("sup")), mp(blobstore.Key("nerds"))},
			},
		},
		{
			"property map",
			ds.PropertyMap{
				"pm": ds.PropertySlice{
					mpNI(ds.PropertyMap{
						"k": mpNI(mkKey("entity", "id")),
						"map": mpNI(ds.PropertyMap{
							"b": mpNI([]byte("byte")),
						}),
						"str": mpNI("string")},
					),
				},
			},
		},
		{
			"geo",
			ds.PropertyMap{
				"G": mp(ds.GeoPoint{Lat: 1, Lng: 2}),
			},
		},
		{
			"data",
			ds.PropertyMap{
				"S":                    ds.PropertySlice{mp("sup"), mp("fool"), mp("nerd")},
				"Deserialize.Foo.Nerd": ds.PropertySlice{mp([]byte("sup")), mp([]byte("fool"))},
			},
		},
		{
			"time",
			ds.PropertyMap{
				"T": ds.PropertySlice{
					mp(now),
					mp(now.Add(time.Second)),
				},
			},
		},
		{
			"empty vals",
			ds.PropertyMap{
				"T": ds.PropertySlice{mp(true), mp(true)},
				"F": ds.PropertySlice{mp(false), mp(false)},
				"N": ds.PropertySlice{mp(nil), mp(nil)},
				"E": ds.PropertySlice{},
			},
		},
	}

	Convey("PropertyMap serialization", t, func() {
		Convey("round trip", func() {
			for _, tc := range tests {
				tc := tc
				Convey(tc.name, func() {
					data := SerializeKC.ToBytes(tc.props)
					dec, err := Deserialize.PropertyMap(mkBuf(data))
					So(err, ShouldBeNil)
					So(dec, ShouldResemble, tc.props)
				})
			}
		})
	})
}

func die(err error) {
	if err != nil {
		panic(err)
	}
}

func wf(w io.Writer, v float64) int {
	ret, err := cmpbin.WriteFloat64(w, v)
	die(err)
	return ret
}

func ws(w io.ByteWriter, s string) int {
	ret, err := cmpbin.WriteString(w, s)
	die(err)
	return ret
}

func wi(w io.ByteWriter, i int64) int {
	ret, err := cmpbin.WriteInt(w, i)
	die(err)
	return ret
}

func wui(w io.ByteWriter, i uint64) int {
	ret, err := cmpbin.WriteUint(w, i)
	die(err)
	return ret
}

func TestSerializationReadMisc(t *testing.T) {
	t.Parallel()

	Convey("Misc Serialization tests", t, func() {
		Convey("GeoPoint", func() {
			buf := mkBuf(nil)
			wf(buf, 10)
			wf(buf, 20)
			So(string(Serialize.ToBytes(ds.GeoPoint{Lat: 10, Lng: 20})), ShouldEqual, buf.String())
		})

		Convey("IndexColumn", func() {
			buf := mkBuf(nil)
			die(buf.WriteByte(1))
			ws(buf, "hi")
			So(string(Serialize.ToBytes(ds.IndexColumn{Property: "hi", Descending: true})),
				ShouldEqual, buf.String())
		})

		Convey("KeyTok", func() {
			buf := mkBuf(nil)
			ws(buf, "foo")
			die(buf.WriteByte(byte(ds.PTInt)))
			wi(buf, 20)
			So(string(Serialize.ToBytes(ds.KeyTok{Kind: "foo", IntID: 20})),
				ShouldEqual, buf.String())
		})

		Convey("Property", func() {
			buf := mkBuf(nil)
			die(buf.WriteByte(0x80 | byte(ds.PTString)))
			ws(buf, "nerp")
			So(string(Serialize.ToBytes(mp("nerp"))),
				ShouldEqual, buf.String())
		})

		Convey("Time", func() {
			tp := mp(time.Now().UTC())
			So(string(Serialize.ToBytes(tp.Value())), ShouldEqual, string(Serialize.ToBytes(tp)[1:]))
		})

		Convey("Zero time", func() {
			buf := mkBuf(nil)
			So(Serialize.Time(buf, time.Time{}), ShouldBeNil)
			t, err := Deserialize.Time(mkBuf(buf.Bytes()))
			So(err, ShouldBeNil)
			So(t.Equal(time.Time{}), ShouldBeTrue)
		})

		Convey("ReadKey", func() {
			Convey("good cases", func() {
				dwc := Deserializer{ds.MkKeyContext("spam", "nerd")}

				Convey("w/ ctx decodes normally w/ ctx", func() {
					k := mkKey("aid", "ns", "knd", "yo", "other", 10)
					data := SerializeKC.ToBytes(k)
					dk, err := Deserialize.Key(mkBuf(data))
					So(err, ShouldBeNil)
					So(dk, ShouldEqualKey, k)
				})
				Convey("w/ ctx decodes normally w/o ctx", func() {
					k := mkKey("aid", "ns", "knd", "yo", "other", 10)
					data := SerializeKC.ToBytes(k)
					dk, err := dwc.Key(mkBuf(data))
					So(err, ShouldBeNil)
					So(dk, ShouldEqualKey, mkKey("spam", "nerd", "knd", "yo", "other", 10))
				})
				Convey("w/o ctx decodes normally w/ ctx", func() {
					k := mkKey("aid", "ns", "knd", "yo", "other", 10)
					data := Serialize.ToBytes(k)
					dk, err := Deserialize.Key(mkBuf(data))
					So(err, ShouldBeNil)
					So(dk, ShouldEqualKey, mkKey("", "", "knd", "yo", "other", 10))
				})
				Convey("w/o ctx decodes normally w/o ctx", func() {
					k := mkKey("aid", "ns", "knd", "yo", "other", 10)
					data := Serialize.ToBytes(k)
					dk, err := dwc.Key(mkBuf(data))
					So(err, ShouldBeNil)
					So(dk, ShouldEqualKey, mkKey("spam", "nerd", "knd", "yo", "other", 10))
				})
				Convey("IntIDs always sort before StringIDs", func() {
					// -1 writes as almost all 1's in the first byte under cmpbin, even
					// though it's technically not a valid key.
					k := mkKey("aid", "ns", "knd", -1)
					data := Serialize.ToBytes(k)

					k = mkKey("aid", "ns", "knd", "hat")
					data2 := Serialize.ToBytes(k)

					So(string(data), ShouldBeLessThan, string(data2))
				})
			})

			Convey("err cases", func() {
				buf := mkBuf(nil)

				Convey("nil", func() {
					_, err := Deserialize.Key(buf)
					So(err, ShouldEqual, io.EOF)
				})
				Convey("str", func() {
					_, err := buf.WriteString("sup")
					die(err)
					_, err = Deserialize.Key(buf)
					So(err, ShouldErrLike, "expected actualCtx")
				})
				Convey("truncated 1", func() {
					die(buf.WriteByte(1)) // actualCtx == 1
					_, err := Deserialize.Key(buf)
					So(err, ShouldEqual, io.EOF)
				})
				Convey("truncated 2", func() {
					die(buf.WriteByte(1)) // actualCtx == 1
					ws(buf, "aid")
					_, err := Deserialize.Key(buf)
					So(err, ShouldEqual, io.EOF)
				})
				Convey("truncated 3", func() {
					die(buf.WriteByte(1)) // actualCtx == 1
					ws(buf, "aid")
					ws(buf, "ns")
					_, err := Deserialize.Key(buf)
					So(err, ShouldEqual, io.EOF)
				})
				Convey("huge key", func() {
					die(buf.WriteByte(1)) // actualCtx == 1
					ws(buf, "aid")
					ws(buf, "ns")
					for i := 1; i < 60; i++ {
						die(buf.WriteByte(1))
						die(Serialize.KeyTok(buf, ds.KeyTok{Kind: "sup", IntID: int64(i)}))
					}
					die(buf.WriteByte(0))
					_, err := Deserialize.Key(buf)
					So(err, ShouldErrLike, "huge key")
				})
				Convey("insufficient tokens", func() {
					die(buf.WriteByte(1)) // actualCtx == 1
					ws(buf, "aid")
					ws(buf, "ns")
					wui(buf, 2)
					_, err := Deserialize.Key(buf)
					So(err, ShouldEqual, io.EOF)
				})
				Convey("partial token 1", func() {
					die(buf.WriteByte(1)) // actualCtx == 1
					ws(buf, "aid")
					ws(buf, "ns")
					die(buf.WriteByte(1))
					ws(buf, "hi")
					_, err := Deserialize.Key(buf)
					So(err, ShouldEqual, io.EOF)
				})
				Convey("partial token 2", func() {
					die(buf.WriteByte(1)) // actualCtx == 1
					ws(buf, "aid")
					ws(buf, "ns")
					die(buf.WriteByte(1))
					ws(buf, "hi")
					die(buf.WriteByte(byte(ds.PTString)))
					_, err := Deserialize.Key(buf)
					So(err, ShouldEqual, io.EOF)
				})
				Convey("bad token (invalid type)", func() {
					die(buf.WriteByte(1)) // actualCtx == 1
					ws(buf, "aid")
					ws(buf, "ns")
					die(buf.WriteByte(1))
					ws(buf, "hi")
					die(buf.WriteByte(byte(ds.PTBlobKey)))
					_, err := Deserialize.Key(buf)
					So(err, ShouldErrLike, "invalid type PTBlobKey")
				})
				Convey("bad token (invalid IntID)", func() {
					die(buf.WriteByte(1)) // actualCtx == 1
					ws(buf, "aid")
					ws(buf, "ns")
					die(buf.WriteByte(1))
					ws(buf, "hi")
					die(buf.WriteByte(byte(ds.PTInt)))
					wi(buf, -2)
					_, err := Deserialize.Key(buf)
					So(err, ShouldErrLike, "zero/negative")
				})
			})
		})

		Convey("ReadGeoPoint", func() {
			buf := mkBuf(nil)
			Convey("trunc 1", func() {
				_, err := Deserialize.GeoPoint(buf)
				So(err, ShouldEqual, io.EOF)
			})
			Convey("trunc 2", func() {
				wf(buf, 100)
				_, err := Deserialize.GeoPoint(buf)
				So(err, ShouldEqual, io.EOF)
			})
			Convey("invalid", func() {
				wf(buf, 100)
				wf(buf, 1000)
				_, err := Deserialize.GeoPoint(buf)
				So(err, ShouldErrLike, "invalid GeoPoint")
			})
		})

		Convey("WriteTime", func() {
			Convey("in non-UTC!", func() {
				pst, err := time.LoadLocation("America/Los_Angeles")
				So(err, ShouldBeNil)
				So(func() {
					die(Serialize.Time(mkBuf(nil), time.Now().In(pst)))
				}, ShouldPanic)
			})
		})

		Convey("ReadTime", func() {
			Convey("trunc 1", func() {
				_, err := Deserialize.Time(mkBuf(nil))
				So(err, ShouldEqual, io.EOF)
			})
		})

		Convey("ReadProperty", func() {
			buf := mkBuf(nil)
			Convey("trunc 1", func() {
				p, err := Deserialize.Property(buf)
				So(err, ShouldEqual, io.EOF)
				So(p.Type(), ShouldEqual, ds.PTNull)
				So(p.Value(), ShouldBeNil)
			})
			Convey("trunc (PTBytes)", func() {
				die(buf.WriteByte(byte(ds.PTBytes)))
				_, err := Deserialize.Property(buf)
				So(err, ShouldEqual, io.EOF)
			})
			Convey("trunc (PTBlobKey)", func() {
				die(buf.WriteByte(byte(ds.PTBlobKey)))
				_, err := Deserialize.Property(buf)
				So(err, ShouldEqual, io.EOF)
			})
			Convey("invalid type", func() {
				die(buf.WriteByte(byte(ds.PTUnknown + 1)))
				_, err := Deserialize.Property(buf)
				So(err, ShouldErrLike, "unknown type!")
			})
		})

		Convey("ReadPropertyMap", func() {
			buf := mkBuf(nil)
			Convey("trunc 1", func() {
				_, err := Deserialize.PropertyMap(buf)
				So(err, ShouldEqual, io.EOF)
			})
			Convey("too many rows", func() {
				wui(buf, 1000000)
				_, err := Deserialize.PropertyMap(buf)
				So(err, ShouldErrLike, "huge number of rows")
			})
			Convey("trunc 2", func() {
				wui(buf, 10)
				_, err := Deserialize.PropertyMap(buf)
				So(err, ShouldEqual, io.EOF)
			})
			Convey("trunc 3", func() {
				wui(buf, 10)
				ws(buf, "ohai")
				_, err := Deserialize.PropertyMap(buf)
				So(err, ShouldEqual, io.EOF)
			})
			Convey("too many values", func() {
				wui(buf, 10)
				ws(buf, "ohai")
				wui(buf, 100000)
				_, err := Deserialize.PropertyMap(buf)
				So(err, ShouldErrLike, "huge number of properties")
			})
			Convey("trunc 4", func() {
				wui(buf, 10)
				ws(buf, "ohai")
				wui(buf, 10)
				_, err := Deserialize.PropertyMap(buf)
				So(err, ShouldEqual, io.EOF)
			})
		})

		Convey("IndexDefinition", func() {
			id := ds.IndexDefinition{Kind: "kind"}
			data := Serialize.ToBytes(*id.PrepForIdxTable())
			newID, err := Deserialize.IndexDefinition(mkBuf(data))
			So(err, ShouldBeNil)
			So(newID.Flip(), ShouldResemble, id.Normalize())

			id.SortBy = append(id.SortBy, ds.IndexColumn{Property: "prop"})
			data = Serialize.ToBytes(*id.PrepForIdxTable())
			newID, err = Deserialize.IndexDefinition(mkBuf(data))
			So(err, ShouldBeNil)
			So(newID.Flip(), ShouldResemble, id.Normalize())

			id.SortBy = append(id.SortBy, ds.IndexColumn{Property: "other", Descending: true})
			id.Ancestor = true
			data = Serialize.ToBytes(*id.PrepForIdxTable())
			newID, err = Deserialize.IndexDefinition(mkBuf(data))
			So(err, ShouldBeNil)
			So(newID.Flip(), ShouldResemble, id.Normalize())

			// invalid
			id.SortBy = append(id.SortBy, ds.IndexColumn{Property: "", Descending: true})
			data = Serialize.ToBytes(*id.PrepForIdxTable())
			newID, err = Deserialize.IndexDefinition(mkBuf(data))
			So(err, ShouldBeNil)
			So(newID.Flip(), ShouldResemble, id.Normalize())

			Convey("too many", func() {
				id := ds.IndexDefinition{Kind: "wat"}
				for i := 0; i < maxIndexColumns+1; i++ {
					id.SortBy = append(id.SortBy, ds.IndexColumn{Property: "Hi", Descending: true})
				}
				data := Serialize.ToBytes(*id.PrepForIdxTable())
				newID, err = Deserialize.IndexDefinition(mkBuf(data))
				So(err, ShouldErrLike, "over 64 sort orders")
			})
		})
	})
}

func TestPartialSerialization(t *testing.T) {
	t.Parallel()

	fakeKey := mkKey("dev~app", "ns", "parentKind", "sid", "knd", 10)

	Convey("TestPartialSerialization", t, func() {
		Convey("list", func() {
			pm := ds.PropertyMap{
				"wat":  ds.PropertySlice{mpNI("thing"), mp("hat"), mp(100)},
				"nerd": mp(103.7),
				"spaz": mpNI(false),
			}
			sip := Serialize.PropertyMapPartially(fakeKey, pm)
			So(len(sip), ShouldEqual, 4)

			Convey("single collated", func() {
				Convey("indexableMap", func() {
					So(sip, ShouldResemble, SerializedPmap{
						"wat": {
							Serialize.ToBytes(mp("hat")),
							Serialize.ToBytes(mp(100)),
							// 'thing' is skipped, because it's not NoIndex
						},
						"nerd": {
							Serialize.ToBytes(mp(103.7)),
						},
						"__key__": {
							Serialize.ToBytes(mp(fakeKey)),
						},
						"__ancestor__": {
							Serialize.ToBytes(mp(fakeKey)),
							Serialize.ToBytes(mp(fakeKey.Parent())),
						},
					})
				})
			})
		})
	})
}
