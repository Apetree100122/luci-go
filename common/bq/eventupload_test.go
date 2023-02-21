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

package bq

import (
	"testing"
	"time"

	"cloud.google.com/go/bigquery"
	"github.com/golang/protobuf/ptypes"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"go.chromium.org/luci/common/bq/testdata"
	"go.chromium.org/luci/common/clock/testclock"

	. "github.com/smartystreets/goconvey/convey"
)

func TestMetric(t *testing.T) {
	t.Parallel()
	u := Uploader{}
	u.UploadsMetricName = "fakeCounter"
	Convey("Test metric creation", t, func() {
		Convey("Expect uploads metric was created", func() {
			_ = u.getCounter() // To actually create the metric
			So(u.uploads.Info().Name, ShouldEqual, "fakeCounter")
		})
	})
}

func TestSave(t *testing.T) {
	Convey("filled", t, func() {
		recentTime := testclock.TestRecentTimeUTC
		ts, err := ptypes.TimestampProto(recentTime)
		So(err, ShouldBeNil)

		r := &Row{
			Message: &testdata.TestMessage{
				Name:      "testname",
				Timestamp: ts,
				Nested: &testdata.NestedTestMessage{
					Name: "nestedname",
				},
				RepeatedNested: []*testdata.NestedTestMessage{
					{Name: "repeated_one"},
					{Name: "repeated_two"},
				},
				Foo: testdata.TestMessage_Y,
				FooRepeated: []testdata.TestMessage_FOO{
					testdata.TestMessage_Y,
					testdata.TestMessage_X,
				},
				Struct: &structpb.Struct{
					Fields: map[string]*structpb.Value{
						"num": {Kind: &structpb.Value_NumberValue{NumberValue: 1}},
						"str": {Kind: &structpb.Value_StringValue{StringValue: "a"}},
					},
				},

				Empty:    &emptypb.Empty{},
				Empties:  []*emptypb.Empty{{}, {}},
				Duration: durationpb.New(2*time.Second + 3*time.Millisecond),
				OneOf: &testdata.TestMessage_First{First: &testdata.NestedTestMessage{
					Name: "first",
				}},
				StringMap: map[string]string{
					"map_key_1": "map_value_1",
					"map_key_2": "map_value_2",
				},
				BqTypeOverride: 1234,
				StringEnumMap: map[string]testdata.TestMessage_FOO{
					"e_map_key_1": testdata.TestMessage_Y,
					"e_map_key_2": testdata.TestMessage_X,
				},
				StringDurationMap: map[string]*durationpb.Duration{
					"d_map_key_1": durationpb.New(1*time.Second + 1*time.Millisecond),
					"d_map_key_2": durationpb.New(2*time.Second + 2*time.Millisecond),
				},
				StringTimestampMap: map[string]*timestamppb.Timestamp{
					"t_map_key": ts,
				},
				StringProtoMap: map[string]*testdata.NestedTestMessage{
					"p_map_key": {Name: "nestedname"},
				},
			},
			InsertID: "testid",
		}
		row, id, err := r.Save()
		So(err, ShouldBeNil)
		So(id, ShouldEqual, "testid")
		So(row, ShouldResemble, map[string]bigquery.Value{
			"name":      "testname",
			"timestamp": recentTime,
			"nested":    map[string]bigquery.Value{"name": "nestedname"},
			"repeated_nested": []any{
				map[string]bigquery.Value{"name": "repeated_one"},
				map[string]bigquery.Value{"name": "repeated_two"},
			},
			"foo":          "Y",
			"foo_repeated": []any{"Y", "X"},
			"struct":       `{"num":1,"str":"a"}`,
			"duration":     2.003,
			"first":        map[string]bigquery.Value{"name": "first"},
			"string_map": []any{
				map[string]bigquery.Value{"key": "map_key_1", "value": "map_value_1"},
				map[string]bigquery.Value{"key": "map_key_2", "value": "map_value_2"},
			},
			"bq_type_override": int64(1234),
			"string_enum_map": []any{
				map[string]bigquery.Value{"key": "e_map_key_1", "value": "Y"},
				map[string]bigquery.Value{"key": "e_map_key_2", "value": "X"},
			},
			"string_duration_map": []any{
				map[string]bigquery.Value{"key": "d_map_key_1", "value": 1.001},
				map[string]bigquery.Value{"key": "d_map_key_2", "value": 2.002},
			},
			"string_timestamp_map": []any{
				map[string]bigquery.Value{"key": "t_map_key", "value": recentTime},
			},
			"string_proto_map": []any{
				map[string]bigquery.Value{"key": "p_map_key", "value": map[string]bigquery.Value{"name": "nestedname"}},
			},
		})
	})

	Convey("empty", t, func() {
		r := &Row{
			Message:  &testdata.TestMessage{},
			InsertID: "testid",
		}
		row, id, err := r.Save()
		So(err, ShouldBeNil)
		So(id, ShouldEqual, "testid")
		So(row, ShouldResemble, map[string]bigquery.Value{
			// only scalar proto fields
			// because for them, proto3 does not distinguish empty and unset
			// values.
			"bq_type_override": int64(0),
			"foo":              "X", // enums are always set
			"name":             "",  // in proto3, empty string and unset are indistinguishable
		})
	})
}

func TestBatch(t *testing.T) {
	t.Parallel()

	Convey("Test batch", t, func() {
		rowLimit := 2
		rows := make([]*Row, 3)
		for i := 0; i < 3; i++ {
			rows[i] = &Row{}
		}

		want := [][]*Row{
			{{}, {}},
			{{}},
		}
		rowSets := batch(rows, rowLimit)
		So(rowSets, ShouldResemble, want)
	})
}
