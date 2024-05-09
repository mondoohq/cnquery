// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.34.1
// 	protoc        v5.26.1
// source: cnquery.proto

package proto

import (
	inventory "go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"
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

type RunQueryConfig struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Command        string               `protobuf:"bytes,1,opt,name=command,proto3" json:"command,omitempty"`
	CallbackServer uint32               `protobuf:"varint,2,opt,name=callback_server,json=callbackServer,proto3" json:"callback_server,omitempty"`
	Inventory      *inventory.Inventory `protobuf:"bytes,3,opt,name=inventory,proto3" json:"inventory,omitempty"`
	Features       []byte               `protobuf:"bytes,4,opt,name=features,proto3" json:"features,omitempty"`
	DoParse        bool                 `protobuf:"varint,5,opt,name=do_parse,json=doParse,proto3" json:"do_parse,omitempty"`
	DoAst          bool                 `protobuf:"varint,6,opt,name=do_ast,json=doAst,proto3" json:"do_ast,omitempty"`
	DoInfo         bool                 `protobuf:"varint,13,opt,name=do_info,json=doInfo,proto3" json:"do_info,omitempty"`
	DoRecord       bool                 `protobuf:"varint,7,opt,name=do_record,json=doRecord,proto3" json:"do_record,omitempty"`
	Format         string               `protobuf:"bytes,8,opt,name=format,proto3" json:"format,omitempty"`
	PlatformId     string               `protobuf:"bytes,9,opt,name=platform_id,json=platformId,proto3" json:"platform_id,omitempty"`
	Incognito      bool                 `protobuf:"varint,10,opt,name=incognito,proto3" json:"incognito,omitempty"`
	Output         string               `protobuf:"bytes,11,opt,name=output,proto3" json:"output,omitempty"`
	Input          string               `protobuf:"bytes,12,opt,name=input,proto3" json:"input,omitempty"`
}

func (x *RunQueryConfig) Reset() {
	*x = RunQueryConfig{}
	if protoimpl.UnsafeEnabled {
		mi := &file_cnquery_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *RunQueryConfig) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RunQueryConfig) ProtoMessage() {}

func (x *RunQueryConfig) ProtoReflect() protoreflect.Message {
	mi := &file_cnquery_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RunQueryConfig.ProtoReflect.Descriptor instead.
func (*RunQueryConfig) Descriptor() ([]byte, []int) {
	return file_cnquery_proto_rawDescGZIP(), []int{0}
}

func (x *RunQueryConfig) GetCommand() string {
	if x != nil {
		return x.Command
	}
	return ""
}

func (x *RunQueryConfig) GetCallbackServer() uint32 {
	if x != nil {
		return x.CallbackServer
	}
	return 0
}

func (x *RunQueryConfig) GetInventory() *inventory.Inventory {
	if x != nil {
		return x.Inventory
	}
	return nil
}

func (x *RunQueryConfig) GetFeatures() []byte {
	if x != nil {
		return x.Features
	}
	return nil
}

func (x *RunQueryConfig) GetDoParse() bool {
	if x != nil {
		return x.DoParse
	}
	return false
}

func (x *RunQueryConfig) GetDoAst() bool {
	if x != nil {
		return x.DoAst
	}
	return false
}

func (x *RunQueryConfig) GetDoInfo() bool {
	if x != nil {
		return x.DoInfo
	}
	return false
}

func (x *RunQueryConfig) GetDoRecord() bool {
	if x != nil {
		return x.DoRecord
	}
	return false
}

func (x *RunQueryConfig) GetFormat() string {
	if x != nil {
		return x.Format
	}
	return ""
}

func (x *RunQueryConfig) GetPlatformId() string {
	if x != nil {
		return x.PlatformId
	}
	return ""
}

func (x *RunQueryConfig) GetIncognito() bool {
	if x != nil {
		return x.Incognito
	}
	return false
}

func (x *RunQueryConfig) GetOutput() string {
	if x != nil {
		return x.Output
	}
	return ""
}

func (x *RunQueryConfig) GetInput() string {
	if x != nil {
		return x.Input
	}
	return ""
}

type Empty struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
}

func (x *Empty) Reset() {
	*x = Empty{}
	if protoimpl.UnsafeEnabled {
		mi := &file_cnquery_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Empty) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Empty) ProtoMessage() {}

func (x *Empty) ProtoReflect() protoreflect.Message {
	mi := &file_cnquery_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Empty.ProtoReflect.Descriptor instead.
func (*Empty) Descriptor() ([]byte, []int) {
	return file_cnquery_proto_rawDescGZIP(), []int{1}
}

type String struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Data string `protobuf:"bytes,1,opt,name=data,proto3" json:"data,omitempty"`
}

func (x *String) Reset() {
	*x = String{}
	if protoimpl.UnsafeEnabled {
		mi := &file_cnquery_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *String) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*String) ProtoMessage() {}

func (x *String) ProtoReflect() protoreflect.Message {
	mi := &file_cnquery_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use String.ProtoReflect.Descriptor instead.
func (*String) Descriptor() ([]byte, []int) {
	return file_cnquery_proto_rawDescGZIP(), []int{2}
}

func (x *String) GetData() string {
	if x != nil {
		return x.Data
	}
	return ""
}

var File_cnquery_proto protoreflect.FileDescriptor

var file_cnquery_proto_rawDesc = []byte{
	0x0a, 0x0d, 0x63, 0x6e, 0x71, 0x75, 0x65, 0x72, 0x79, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12,
	0x05, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x2a, 0x70, 0x72, 0x6f, 0x76, 0x69, 0x64, 0x65, 0x72,
	0x73, 0x2d, 0x73, 0x64, 0x6b, 0x2f, 0x76, 0x31, 0x2f, 0x69, 0x6e, 0x76, 0x65, 0x6e, 0x74, 0x6f,
	0x72, 0x79, 0x2f, 0x69, 0x6e, 0x76, 0x65, 0x6e, 0x74, 0x6f, 0x72, 0x79, 0x2e, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x22, 0x9b, 0x03, 0x0a, 0x0e, 0x52, 0x75, 0x6e, 0x51, 0x75, 0x65, 0x72, 0x79, 0x43,
	0x6f, 0x6e, 0x66, 0x69, 0x67, 0x12, 0x18, 0x0a, 0x07, 0x63, 0x6f, 0x6d, 0x6d, 0x61, 0x6e, 0x64,
	0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x07, 0x63, 0x6f, 0x6d, 0x6d, 0x61, 0x6e, 0x64, 0x12,
	0x27, 0x0a, 0x0f, 0x63, 0x61, 0x6c, 0x6c, 0x62, 0x61, 0x63, 0x6b, 0x5f, 0x73, 0x65, 0x72, 0x76,
	0x65, 0x72, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0d, 0x52, 0x0e, 0x63, 0x61, 0x6c, 0x6c, 0x62, 0x61,
	0x63, 0x6b, 0x53, 0x65, 0x72, 0x76, 0x65, 0x72, 0x12, 0x3d, 0x0a, 0x09, 0x69, 0x6e, 0x76, 0x65,
	0x6e, 0x74, 0x6f, 0x72, 0x79, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1f, 0x2e, 0x63, 0x6e,
	0x71, 0x75, 0x65, 0x72, 0x79, 0x2e, 0x70, 0x72, 0x6f, 0x76, 0x69, 0x64, 0x65, 0x72, 0x73, 0x2e,
	0x76, 0x31, 0x2e, 0x49, 0x6e, 0x76, 0x65, 0x6e, 0x74, 0x6f, 0x72, 0x79, 0x52, 0x09, 0x69, 0x6e,
	0x76, 0x65, 0x6e, 0x74, 0x6f, 0x72, 0x79, 0x12, 0x1a, 0x0a, 0x08, 0x66, 0x65, 0x61, 0x74, 0x75,
	0x72, 0x65, 0x73, 0x18, 0x04, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x08, 0x66, 0x65, 0x61, 0x74, 0x75,
	0x72, 0x65, 0x73, 0x12, 0x19, 0x0a, 0x08, 0x64, 0x6f, 0x5f, 0x70, 0x61, 0x72, 0x73, 0x65, 0x18,
	0x05, 0x20, 0x01, 0x28, 0x08, 0x52, 0x07, 0x64, 0x6f, 0x50, 0x61, 0x72, 0x73, 0x65, 0x12, 0x15,
	0x0a, 0x06, 0x64, 0x6f, 0x5f, 0x61, 0x73, 0x74, 0x18, 0x06, 0x20, 0x01, 0x28, 0x08, 0x52, 0x05,
	0x64, 0x6f, 0x41, 0x73, 0x74, 0x12, 0x17, 0x0a, 0x07, 0x64, 0x6f, 0x5f, 0x69, 0x6e, 0x66, 0x6f,
	0x18, 0x0d, 0x20, 0x01, 0x28, 0x08, 0x52, 0x06, 0x64, 0x6f, 0x49, 0x6e, 0x66, 0x6f, 0x12, 0x1b,
	0x0a, 0x09, 0x64, 0x6f, 0x5f, 0x72, 0x65, 0x63, 0x6f, 0x72, 0x64, 0x18, 0x07, 0x20, 0x01, 0x28,
	0x08, 0x52, 0x08, 0x64, 0x6f, 0x52, 0x65, 0x63, 0x6f, 0x72, 0x64, 0x12, 0x16, 0x0a, 0x06, 0x66,
	0x6f, 0x72, 0x6d, 0x61, 0x74, 0x18, 0x08, 0x20, 0x01, 0x28, 0x09, 0x52, 0x06, 0x66, 0x6f, 0x72,
	0x6d, 0x61, 0x74, 0x12, 0x1f, 0x0a, 0x0b, 0x70, 0x6c, 0x61, 0x74, 0x66, 0x6f, 0x72, 0x6d, 0x5f,
	0x69, 0x64, 0x18, 0x09, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0a, 0x70, 0x6c, 0x61, 0x74, 0x66, 0x6f,
	0x72, 0x6d, 0x49, 0x64, 0x12, 0x1c, 0x0a, 0x09, 0x69, 0x6e, 0x63, 0x6f, 0x67, 0x6e, 0x69, 0x74,
	0x6f, 0x18, 0x0a, 0x20, 0x01, 0x28, 0x08, 0x52, 0x09, 0x69, 0x6e, 0x63, 0x6f, 0x67, 0x6e, 0x69,
	0x74, 0x6f, 0x12, 0x16, 0x0a, 0x06, 0x6f, 0x75, 0x74, 0x70, 0x75, 0x74, 0x18, 0x0b, 0x20, 0x01,
	0x28, 0x09, 0x52, 0x06, 0x6f, 0x75, 0x74, 0x70, 0x75, 0x74, 0x12, 0x14, 0x0a, 0x05, 0x69, 0x6e,
	0x70, 0x75, 0x74, 0x18, 0x0c, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x69, 0x6e, 0x70, 0x75, 0x74,
	0x22, 0x07, 0x0a, 0x05, 0x45, 0x6d, 0x70, 0x74, 0x79, 0x22, 0x1c, 0x0a, 0x06, 0x53, 0x74, 0x72,
	0x69, 0x6e, 0x67, 0x12, 0x12, 0x0a, 0x04, 0x64, 0x61, 0x74, 0x61, 0x18, 0x01, 0x20, 0x01, 0x28,
	0x09, 0x52, 0x04, 0x64, 0x61, 0x74, 0x61, 0x32, 0x3a, 0x0a, 0x07, 0x43, 0x4e, 0x51, 0x75, 0x65,
	0x72, 0x79, 0x12, 0x2f, 0x0a, 0x08, 0x52, 0x75, 0x6e, 0x51, 0x75, 0x65, 0x72, 0x79, 0x12, 0x15,
	0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x52, 0x75, 0x6e, 0x51, 0x75, 0x65, 0x72, 0x79, 0x43,
	0x6f, 0x6e, 0x66, 0x69, 0x67, 0x1a, 0x0c, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x45, 0x6d,
	0x70, 0x74, 0x79, 0x32, 0x34, 0x0a, 0x0c, 0x4f, 0x75, 0x74, 0x70, 0x75, 0x74, 0x48, 0x65, 0x6c,
	0x70, 0x65, 0x72, 0x12, 0x24, 0x0a, 0x05, 0x57, 0x72, 0x69, 0x74, 0x65, 0x12, 0x0d, 0x2e, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x53, 0x74, 0x72, 0x69, 0x6e, 0x67, 0x1a, 0x0c, 0x2e, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x2e, 0x45, 0x6d, 0x70, 0x74, 0x79, 0x42, 0x28, 0x5a, 0x26, 0x67, 0x6f, 0x2e,
	0x6d, 0x6f, 0x6e, 0x64, 0x6f, 0x6f, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x63, 0x6e, 0x71, 0x75, 0x65,
	0x72, 0x79, 0x2f, 0x76, 0x31, 0x31, 0x2f, 0x73, 0x68, 0x61, 0x72, 0x65, 0x64, 0x2f, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_cnquery_proto_rawDescOnce sync.Once
	file_cnquery_proto_rawDescData = file_cnquery_proto_rawDesc
)

func file_cnquery_proto_rawDescGZIP() []byte {
	file_cnquery_proto_rawDescOnce.Do(func() {
		file_cnquery_proto_rawDescData = protoimpl.X.CompressGZIP(file_cnquery_proto_rawDescData)
	})
	return file_cnquery_proto_rawDescData
}

var file_cnquery_proto_msgTypes = make([]protoimpl.MessageInfo, 3)
var file_cnquery_proto_goTypes = []interface{}{
	(*RunQueryConfig)(nil),      // 0: proto.RunQueryConfig
	(*Empty)(nil),               // 1: proto.Empty
	(*String)(nil),              // 2: proto.String
	(*inventory.Inventory)(nil), // 3: cnquery.providers.v1.Inventory
}
var file_cnquery_proto_depIdxs = []int32{
	3, // 0: proto.RunQueryConfig.inventory:type_name -> cnquery.providers.v1.Inventory
	0, // 1: proto.CNQuery.RunQuery:input_type -> proto.RunQueryConfig
	2, // 2: proto.OutputHelper.Write:input_type -> proto.String
	1, // 3: proto.CNQuery.RunQuery:output_type -> proto.Empty
	1, // 4: proto.OutputHelper.Write:output_type -> proto.Empty
	3, // [3:5] is the sub-list for method output_type
	1, // [1:3] is the sub-list for method input_type
	1, // [1:1] is the sub-list for extension type_name
	1, // [1:1] is the sub-list for extension extendee
	0, // [0:1] is the sub-list for field type_name
}

func init() { file_cnquery_proto_init() }
func file_cnquery_proto_init() {
	if File_cnquery_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_cnquery_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*RunQueryConfig); i {
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
		file_cnquery_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Empty); i {
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
		file_cnquery_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*String); i {
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
			RawDescriptor: file_cnquery_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   3,
			NumExtensions: 0,
			NumServices:   2,
		},
		GoTypes:           file_cnquery_proto_goTypes,
		DependencyIndexes: file_cnquery_proto_depIdxs,
		MessageInfos:      file_cnquery_proto_msgTypes,
	}.Build()
	File_cnquery_proto = out.File
	file_cnquery_proto_rawDesc = nil
	file_cnquery_proto_goTypes = nil
	file_cnquery_proto_depIdxs = nil
}
