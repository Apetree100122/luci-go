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

package datastore

import (
	"bytes"
	"context"
	"strconv"
	"testing"

	"go.chromium.org/luci/common/data/cmpbin"
	"go.chromium.org/luci/gae/service/info"

	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"
)

func TestDatastoreQueryIterator(t *testing.T) {
	t.Parallel()
	Convey("queryIterator", t, func() {
		Convey("normal", func() {
			qi := queryIterator{
				order: []IndexColumn{
					{Property: "field1", Descending: true},
					{Property: "field2"},
					{Property: "__key__"},
				},
				itemCh: make(chan *rawQueryResult),
			}

			key := MkKeyContext("s~aid", "ns").MakeKey("testKind", 1)
			// populating results to the pipeline
			go func() {
				defer close(qi.itemCh)
				qi.itemCh <- &rawQueryResult{
					key: key,
					data: PropertyMap{
						"field1": PropertySlice{
							MkProperty("1"),
							MkProperty("11"),
						},
						"field2": MkProperty("aa1"),
					},
				}
			}()
			err := qi.Next()
			So(err, ShouldBeNil)
			So(qi.currentQueryResult, ShouldNotBeNil)

			Convey("CurrentItemKey", func() {

				itemKey := qi.CurrentItemKey()
				expectedKey := MkKeyContext("s~aid", "ns").MakeKey("testKind", 1)
				e := string(Serialize.ToBytes(expectedKey))
				So(itemKey, ShouldEqual, e)
			})

			Convey("CurrentItemOrder", func() {
				itemOrder := qi.CurrentItemOrder()
				So(err, ShouldBeNil)

				invBuf := cmpbin.Invertible(&bytes.Buffer{})
				invBuf.SetInvert(true)
				Serialize.Property(invBuf, MkProperty(strconv.Itoa(11)))
				invBuf.SetInvert(false)
				Serialize.Property(invBuf, MkProperty("aa1"))
				Serialize.Key(invBuf, key)

				So(itemOrder, ShouldEqual, invBuf.String())
			})

			Convey("CurrentItem", func() {
				key, data := qi.CurrentItem()
				expectedPM := PropertyMap{
					"field1": PropertySlice{
						MkProperty("1"),
						MkProperty("11"),
					},
					"field2": MkProperty("aa1"),
				}
				So(key, ShouldResemble, key)
				So(data, ShouldResemble, expectedPM)
			})

			// end of results
			err = qi.Next()
			So(err, ShouldResemble, Stop)
		})

		Convey("invalid queryIterator", func() {
			qi := queryIterator{}
			So(func() { qi.Next() }, ShouldPanicWith,
				"item channel for queryIterator is not properly initiated")
		})

		Convey("empty query results", func() {
			qi := &queryIterator{
				order:  []IndexColumn{},
				itemCh: make(chan *rawQueryResult),
			}
			go func() {
				qi.itemCh <- &rawQueryResult{
					key:  nil,
					data: PropertyMap{},
				}
				close(qi.itemCh)
			}()

			err := qi.Next()
			So(err, ShouldBeNil)
			So(qi.CurrentItemKey(), ShouldEqual, "")
			itemOrder := qi.CurrentItemOrder()
			So(itemOrder, ShouldEqual, "")
			key, data := qi.CurrentItem()
			So(key, ShouldBeNil)
			So(data, ShouldResemble, PropertyMap{})
		})
	})

}

func TestStartQueryIterator(t *testing.T) {
	t.Parallel()
	Convey("start queryIterator", t, func() {
		ctx := info.Set(context.Background(), fakeInfo{})
		fds := fakeDatastore{}
		ctx = SetRawFactory(ctx, fds.factory())
		ctx, cancel := context.WithCancel(ctx)

		fds.entities = 2
		dq := NewQuery("Kind").Order("Value")

		Convey("found", func() {
			fq, err := dq.Finalize()
			So(err, ShouldBeNil)
			qi := startQueryIterator(ctx, fq)

			err = qi.Next()
			So(err, ShouldBeNil)
			So(qi.currentQueryResult.key, ShouldResemble, MakeKey(ctx, "Kind", 1))
			So(qi.currentQueryResult.data, ShouldResemble,
				PropertyMap{
					"Value": MkProperty(0),
				})

			err = qi.Next()
			So(err, ShouldBeNil)
			So(qi.currentQueryResult.key, ShouldResemble, MakeKey(ctx, "Kind", 2))
			So(qi.currentQueryResult.data, ShouldResemble,
				PropertyMap{
					"Value": MkProperty(1),
				})
			So(qi.currentItemOrderCache, ShouldEqual, "")
			order := qi.CurrentItemOrder()
			So(qi.currentItemOrderCache, ShouldEqual, order)

			err = qi.Next()
			So(err, ShouldResemble, Stop)
		})

		Convey("cancel", func() {
			fq, err := dq.Finalize()
			So(err, ShouldBeNil)
			qi := startQueryIterator(ctx, fq)

			cancel()
			<-ctx.Done() // wait till the cancellation propagates

			// When calling `cancel()`, one rawQueryResult may already be put into the itemCh.
			// So it asserts the two possible scenarios: 1) one rawQueryResult with a followed Stop signal.
			// 2) qi.Next() directly returns a Stop signal.
			err = qi.Next()
			if err == nil {
				So(qi.currentQueryResult, ShouldResemble, &rawQueryResult{
					key: MakeKey(ctx, "Kind", 1),
					data: PropertyMap{
						"Value": MkProperty(0),
					},
				})
				err = qi.Next()
				So(err, ShouldResemble, Stop)
			} else {
				So(err, ShouldResemble, Stop)
			}
		})

		Convey("not found", func() {
			fds.entities = 0
			fq, err := dq.Finalize()
			So(err, ShouldBeNil)
			qi := startQueryIterator(ctx, fq)

			err = qi.Next()
			So(err, ShouldResemble, Stop)
		})

		Convey("errors from raw datastore", func() {
			dq = dq.Eq("$err_single", "Query fail").Eq("$err_single_idx", 0)
			fq, err := dq.Finalize()
			So(err, ShouldBeNil)
			qi := startQueryIterator(ctx, fq)

			err = qi.Next()
			So(err, ShouldErrLike, "Query fail")
		})
	})
}
