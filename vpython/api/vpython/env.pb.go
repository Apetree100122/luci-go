// Copyright 2017 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.31.0
// 	protoc        v3.21.7
// source: go.chromium.org/luci/vpython/api/vpython/env.proto

package vpython

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

// Environment describes a constructed VirtualEnv.
type Environment struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// A constructed VirtualEnv.
	Spec *Spec `protobuf:"bytes,1,opt,name=spec,proto3" json:"spec,omitempty"`
	// The resolved runtime parameters.
	Runtime *Runtime `protobuf:"bytes,2,opt,name=runtime,proto3" json:"runtime,omitempty"`
	// The PEP425 tags that were probed for this Python environment.
	Pep425Tag []*PEP425Tag `protobuf:"bytes,3,rep,name=pep425_tag,json=pep425Tag,proto3" json:"pep425_tag,omitempty"`
}

func (x *Environment) Reset() {
	*x = Environment{}
	if protoimpl.UnsafeEnabled {
		mi := &file_go_chromium_org_luci_vpython_api_vpython_env_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Environment) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Environment) ProtoMessage() {}

func (x *Environment) ProtoReflect() protoreflect.Message {
	mi := &file_go_chromium_org_luci_vpython_api_vpython_env_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Environment.ProtoReflect.Descriptor instead.
func (*Environment) Descriptor() ([]byte, []int) {
	return file_go_chromium_org_luci_vpython_api_vpython_env_proto_rawDescGZIP(), []int{0}
}

func (x *Environment) GetSpec() *Spec {
	if x != nil {
		return x.Spec
	}
	return nil
}

func (x *Environment) GetRuntime() *Runtime {
	if x != nil {
		return x.Runtime
	}
	return nil
}

func (x *Environment) GetPep425Tag() []*PEP425Tag {
	if x != nil {
		return x.Pep425Tag
	}
	return nil
}

// Runtime is the set of resolved runtime parameters.
type Runtime struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// The absolute path to the resolved interpreter (sys.executable).
	Path string `protobuf:"bytes,1,opt,name=path,proto3" json:"path,omitempty"`
	// The SHA256 hash of the resolved interpreter.
	Hash string `protobuf:"bytes,2,opt,name=hash,proto3" json:"hash,omitempty"`
	// The resolved Python interpreter version.
	Version string `protobuf:"bytes,3,opt,name=version,proto3" json:"version,omitempty"`
	// The prefix of the Python interpreter (sys.prefix).
	Prefix string `protobuf:"bytes,4,opt,name=prefix,proto3" json:"prefix,omitempty"`
	// The architecture of vpython binary
	Arch string `protobuf:"bytes,5,opt,name=arch,proto3" json:"arch,omitempty"`
}

func (x *Runtime) Reset() {
	*x = Runtime{}
	if protoimpl.UnsafeEnabled {
		mi := &file_go_chromium_org_luci_vpython_api_vpython_env_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Runtime) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Runtime) ProtoMessage() {}

func (x *Runtime) ProtoReflect() protoreflect.Message {
	mi := &file_go_chromium_org_luci_vpython_api_vpython_env_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Runtime.ProtoReflect.Descriptor instead.
func (*Runtime) Descriptor() ([]byte, []int) {
	return file_go_chromium_org_luci_vpython_api_vpython_env_proto_rawDescGZIP(), []int{1}
}

func (x *Runtime) GetPath() string {
	if x != nil {
		return x.Path
	}
	return ""
}

func (x *Runtime) GetHash() string {
	if x != nil {
		return x.Hash
	}
	return ""
}

func (x *Runtime) GetVersion() string {
	if x != nil {
		return x.Version
	}
	return ""
}

func (x *Runtime) GetPrefix() string {
	if x != nil {
		return x.Prefix
	}
	return ""
}

func (x *Runtime) GetArch() string {
	if x != nil {
		return x.Arch
	}
	return ""
}

var File_go_chromium_org_luci_vpython_api_vpython_env_proto protoreflect.FileDescriptor

var file_go_chromium_org_luci_vpython_api_vpython_env_proto_rawDesc = []byte{
	0x0a, 0x32, 0x67, 0x6f, 0x2e, 0x63, 0x68, 0x72, 0x6f, 0x6d, 0x69, 0x75, 0x6d, 0x2e, 0x6f, 0x72,
	0x67, 0x2f, 0x6c, 0x75, 0x63, 0x69, 0x2f, 0x76, 0x70, 0x79, 0x74, 0x68, 0x6f, 0x6e, 0x2f, 0x61,
	0x70, 0x69, 0x2f, 0x76, 0x70, 0x79, 0x74, 0x68, 0x6f, 0x6e, 0x2f, 0x65, 0x6e, 0x76, 0x2e, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x12, 0x07, 0x76, 0x70, 0x79, 0x74, 0x68, 0x6f, 0x6e, 0x1a, 0x35, 0x67,
	0x6f, 0x2e, 0x63, 0x68, 0x72, 0x6f, 0x6d, 0x69, 0x75, 0x6d, 0x2e, 0x6f, 0x72, 0x67, 0x2f, 0x6c,
	0x75, 0x63, 0x69, 0x2f, 0x76, 0x70, 0x79, 0x74, 0x68, 0x6f, 0x6e, 0x2f, 0x61, 0x70, 0x69, 0x2f,
	0x76, 0x70, 0x79, 0x74, 0x68, 0x6f, 0x6e, 0x2f, 0x70, 0x65, 0x70, 0x34, 0x32, 0x35, 0x2e, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x33, 0x67, 0x6f, 0x2e, 0x63, 0x68, 0x72, 0x6f, 0x6d, 0x69, 0x75,
	0x6d, 0x2e, 0x6f, 0x72, 0x67, 0x2f, 0x6c, 0x75, 0x63, 0x69, 0x2f, 0x76, 0x70, 0x79, 0x74, 0x68,
	0x6f, 0x6e, 0x2f, 0x61, 0x70, 0x69, 0x2f, 0x76, 0x70, 0x79, 0x74, 0x68, 0x6f, 0x6e, 0x2f, 0x73,
	0x70, 0x65, 0x63, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x8f, 0x01, 0x0a, 0x0b, 0x45, 0x6e,
	0x76, 0x69, 0x72, 0x6f, 0x6e, 0x6d, 0x65, 0x6e, 0x74, 0x12, 0x21, 0x0a, 0x04, 0x73, 0x70, 0x65,
	0x63, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x0d, 0x2e, 0x76, 0x70, 0x79, 0x74, 0x68, 0x6f,
	0x6e, 0x2e, 0x53, 0x70, 0x65, 0x63, 0x52, 0x04, 0x73, 0x70, 0x65, 0x63, 0x12, 0x2a, 0x0a, 0x07,
	0x72, 0x75, 0x6e, 0x74, 0x69, 0x6d, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x10, 0x2e,
	0x76, 0x70, 0x79, 0x74, 0x68, 0x6f, 0x6e, 0x2e, 0x52, 0x75, 0x6e, 0x74, 0x69, 0x6d, 0x65, 0x52,
	0x07, 0x72, 0x75, 0x6e, 0x74, 0x69, 0x6d, 0x65, 0x12, 0x31, 0x0a, 0x0a, 0x70, 0x65, 0x70, 0x34,
	0x32, 0x35, 0x5f, 0x74, 0x61, 0x67, 0x18, 0x03, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x12, 0x2e, 0x76,
	0x70, 0x79, 0x74, 0x68, 0x6f, 0x6e, 0x2e, 0x50, 0x45, 0x50, 0x34, 0x32, 0x35, 0x54, 0x61, 0x67,
	0x52, 0x09, 0x70, 0x65, 0x70, 0x34, 0x32, 0x35, 0x54, 0x61, 0x67, 0x22, 0x77, 0x0a, 0x07, 0x52,
	0x75, 0x6e, 0x74, 0x69, 0x6d, 0x65, 0x12, 0x12, 0x0a, 0x04, 0x70, 0x61, 0x74, 0x68, 0x18, 0x01,
	0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x70, 0x61, 0x74, 0x68, 0x12, 0x12, 0x0a, 0x04, 0x68, 0x61,
	0x73, 0x68, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x68, 0x61, 0x73, 0x68, 0x12, 0x18,
	0x0a, 0x07, 0x76, 0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52,
	0x07, 0x76, 0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e, 0x12, 0x16, 0x0a, 0x06, 0x70, 0x72, 0x65, 0x66,
	0x69, 0x78, 0x18, 0x04, 0x20, 0x01, 0x28, 0x09, 0x52, 0x06, 0x70, 0x72, 0x65, 0x66, 0x69, 0x78,
	0x12, 0x12, 0x0a, 0x04, 0x61, 0x72, 0x63, 0x68, 0x18, 0x05, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04,
	0x61, 0x72, 0x63, 0x68, 0x42, 0x2a, 0x5a, 0x28, 0x67, 0x6f, 0x2e, 0x63, 0x68, 0x72, 0x6f, 0x6d,
	0x69, 0x75, 0x6d, 0x2e, 0x6f, 0x72, 0x67, 0x2f, 0x6c, 0x75, 0x63, 0x69, 0x2f, 0x76, 0x70, 0x79,
	0x74, 0x68, 0x6f, 0x6e, 0x2f, 0x61, 0x70, 0x69, 0x2f, 0x76, 0x70, 0x79, 0x74, 0x68, 0x6f, 0x6e,
	0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_go_chromium_org_luci_vpython_api_vpython_env_proto_rawDescOnce sync.Once
	file_go_chromium_org_luci_vpython_api_vpython_env_proto_rawDescData = file_go_chromium_org_luci_vpython_api_vpython_env_proto_rawDesc
)

func file_go_chromium_org_luci_vpython_api_vpython_env_proto_rawDescGZIP() []byte {
	file_go_chromium_org_luci_vpython_api_vpython_env_proto_rawDescOnce.Do(func() {
		file_go_chromium_org_luci_vpython_api_vpython_env_proto_rawDescData = protoimpl.X.CompressGZIP(file_go_chromium_org_luci_vpython_api_vpython_env_proto_rawDescData)
	})
	return file_go_chromium_org_luci_vpython_api_vpython_env_proto_rawDescData
}

var file_go_chromium_org_luci_vpython_api_vpython_env_proto_msgTypes = make([]protoimpl.MessageInfo, 2)
var file_go_chromium_org_luci_vpython_api_vpython_env_proto_goTypes = []interface{}{
	(*Environment)(nil), // 0: vpython.Environment
	(*Runtime)(nil),     // 1: vpython.Runtime
	(*Spec)(nil),        // 2: vpython.Spec
	(*PEP425Tag)(nil),   // 3: vpython.PEP425Tag
}
var file_go_chromium_org_luci_vpython_api_vpython_env_proto_depIdxs = []int32{
	2, // 0: vpython.Environment.spec:type_name -> vpython.Spec
	1, // 1: vpython.Environment.runtime:type_name -> vpython.Runtime
	3, // 2: vpython.Environment.pep425_tag:type_name -> vpython.PEP425Tag
	3, // [3:3] is the sub-list for method output_type
	3, // [3:3] is the sub-list for method input_type
	3, // [3:3] is the sub-list for extension type_name
	3, // [3:3] is the sub-list for extension extendee
	0, // [0:3] is the sub-list for field type_name
}

func init() { file_go_chromium_org_luci_vpython_api_vpython_env_proto_init() }
func file_go_chromium_org_luci_vpython_api_vpython_env_proto_init() {
	if File_go_chromium_org_luci_vpython_api_vpython_env_proto != nil {
		return
	}
	file_go_chromium_org_luci_vpython_api_vpython_pep425_proto_init()
	file_go_chromium_org_luci_vpython_api_vpython_spec_proto_init()
	if !protoimpl.UnsafeEnabled {
		file_go_chromium_org_luci_vpython_api_vpython_env_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Environment); i {
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
		file_go_chromium_org_luci_vpython_api_vpython_env_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Runtime); i {
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
			RawDescriptor: file_go_chromium_org_luci_vpython_api_vpython_env_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   2,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_go_chromium_org_luci_vpython_api_vpython_env_proto_goTypes,
		DependencyIndexes: file_go_chromium_org_luci_vpython_api_vpython_env_proto_depIdxs,
		MessageInfos:      file_go_chromium_org_luci_vpython_api_vpython_env_proto_msgTypes,
	}.Build()
	File_go_chromium_org_luci_vpython_api_vpython_env_proto = out.File
	file_go_chromium_org_luci_vpython_api_vpython_env_proto_rawDesc = nil
	file_go_chromium_org_luci_vpython_api_vpython_env_proto_goTypes = nil
	file_go_chromium_org_luci_vpython_api_vpython_env_proto_depIdxs = nil
}
