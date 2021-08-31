// Copyright 2016 The LUCI Authors.
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

package monitor

import (
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	"go.chromium.org/luci/common/tsmon/distribution"
	"go.chromium.org/luci/common/tsmon/field"
	"go.chromium.org/luci/common/tsmon/target"
	"go.chromium.org/luci/common/tsmon/types"
	"google.golang.org/protobuf/types/known/timestamppb"

	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"
	pb "go.chromium.org/luci/common/tsmon/ts_mon_proto"
)

func TestSerializeDistribution(t *testing.T) {
	Convey("Fixed width params", t, func() {
		d := distribution.New(distribution.FixedWidthBucketer(10, 20))
		dpb := serializeDistribution(d)

		So(dpb, ShouldResembleProto, &pb.MetricsData_Distribution{
			Count: proto.Int64(0),
			BucketOptions: &pb.MetricsData_Distribution_LinearBuckets{
				LinearBuckets: &pb.MetricsData_Distribution_LinearOptions{
					NumFiniteBuckets: proto.Int32(20),
					Width:            proto.Float64(10),
					Offset:           proto.Float64(0),
				},
			},
		})
	})

	Convey("Exponential buckets", t, func() {
		d := distribution.New(distribution.GeometricBucketer(2, 20))
		d.Add(0)
		d.Add(4)
		d.Add(1024)
		d.Add(1536)
		d.Add(1048576)

		dpb := serializeDistribution(d)
		So(dpb, ShouldResembleProto, &pb.MetricsData_Distribution{
			Count: proto.Int64(5),
			Mean:  proto.Float64(210228),
			BucketOptions: &pb.MetricsData_Distribution_ExponentialBuckets{
				ExponentialBuckets: &pb.MetricsData_Distribution_ExponentialOptions{
					NumFiniteBuckets: proto.Int32(20),
					GrowthFactor:     proto.Float64(2),
					Scale:            proto.Float64(1),
				},
			},
			BucketCount: []int64{1, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 2, 0, 0, 0, 0,
				0, 0, 0, 0, 0, 1},
		})
	})

	Convey("Linear buckets", t, func() {
		d := distribution.New(distribution.FixedWidthBucketer(10, 2))
		d.Add(0)
		d.Add(1)
		d.Add(2)
		d.Add(20)

		dpb := serializeDistribution(d)
		So(dpb, ShouldResembleProto, &pb.MetricsData_Distribution{
			Count: proto.Int64(4),
			Mean:  proto.Float64(5.75),
			BucketOptions: &pb.MetricsData_Distribution_LinearBuckets{
				LinearBuckets: &pb.MetricsData_Distribution_LinearOptions{
					NumFiniteBuckets: proto.Int32(2),
					Width:            proto.Float64(10),
					Offset:           proto.Float64(0),
				},
			},
			BucketCount: []int64{0, 3, 0, 1},
		})
	})
}

func TestSerializeCell(t *testing.T) {
	now := time.Date(2001, 1, 2, 3, 4, 5, 6, time.UTC)
	reset := time.Date(2000, 1, 2, 3, 4, 5, 6, time.UTC)

	nowTS := timestamppb.New(now)
	resetTS := timestamppb.New(reset)

	emptyTask := &pb.MetricsCollection_Task{&pb.Task{
		ServiceName: proto.String(""),
		JobName:     proto.String(""),
		DataCenter:  proto.String(""),
		HostName:    proto.String(""),
		TaskNum:     proto.Int(0),
	}}

	Convey("Int", t, func() {
		ret := SerializeCells([]types.Cell{{
			types.MetricInfo{
				Name:        "foo",
				Description: "bar",
				Fields:      []field.Field{},
				ValueType:   types.NonCumulativeIntType,
			},
			types.MetricMetadata{},
			types.CellData{
				FieldVals: []interface{}{},
				Target:    &target.Task{},
				ResetTime: reset,
				Value:     int64(42),
			},
		}}, now)
		So(ret, ShouldResemble, []*pb.MetricsCollection{{
			TargetSchema: emptyTask,
			MetricsDataSet: []*pb.MetricsDataSet{{
				MetricName:      proto.String("/chrome/infra/foo"),
				FieldDescriptor: []*pb.MetricsDataSet_MetricFieldDescriptor{},
				StreamKind:      pb.StreamKind_GAUGE.Enum(),
				ValueType:       pb.ValueType_INT64.Enum(),
				Description:     proto.String("bar"),
				Data: []*pb.MetricsData{{
					Value:          &pb.MetricsData_Int64Value{42},
					Field:          []*pb.MetricsData_MetricField{},
					StartTimestamp: nowTS,
					EndTimestamp:   nowTS,
				}},
			}},
		}})
	})

	Convey("Counter", t, func() {
		ret := SerializeCells([]types.Cell{{
			types.MetricInfo{
				Name:        "foo",
				Description: "bar",
				Fields:      []field.Field{},
				ValueType:   types.CumulativeIntType,
			},
			types.MetricMetadata{},
			types.CellData{
				FieldVals: []interface{}{},
				Target:    &target.Task{},
				ResetTime: reset,
				Value:     int64(42),
			},
		}}, now)
		So(ret, ShouldResemble, []*pb.MetricsCollection{{
			TargetSchema: emptyTask,
			MetricsDataSet: []*pb.MetricsDataSet{{
				MetricName:      proto.String("/chrome/infra/foo"),
				FieldDescriptor: []*pb.MetricsDataSet_MetricFieldDescriptor{},
				StreamKind:      pb.StreamKind_CUMULATIVE.Enum(),
				ValueType:       pb.ValueType_INT64.Enum(),
				Description:     proto.String("bar"),
				Data: []*pb.MetricsData{{
					Value:          &pb.MetricsData_Int64Value{42},
					Field:          []*pb.MetricsData_MetricField{},
					StartTimestamp: resetTS,
					EndTimestamp:   nowTS,
				}},
			}},
		}})
	})

	Convey("Float", t, func() {
		ret := SerializeCells([]types.Cell{{
			types.MetricInfo{
				Name:        "foo",
				Description: "bar",
				Fields:      []field.Field{},
				ValueType:   types.NonCumulativeFloatType,
			},
			types.MetricMetadata{},
			types.CellData{
				FieldVals: []interface{}{},
				Target:    &target.Task{},
				ResetTime: reset,
				Value:     float64(42),
			},
		}}, now)
		So(ret, ShouldResemble, []*pb.MetricsCollection{{
			TargetSchema: emptyTask,
			MetricsDataSet: []*pb.MetricsDataSet{{
				MetricName:      proto.String("/chrome/infra/foo"),
				FieldDescriptor: []*pb.MetricsDataSet_MetricFieldDescriptor{},
				StreamKind:      pb.StreamKind_GAUGE.Enum(),
				ValueType:       pb.ValueType_DOUBLE.Enum(),
				Description:     proto.String("bar"),
				Data: []*pb.MetricsData{{
					Value:          &pb.MetricsData_DoubleValue{42},
					Field:          []*pb.MetricsData_MetricField{},
					StartTimestamp: nowTS,
					EndTimestamp:   nowTS,
				}},
			}},
		}})
	})

	Convey("FloatCounter", t, func() {
		ret := SerializeCells([]types.Cell{{
			types.MetricInfo{
				Name:        "foo",
				Description: "bar",
				Fields:      []field.Field{},
				ValueType:   types.CumulativeFloatType,
			},
			types.MetricMetadata{},
			types.CellData{
				FieldVals: []interface{}{},
				Target:    &target.Task{},
				ResetTime: reset,
				Value:     float64(42),
			},
		}}, now)
		So(ret, ShouldResemble, []*pb.MetricsCollection{{
			TargetSchema: emptyTask,
			MetricsDataSet: []*pb.MetricsDataSet{{
				MetricName:      proto.String("/chrome/infra/foo"),
				FieldDescriptor: []*pb.MetricsDataSet_MetricFieldDescriptor{},
				StreamKind:      pb.StreamKind_CUMULATIVE.Enum(),
				ValueType:       pb.ValueType_DOUBLE.Enum(),
				Description:     proto.String("bar"),
				Data: []*pb.MetricsData{{
					Value:          &pb.MetricsData_DoubleValue{42},
					Field:          []*pb.MetricsData_MetricField{},
					StartTimestamp: resetTS,
					EndTimestamp:   nowTS,
				}},
			}},
		}})
	})

	Convey("String", t, func() {
		ret := SerializeCells([]types.Cell{{
			types.MetricInfo{
				Name:        "foo",
				Description: "bar",
				Fields:      []field.Field{},
				ValueType:   types.StringType,
			},
			types.MetricMetadata{},
			types.CellData{
				FieldVals: []interface{}{},
				Target:    &target.Task{},
				ResetTime: reset,
				Value:     "hello",
			},
		}}, now)
		So(ret, ShouldResemble, []*pb.MetricsCollection{{
			TargetSchema: emptyTask,
			MetricsDataSet: []*pb.MetricsDataSet{{
				MetricName:      proto.String("/chrome/infra/foo"),
				FieldDescriptor: []*pb.MetricsDataSet_MetricFieldDescriptor{},
				StreamKind:      pb.StreamKind_GAUGE.Enum(),
				ValueType:       pb.ValueType_STRING.Enum(),
				Description:     proto.String("bar"),
				Data: []*pb.MetricsData{{
					Value:          &pb.MetricsData_StringValue{"hello"},
					Field:          []*pb.MetricsData_MetricField{},
					StartTimestamp: nowTS,
					EndTimestamp:   nowTS,
				}},
			}},
		}})
	})

	Convey("Boolean", t, func() {
		ret := SerializeCells([]types.Cell{{
			types.MetricInfo{
				Name:        "foo",
				Description: "bar",
				Fields:      []field.Field{},
				ValueType:   types.BoolType,
			},
			types.MetricMetadata{},
			types.CellData{
				FieldVals: []interface{}{},
				Target:    &target.Task{},
				ResetTime: reset,
				Value:     true,
			},
		}}, now)
		So(ret, ShouldResemble, []*pb.MetricsCollection{{
			TargetSchema: emptyTask,
			MetricsDataSet: []*pb.MetricsDataSet{{
				MetricName:      proto.String("/chrome/infra/foo"),
				FieldDescriptor: []*pb.MetricsDataSet_MetricFieldDescriptor{},
				StreamKind:      pb.StreamKind_GAUGE.Enum(),
				ValueType:       pb.ValueType_BOOL.Enum(),
				Description:     proto.String("bar"),
				Data: []*pb.MetricsData{{
					Value:          &pb.MetricsData_BoolValue{true},
					Field:          []*pb.MetricsData_MetricField{},
					StartTimestamp: nowTS,
					EndTimestamp:   nowTS,
				}},
			}},
		}})
	})

	Convey("NonDefaultTarget", t, func() {
		target := target.Task{
			ServiceName: "hello",
			JobName:     "world",
		}

		ret := SerializeCells([]types.Cell{{
			types.MetricInfo{
				Name:        "foo",
				Description: "bar",
				Fields:      []field.Field{},
				ValueType:   types.NonCumulativeIntType,
			},
			types.MetricMetadata{},
			types.CellData{
				FieldVals: []interface{}{},
				Target:    &target,
				ResetTime: reset,
				Value:     int64(42),
			},
		}}, now)
		So(ret, ShouldResemble, []*pb.MetricsCollection{{
			TargetSchema: &pb.MetricsCollection_Task{&pb.Task{
				ServiceName: proto.String("hello"),
				JobName:     proto.String("world"),
				DataCenter:  proto.String(""),
				HostName:    proto.String(""),
				TaskNum:     proto.Int(0),
			}},
			MetricsDataSet: []*pb.MetricsDataSet{{
				MetricName:      proto.String("/chrome/infra/foo"),
				FieldDescriptor: []*pb.MetricsDataSet_MetricFieldDescriptor{},
				StreamKind:      pb.StreamKind_GAUGE.Enum(),
				ValueType:       pb.ValueType_INT64.Enum(),
				Description:     proto.String("bar"),
				Data: []*pb.MetricsData{{
					Value:          &pb.MetricsData_Int64Value{42},
					Field:          []*pb.MetricsData_MetricField{},
					StartTimestamp: nowTS,
					EndTimestamp:   nowTS,
				}},
			}},
		}})
	})
}
