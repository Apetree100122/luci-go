// Copyright 2022 The LUCI Authors.
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

// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.31.0
// 	protoc        v3.21.7
// source: go.chromium.org/luci/analysis/internal/tasks/taskspb/tasks.proto

package taskspb

import (
	proto "go.chromium.org/luci/analysis/internal/ingestion/control/proto"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

// Payload of IngestTestResults task.
type IngestTestResults struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Timestamp representing the start of the data retention period
	// for the ingested test results. In case of multiple builds
	// ingested for one CV run, the partition_time used for all
	// builds must be the same.
	PartitionTime *timestamppb.Timestamp `protobuf:"bytes,3,opt,name=partition_time,json=partitionTime,proto3" json:"partition_time,omitempty"`
	// The build that is being ingested.
	Build *proto.BuildResult `protobuf:"bytes,8,opt,name=build,proto3" json:"build,omitempty"`
	// Context about the presubmit run the build was a part of. Only
	// populated if the build is a presubmit run.
	PresubmitRun *proto.PresubmitResult `protobuf:"bytes,9,opt,name=presubmit_run,json=presubmitRun,proto3" json:"presubmit_run,omitempty"`
	// The page token value to use when calling QueryTestVariants.
	// For the first task, this should be "". For subsequent tasks,
	// this is the next_page_token value returned by the last call.
	PageToken string `protobuf:"bytes,10,opt,name=page_token,json=pageToken,proto3" json:"page_token,omitempty"`
	// The task number of test results task. 0 for the first
	// task, 1 for the second task, and so on. Used to avoid creating
	// duplicate tasks.
	TaskIndex int64 `protobuf:"varint,11,opt,name=task_index,json=taskIndex,proto3" json:"task_index,omitempty"`
}

func (x *IngestTestResults) Reset() {
	*x = IngestTestResults{}
	if protoimpl.UnsafeEnabled {
		mi := &file_go_chromium_org_luci_analysis_internal_tasks_taskspb_tasks_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *IngestTestResults) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*IngestTestResults) ProtoMessage() {}

func (x *IngestTestResults) ProtoReflect() protoreflect.Message {
	mi := &file_go_chromium_org_luci_analysis_internal_tasks_taskspb_tasks_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use IngestTestResults.ProtoReflect.Descriptor instead.
func (*IngestTestResults) Descriptor() ([]byte, []int) {
	return file_go_chromium_org_luci_analysis_internal_tasks_taskspb_tasks_proto_rawDescGZIP(), []int{0}
}

func (x *IngestTestResults) GetPartitionTime() *timestamppb.Timestamp {
	if x != nil {
		return x.PartitionTime
	}
	return nil
}

func (x *IngestTestResults) GetBuild() *proto.BuildResult {
	if x != nil {
		return x.Build
	}
	return nil
}

func (x *IngestTestResults) GetPresubmitRun() *proto.PresubmitResult {
	if x != nil {
		return x.PresubmitRun
	}
	return nil
}

func (x *IngestTestResults) GetPageToken() string {
	if x != nil {
		return x.PageToken
	}
	return ""
}

func (x *IngestTestResults) GetTaskIndex() int64 {
	if x != nil {
		return x.TaskIndex
	}
	return 0
}

// Payload of the ReclusterChunks task.
type ReclusterChunks struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// The number of the reclustering shard being processed by this task.
	// A shard corresponds to a project + Chunk ID keyspace fraction that
	// is being re-clustered.
	// Shards are numbered sequentially, starting at one.
	ShardNumber int64 `protobuf:"varint,6,opt,name=shard_number,json=shardNumber,proto3" json:"shard_number,omitempty"`
	// The LUCI Project containing test results to be re-clustered.
	Project string `protobuf:"bytes,1,opt,name=project,proto3" json:"project,omitempty"`
	// The attempt time for which this task is. This should be cross-referenced
	// with the ReclusteringRuns table to identify the reclustering parameters.
	// This is also the soft deadline for the task.
	AttemptTime *timestamppb.Timestamp `protobuf:"bytes,2,opt,name=attempt_time,json=attemptTime,proto3" json:"attempt_time,omitempty"`
	// The exclusive lower bound defining the range of Chunk IDs to
	// be re-clustered. To define the table start, use the empty string ("").
	StartChunkId string `protobuf:"bytes,3,opt,name=start_chunk_id,json=startChunkId,proto3" json:"start_chunk_id,omitempty"`
	// The inclusive upper bound defining the range of Chunk IDs to
	// be re-clustered. To define the table end use "ff" x 16, i.e.
	// "ffffffffffffffffffffffffffffffff".
	EndChunkId string `protobuf:"bytes,4,opt,name=end_chunk_id,json=endChunkId,proto3" json:"end_chunk_id,omitempty"`
	// The version of algorithms to re-cluster to. If the worker executing the
	// task is not running at least this version of algorithms, it is an error.
	AlgorithmsVersion int64 `protobuf:"varint,7,opt,name=algorithms_version,json=algorithmsVersion,proto3" json:"algorithms_version,omitempty"`
	// The version of rules to recluster to.
	RulesVersion *timestamppb.Timestamp `protobuf:"bytes,8,opt,name=rules_version,json=rulesVersion,proto3" json:"rules_version,omitempty"`
	// The version of project configuration to recluster to.
	ConfigVersion *timestamppb.Timestamp `protobuf:"bytes,9,opt,name=config_version,json=configVersion,proto3" json:"config_version,omitempty"`
	// State to be passed from one execution of the task to the next.
	// To fit with autoscaling, each task aims to execute only for a short time
	// before enqueuing another task to act as its continuation.
	// Must be populated on all tasks, even on the initial task.
	State *ReclusterChunkState `protobuf:"bytes,5,opt,name=state,proto3" json:"state,omitempty"`
}

func (x *ReclusterChunks) Reset() {
	*x = ReclusterChunks{}
	if protoimpl.UnsafeEnabled {
		mi := &file_go_chromium_org_luci_analysis_internal_tasks_taskspb_tasks_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ReclusterChunks) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ReclusterChunks) ProtoMessage() {}

func (x *ReclusterChunks) ProtoReflect() protoreflect.Message {
	mi := &file_go_chromium_org_luci_analysis_internal_tasks_taskspb_tasks_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ReclusterChunks.ProtoReflect.Descriptor instead.
func (*ReclusterChunks) Descriptor() ([]byte, []int) {
	return file_go_chromium_org_luci_analysis_internal_tasks_taskspb_tasks_proto_rawDescGZIP(), []int{1}
}

func (x *ReclusterChunks) GetShardNumber() int64 {
	if x != nil {
		return x.ShardNumber
	}
	return 0
}

func (x *ReclusterChunks) GetProject() string {
	if x != nil {
		return x.Project
	}
	return ""
}

func (x *ReclusterChunks) GetAttemptTime() *timestamppb.Timestamp {
	if x != nil {
		return x.AttemptTime
	}
	return nil
}

func (x *ReclusterChunks) GetStartChunkId() string {
	if x != nil {
		return x.StartChunkId
	}
	return ""
}

func (x *ReclusterChunks) GetEndChunkId() string {
	if x != nil {
		return x.EndChunkId
	}
	return ""
}

func (x *ReclusterChunks) GetAlgorithmsVersion() int64 {
	if x != nil {
		return x.AlgorithmsVersion
	}
	return 0
}

func (x *ReclusterChunks) GetRulesVersion() *timestamppb.Timestamp {
	if x != nil {
		return x.RulesVersion
	}
	return nil
}

func (x *ReclusterChunks) GetConfigVersion() *timestamppb.Timestamp {
	if x != nil {
		return x.ConfigVersion
	}
	return nil
}

func (x *ReclusterChunks) GetState() *ReclusterChunkState {
	if x != nil {
		return x.State
	}
	return nil
}

// ReclusterChunkState captures state passed from one execution of a
// ReclusterChunks task to the next.
type ReclusterChunkState struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// The exclusive lower bound of Chunk IDs processed to date.
	CurrentChunkId string `protobuf:"bytes,1,opt,name=current_chunk_id,json=currentChunkId,proto3" json:"current_chunk_id,omitempty"`
	// The next time a progress report should be made.
	NextReportDue *timestamppb.Timestamp `protobuf:"bytes,2,opt,name=next_report_due,json=nextReportDue,proto3" json:"next_report_due,omitempty"`
}

func (x *ReclusterChunkState) Reset() {
	*x = ReclusterChunkState{}
	if protoimpl.UnsafeEnabled {
		mi := &file_go_chromium_org_luci_analysis_internal_tasks_taskspb_tasks_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ReclusterChunkState) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ReclusterChunkState) ProtoMessage() {}

func (x *ReclusterChunkState) ProtoReflect() protoreflect.Message {
	mi := &file_go_chromium_org_luci_analysis_internal_tasks_taskspb_tasks_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ReclusterChunkState.ProtoReflect.Descriptor instead.
func (*ReclusterChunkState) Descriptor() ([]byte, []int) {
	return file_go_chromium_org_luci_analysis_internal_tasks_taskspb_tasks_proto_rawDescGZIP(), []int{2}
}

func (x *ReclusterChunkState) GetCurrentChunkId() string {
	if x != nil {
		return x.CurrentChunkId
	}
	return ""
}

func (x *ReclusterChunkState) GetNextReportDue() *timestamppb.Timestamp {
	if x != nil {
		return x.NextReportDue
	}
	return nil
}

// Payload of the JoinBuild task.
type JoinBuild struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Buildbucket build ID, unique per Buildbucket instance.
	Id int64 `protobuf:"varint,1,opt,name=id,proto3" json:"id,omitempty"`
	// Buildbucket host, e.g. "cr-buildbucket.appspot.com".
	Host string `protobuf:"bytes,2,opt,name=host,proto3" json:"host,omitempty"`
	// The LUCI Project to which the build belongs.
	Project string `protobuf:"bytes,3,opt,name=project,proto3" json:"project,omitempty"`
}

func (x *JoinBuild) Reset() {
	*x = JoinBuild{}
	if protoimpl.UnsafeEnabled {
		mi := &file_go_chromium_org_luci_analysis_internal_tasks_taskspb_tasks_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *JoinBuild) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*JoinBuild) ProtoMessage() {}

func (x *JoinBuild) ProtoReflect() protoreflect.Message {
	mi := &file_go_chromium_org_luci_analysis_internal_tasks_taskspb_tasks_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use JoinBuild.ProtoReflect.Descriptor instead.
func (*JoinBuild) Descriptor() ([]byte, []int) {
	return file_go_chromium_org_luci_analysis_internal_tasks_taskspb_tasks_proto_rawDescGZIP(), []int{3}
}

func (x *JoinBuild) GetId() int64 {
	if x != nil {
		return x.Id
	}
	return 0
}

func (x *JoinBuild) GetHost() string {
	if x != nil {
		return x.Host
	}
	return ""
}

func (x *JoinBuild) GetProject() string {
	if x != nil {
		return x.Project
	}
	return ""
}

// Payload of the UpdateBugs task. Prior to running this task,
// the cluster_summaries table should have been updated from the
// contents of the clustered_failures table.
type UpdateBugs struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// The LUCI Project to update bugs for.
	Project string `protobuf:"bytes,1,opt,name=project,proto3" json:"project,omitempty"`
	// The reclustering attempt minute that reflects the reclustering
	// state of the failures summarized by the cluster_summaries table.
	//
	// Explanation:
	// Bug management relies upon knowing when reclustering
	// is ongoing for rules and algorithms to inhibit erroneous bug updates
	// for those rules / algorithms as cluster metrics may be invalid.
	//
	// The re-clustering progress tracked in ReclusteringRuns table tracks
	// the progress applying re-clustering to the clustered_failures
	// table (not the cluster_summaries table).
	// As there is a delay between when clustered_failures table
	// is updated and when cluster_summaries is updated, we cannot
	// use the latest reclustering run but need to read the run
	// that was current when the clustered_failures table was
	// summarized into the cluster_summaries table.
	//
	// This will be the run that was current when the BigQuery
	// job to recompute cluster_summaries table from the
	// clustered_failures table started.
	ReclusteringAttemptMinute *timestamppb.Timestamp `protobuf:"bytes,2,opt,name=reclustering_attempt_minute,json=reclusteringAttemptMinute,proto3" json:"reclustering_attempt_minute,omitempty"`
	// The time the task should be completed by to avoid overruning
	// the next bug update task.
	Deadline *timestamppb.Timestamp `protobuf:"bytes,3,opt,name=deadline,proto3" json:"deadline,omitempty"`
}

func (x *UpdateBugs) Reset() {
	*x = UpdateBugs{}
	if protoimpl.UnsafeEnabled {
		mi := &file_go_chromium_org_luci_analysis_internal_tasks_taskspb_tasks_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *UpdateBugs) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*UpdateBugs) ProtoMessage() {}

func (x *UpdateBugs) ProtoReflect() protoreflect.Message {
	mi := &file_go_chromium_org_luci_analysis_internal_tasks_taskspb_tasks_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use UpdateBugs.ProtoReflect.Descriptor instead.
func (*UpdateBugs) Descriptor() ([]byte, []int) {
	return file_go_chromium_org_luci_analysis_internal_tasks_taskspb_tasks_proto_rawDescGZIP(), []int{4}
}

func (x *UpdateBugs) GetProject() string {
	if x != nil {
		return x.Project
	}
	return ""
}

func (x *UpdateBugs) GetReclusteringAttemptMinute() *timestamppb.Timestamp {
	if x != nil {
		return x.ReclusteringAttemptMinute
	}
	return nil
}

func (x *UpdateBugs) GetDeadline() *timestamppb.Timestamp {
	if x != nil {
		return x.Deadline
	}
	return nil
}

var File_go_chromium_org_luci_analysis_internal_tasks_taskspb_tasks_proto protoreflect.FileDescriptor

var file_go_chromium_org_luci_analysis_internal_tasks_taskspb_tasks_proto_rawDesc = []byte{
	0x0a, 0x40, 0x67, 0x6f, 0x2e, 0x63, 0x68, 0x72, 0x6f, 0x6d, 0x69, 0x75, 0x6d, 0x2e, 0x6f, 0x72,
	0x67, 0x2f, 0x6c, 0x75, 0x63, 0x69, 0x2f, 0x61, 0x6e, 0x61, 0x6c, 0x79, 0x73, 0x69, 0x73, 0x2f,
	0x69, 0x6e, 0x74, 0x65, 0x72, 0x6e, 0x61, 0x6c, 0x2f, 0x74, 0x61, 0x73, 0x6b, 0x73, 0x2f, 0x74,
	0x61, 0x73, 0x6b, 0x73, 0x70, 0x62, 0x2f, 0x74, 0x61, 0x73, 0x6b, 0x73, 0x2e, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x12, 0x1c, 0x6c, 0x75, 0x63, 0x69, 0x2e, 0x61, 0x6e, 0x61, 0x6c, 0x79, 0x73, 0x69,
	0x73, 0x2e, 0x69, 0x6e, 0x74, 0x65, 0x72, 0x6e, 0x61, 0x6c, 0x2e, 0x74, 0x61, 0x73, 0x6b, 0x73,
	0x1a, 0x1f, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75,
	0x66, 0x2f, 0x74, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x2e, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x1a, 0x4c, 0x67, 0x6f, 0x2e, 0x63, 0x68, 0x72, 0x6f, 0x6d, 0x69, 0x75, 0x6d, 0x2e, 0x6f,
	0x72, 0x67, 0x2f, 0x6c, 0x75, 0x63, 0x69, 0x2f, 0x61, 0x6e, 0x61, 0x6c, 0x79, 0x73, 0x69, 0x73,
	0x2f, 0x69, 0x6e, 0x74, 0x65, 0x72, 0x6e, 0x61, 0x6c, 0x2f, 0x69, 0x6e, 0x67, 0x65, 0x73, 0x74,
	0x69, 0x6f, 0x6e, 0x2f, 0x63, 0x6f, 0x6e, 0x74, 0x72, 0x6f, 0x6c, 0x2f, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x2f, 0x63, 0x6f, 0x6e, 0x74, 0x72, 0x6f, 0x6c, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22,
	0xd3, 0x02, 0x0a, 0x11, 0x49, 0x6e, 0x67, 0x65, 0x73, 0x74, 0x54, 0x65, 0x73, 0x74, 0x52, 0x65,
	0x73, 0x75, 0x6c, 0x74, 0x73, 0x12, 0x41, 0x0a, 0x0e, 0x70, 0x61, 0x72, 0x74, 0x69, 0x74, 0x69,
	0x6f, 0x6e, 0x5f, 0x74, 0x69, 0x6d, 0x65, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1a, 0x2e,
	0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e,
	0x54, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x52, 0x0d, 0x70, 0x61, 0x72, 0x74, 0x69,
	0x74, 0x69, 0x6f, 0x6e, 0x54, 0x69, 0x6d, 0x65, 0x12, 0x4b, 0x0a, 0x05, 0x62, 0x75, 0x69, 0x6c,
	0x64, 0x18, 0x08, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x35, 0x2e, 0x6c, 0x75, 0x63, 0x69, 0x2e, 0x61,
	0x6e, 0x61, 0x6c, 0x79, 0x73, 0x69, 0x73, 0x2e, 0x69, 0x6e, 0x74, 0x65, 0x72, 0x6e, 0x61, 0x6c,
	0x2e, 0x69, 0x6e, 0x67, 0x65, 0x73, 0x74, 0x69, 0x6f, 0x6e, 0x2e, 0x63, 0x6f, 0x6e, 0x74, 0x72,
	0x6f, 0x6c, 0x2e, 0x42, 0x75, 0x69, 0x6c, 0x64, 0x52, 0x65, 0x73, 0x75, 0x6c, 0x74, 0x52, 0x05,
	0x62, 0x75, 0x69, 0x6c, 0x64, 0x12, 0x5e, 0x0a, 0x0d, 0x70, 0x72, 0x65, 0x73, 0x75, 0x62, 0x6d,
	0x69, 0x74, 0x5f, 0x72, 0x75, 0x6e, 0x18, 0x09, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x39, 0x2e, 0x6c,
	0x75, 0x63, 0x69, 0x2e, 0x61, 0x6e, 0x61, 0x6c, 0x79, 0x73, 0x69, 0x73, 0x2e, 0x69, 0x6e, 0x74,
	0x65, 0x72, 0x6e, 0x61, 0x6c, 0x2e, 0x69, 0x6e, 0x67, 0x65, 0x73, 0x74, 0x69, 0x6f, 0x6e, 0x2e,
	0x63, 0x6f, 0x6e, 0x74, 0x72, 0x6f, 0x6c, 0x2e, 0x50, 0x72, 0x65, 0x73, 0x75, 0x62, 0x6d, 0x69,
	0x74, 0x52, 0x65, 0x73, 0x75, 0x6c, 0x74, 0x52, 0x0c, 0x70, 0x72, 0x65, 0x73, 0x75, 0x62, 0x6d,
	0x69, 0x74, 0x52, 0x75, 0x6e, 0x12, 0x1d, 0x0a, 0x0a, 0x70, 0x61, 0x67, 0x65, 0x5f, 0x74, 0x6f,
	0x6b, 0x65, 0x6e, 0x18, 0x0a, 0x20, 0x01, 0x28, 0x09, 0x52, 0x09, 0x70, 0x61, 0x67, 0x65, 0x54,
	0x6f, 0x6b, 0x65, 0x6e, 0x12, 0x1d, 0x0a, 0x0a, 0x74, 0x61, 0x73, 0x6b, 0x5f, 0x69, 0x6e, 0x64,
	0x65, 0x78, 0x18, 0x0b, 0x20, 0x01, 0x28, 0x03, 0x52, 0x09, 0x74, 0x61, 0x73, 0x6b, 0x49, 0x6e,
	0x64, 0x65, 0x78, 0x4a, 0x04, 0x08, 0x01, 0x10, 0x02, 0x4a, 0x04, 0x08, 0x02, 0x10, 0x03, 0x4a,
	0x04, 0x08, 0x04, 0x10, 0x08, 0x22, 0xd1, 0x03, 0x0a, 0x0f, 0x52, 0x65, 0x63, 0x6c, 0x75, 0x73,
	0x74, 0x65, 0x72, 0x43, 0x68, 0x75, 0x6e, 0x6b, 0x73, 0x12, 0x21, 0x0a, 0x0c, 0x73, 0x68, 0x61,
	0x72, 0x64, 0x5f, 0x6e, 0x75, 0x6d, 0x62, 0x65, 0x72, 0x18, 0x06, 0x20, 0x01, 0x28, 0x03, 0x52,
	0x0b, 0x73, 0x68, 0x61, 0x72, 0x64, 0x4e, 0x75, 0x6d, 0x62, 0x65, 0x72, 0x12, 0x18, 0x0a, 0x07,
	0x70, 0x72, 0x6f, 0x6a, 0x65, 0x63, 0x74, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x07, 0x70,
	0x72, 0x6f, 0x6a, 0x65, 0x63, 0x74, 0x12, 0x3d, 0x0a, 0x0c, 0x61, 0x74, 0x74, 0x65, 0x6d, 0x70,
	0x74, 0x5f, 0x74, 0x69, 0x6d, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1a, 0x2e, 0x67,
	0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x54,
	0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x52, 0x0b, 0x61, 0x74, 0x74, 0x65, 0x6d, 0x70,
	0x74, 0x54, 0x69, 0x6d, 0x65, 0x12, 0x24, 0x0a, 0x0e, 0x73, 0x74, 0x61, 0x72, 0x74, 0x5f, 0x63,
	0x68, 0x75, 0x6e, 0x6b, 0x5f, 0x69, 0x64, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0c, 0x73,
	0x74, 0x61, 0x72, 0x74, 0x43, 0x68, 0x75, 0x6e, 0x6b, 0x49, 0x64, 0x12, 0x20, 0x0a, 0x0c, 0x65,
	0x6e, 0x64, 0x5f, 0x63, 0x68, 0x75, 0x6e, 0x6b, 0x5f, 0x69, 0x64, 0x18, 0x04, 0x20, 0x01, 0x28,
	0x09, 0x52, 0x0a, 0x65, 0x6e, 0x64, 0x43, 0x68, 0x75, 0x6e, 0x6b, 0x49, 0x64, 0x12, 0x2d, 0x0a,
	0x12, 0x61, 0x6c, 0x67, 0x6f, 0x72, 0x69, 0x74, 0x68, 0x6d, 0x73, 0x5f, 0x76, 0x65, 0x72, 0x73,
	0x69, 0x6f, 0x6e, 0x18, 0x07, 0x20, 0x01, 0x28, 0x03, 0x52, 0x11, 0x61, 0x6c, 0x67, 0x6f, 0x72,
	0x69, 0x74, 0x68, 0x6d, 0x73, 0x56, 0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e, 0x12, 0x3f, 0x0a, 0x0d,
	0x72, 0x75, 0x6c, 0x65, 0x73, 0x5f, 0x76, 0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e, 0x18, 0x08, 0x20,
	0x01, 0x28, 0x0b, 0x32, 0x1a, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x54, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x52,
	0x0c, 0x72, 0x75, 0x6c, 0x65, 0x73, 0x56, 0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e, 0x12, 0x41, 0x0a,
	0x0e, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x5f, 0x76, 0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e, 0x18,
	0x09, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1a, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x54, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d,
	0x70, 0x52, 0x0d, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x56, 0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e,
	0x12, 0x47, 0x0a, 0x05, 0x73, 0x74, 0x61, 0x74, 0x65, 0x18, 0x05, 0x20, 0x01, 0x28, 0x0b, 0x32,
	0x31, 0x2e, 0x6c, 0x75, 0x63, 0x69, 0x2e, 0x61, 0x6e, 0x61, 0x6c, 0x79, 0x73, 0x69, 0x73, 0x2e,
	0x69, 0x6e, 0x74, 0x65, 0x72, 0x6e, 0x61, 0x6c, 0x2e, 0x74, 0x61, 0x73, 0x6b, 0x73, 0x2e, 0x52,
	0x65, 0x63, 0x6c, 0x75, 0x73, 0x74, 0x65, 0x72, 0x43, 0x68, 0x75, 0x6e, 0x6b, 0x53, 0x74, 0x61,
	0x74, 0x65, 0x52, 0x05, 0x73, 0x74, 0x61, 0x74, 0x65, 0x22, 0x83, 0x01, 0x0a, 0x13, 0x52, 0x65,
	0x63, 0x6c, 0x75, 0x73, 0x74, 0x65, 0x72, 0x43, 0x68, 0x75, 0x6e, 0x6b, 0x53, 0x74, 0x61, 0x74,
	0x65, 0x12, 0x28, 0x0a, 0x10, 0x63, 0x75, 0x72, 0x72, 0x65, 0x6e, 0x74, 0x5f, 0x63, 0x68, 0x75,
	0x6e, 0x6b, 0x5f, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0e, 0x63, 0x75, 0x72,
	0x72, 0x65, 0x6e, 0x74, 0x43, 0x68, 0x75, 0x6e, 0x6b, 0x49, 0x64, 0x12, 0x42, 0x0a, 0x0f, 0x6e,
	0x65, 0x78, 0x74, 0x5f, 0x72, 0x65, 0x70, 0x6f, 0x72, 0x74, 0x5f, 0x64, 0x75, 0x65, 0x18, 0x02,
	0x20, 0x01, 0x28, 0x0b, 0x32, 0x1a, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x54, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70,
	0x52, 0x0d, 0x6e, 0x65, 0x78, 0x74, 0x52, 0x65, 0x70, 0x6f, 0x72, 0x74, 0x44, 0x75, 0x65, 0x22,
	0x49, 0x0a, 0x09, 0x4a, 0x6f, 0x69, 0x6e, 0x42, 0x75, 0x69, 0x6c, 0x64, 0x12, 0x0e, 0x0a, 0x02,
	0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x03, 0x52, 0x02, 0x69, 0x64, 0x12, 0x12, 0x0a, 0x04,
	0x68, 0x6f, 0x73, 0x74, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x68, 0x6f, 0x73, 0x74,
	0x12, 0x18, 0x0a, 0x07, 0x70, 0x72, 0x6f, 0x6a, 0x65, 0x63, 0x74, 0x18, 0x03, 0x20, 0x01, 0x28,
	0x09, 0x52, 0x07, 0x70, 0x72, 0x6f, 0x6a, 0x65, 0x63, 0x74, 0x22, 0xba, 0x01, 0x0a, 0x0a, 0x55,
	0x70, 0x64, 0x61, 0x74, 0x65, 0x42, 0x75, 0x67, 0x73, 0x12, 0x18, 0x0a, 0x07, 0x70, 0x72, 0x6f,
	0x6a, 0x65, 0x63, 0x74, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x07, 0x70, 0x72, 0x6f, 0x6a,
	0x65, 0x63, 0x74, 0x12, 0x5a, 0x0a, 0x1b, 0x72, 0x65, 0x63, 0x6c, 0x75, 0x73, 0x74, 0x65, 0x72,
	0x69, 0x6e, 0x67, 0x5f, 0x61, 0x74, 0x74, 0x65, 0x6d, 0x70, 0x74, 0x5f, 0x6d, 0x69, 0x6e, 0x75,
	0x74, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1a, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c,
	0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x54, 0x69, 0x6d, 0x65, 0x73,
	0x74, 0x61, 0x6d, 0x70, 0x52, 0x19, 0x72, 0x65, 0x63, 0x6c, 0x75, 0x73, 0x74, 0x65, 0x72, 0x69,
	0x6e, 0x67, 0x41, 0x74, 0x74, 0x65, 0x6d, 0x70, 0x74, 0x4d, 0x69, 0x6e, 0x75, 0x74, 0x65, 0x12,
	0x36, 0x0a, 0x08, 0x64, 0x65, 0x61, 0x64, 0x6c, 0x69, 0x6e, 0x65, 0x18, 0x03, 0x20, 0x01, 0x28,
	0x0b, 0x32, 0x1a, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x62, 0x75, 0x66, 0x2e, 0x54, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x52, 0x08, 0x64,
	0x65, 0x61, 0x64, 0x6c, 0x69, 0x6e, 0x65, 0x42, 0x36, 0x5a, 0x34, 0x67, 0x6f, 0x2e, 0x63, 0x68,
	0x72, 0x6f, 0x6d, 0x69, 0x75, 0x6d, 0x2e, 0x6f, 0x72, 0x67, 0x2f, 0x6c, 0x75, 0x63, 0x69, 0x2f,
	0x61, 0x6e, 0x61, 0x6c, 0x79, 0x73, 0x69, 0x73, 0x2f, 0x69, 0x6e, 0x74, 0x65, 0x72, 0x6e, 0x61,
	0x6c, 0x2f, 0x74, 0x61, 0x73, 0x6b, 0x73, 0x2f, 0x74, 0x61, 0x73, 0x6b, 0x73, 0x70, 0x62, 0x62,
	0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_go_chromium_org_luci_analysis_internal_tasks_taskspb_tasks_proto_rawDescOnce sync.Once
	file_go_chromium_org_luci_analysis_internal_tasks_taskspb_tasks_proto_rawDescData = file_go_chromium_org_luci_analysis_internal_tasks_taskspb_tasks_proto_rawDesc
)

func file_go_chromium_org_luci_analysis_internal_tasks_taskspb_tasks_proto_rawDescGZIP() []byte {
	file_go_chromium_org_luci_analysis_internal_tasks_taskspb_tasks_proto_rawDescOnce.Do(func() {
		file_go_chromium_org_luci_analysis_internal_tasks_taskspb_tasks_proto_rawDescData = protoimpl.X.CompressGZIP(file_go_chromium_org_luci_analysis_internal_tasks_taskspb_tasks_proto_rawDescData)
	})
	return file_go_chromium_org_luci_analysis_internal_tasks_taskspb_tasks_proto_rawDescData
}

var file_go_chromium_org_luci_analysis_internal_tasks_taskspb_tasks_proto_msgTypes = make([]protoimpl.MessageInfo, 5)
var file_go_chromium_org_luci_analysis_internal_tasks_taskspb_tasks_proto_goTypes = []interface{}{
	(*IngestTestResults)(nil),     // 0: luci.analysis.internal.tasks.IngestTestResults
	(*ReclusterChunks)(nil),       // 1: luci.analysis.internal.tasks.ReclusterChunks
	(*ReclusterChunkState)(nil),   // 2: luci.analysis.internal.tasks.ReclusterChunkState
	(*JoinBuild)(nil),             // 3: luci.analysis.internal.tasks.JoinBuild
	(*UpdateBugs)(nil),            // 4: luci.analysis.internal.tasks.UpdateBugs
	(*timestamppb.Timestamp)(nil), // 5: google.protobuf.Timestamp
	(*proto.BuildResult)(nil),     // 6: luci.analysis.internal.ingestion.control.BuildResult
	(*proto.PresubmitResult)(nil), // 7: luci.analysis.internal.ingestion.control.PresubmitResult
}
var file_go_chromium_org_luci_analysis_internal_tasks_taskspb_tasks_proto_depIdxs = []int32{
	5,  // 0: luci.analysis.internal.tasks.IngestTestResults.partition_time:type_name -> google.protobuf.Timestamp
	6,  // 1: luci.analysis.internal.tasks.IngestTestResults.build:type_name -> luci.analysis.internal.ingestion.control.BuildResult
	7,  // 2: luci.analysis.internal.tasks.IngestTestResults.presubmit_run:type_name -> luci.analysis.internal.ingestion.control.PresubmitResult
	5,  // 3: luci.analysis.internal.tasks.ReclusterChunks.attempt_time:type_name -> google.protobuf.Timestamp
	5,  // 4: luci.analysis.internal.tasks.ReclusterChunks.rules_version:type_name -> google.protobuf.Timestamp
	5,  // 5: luci.analysis.internal.tasks.ReclusterChunks.config_version:type_name -> google.protobuf.Timestamp
	2,  // 6: luci.analysis.internal.tasks.ReclusterChunks.state:type_name -> luci.analysis.internal.tasks.ReclusterChunkState
	5,  // 7: luci.analysis.internal.tasks.ReclusterChunkState.next_report_due:type_name -> google.protobuf.Timestamp
	5,  // 8: luci.analysis.internal.tasks.UpdateBugs.reclustering_attempt_minute:type_name -> google.protobuf.Timestamp
	5,  // 9: luci.analysis.internal.tasks.UpdateBugs.deadline:type_name -> google.protobuf.Timestamp
	10, // [10:10] is the sub-list for method output_type
	10, // [10:10] is the sub-list for method input_type
	10, // [10:10] is the sub-list for extension type_name
	10, // [10:10] is the sub-list for extension extendee
	0,  // [0:10] is the sub-list for field type_name
}

func init() { file_go_chromium_org_luci_analysis_internal_tasks_taskspb_tasks_proto_init() }
func file_go_chromium_org_luci_analysis_internal_tasks_taskspb_tasks_proto_init() {
	if File_go_chromium_org_luci_analysis_internal_tasks_taskspb_tasks_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_go_chromium_org_luci_analysis_internal_tasks_taskspb_tasks_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*IngestTestResults); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_go_chromium_org_luci_analysis_internal_tasks_taskspb_tasks_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ReclusterChunks); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_go_chromium_org_luci_analysis_internal_tasks_taskspb_tasks_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ReclusterChunkState); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_go_chromium_org_luci_analysis_internal_tasks_taskspb_tasks_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*JoinBuild); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_go_chromium_org_luci_analysis_internal_tasks_taskspb_tasks_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*UpdateBugs); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_go_chromium_org_luci_analysis_internal_tasks_taskspb_tasks_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   5,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_go_chromium_org_luci_analysis_internal_tasks_taskspb_tasks_proto_goTypes,
		DependencyIndexes: file_go_chromium_org_luci_analysis_internal_tasks_taskspb_tasks_proto_depIdxs,
		MessageInfos:      file_go_chromium_org_luci_analysis_internal_tasks_taskspb_tasks_proto_msgTypes,
	}.Build()
	File_go_chromium_org_luci_analysis_internal_tasks_taskspb_tasks_proto = out.File
	file_go_chromium_org_luci_analysis_internal_tasks_taskspb_tasks_proto_rawDesc = nil
	file_go_chromium_org_luci_analysis_internal_tasks_taskspb_tasks_proto_goTypes = nil
	file_go_chromium_org_luci_analysis_internal_tasks_taskspb_tasks_proto_depIdxs = nil
}
