// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.33.0
// 	protoc        v4.25.3
// source: cnquery_explorer_scan.proto

package scan

import (
	explorer "go.mondoo.com/cnquery/v11/explorer"
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

type Job struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Inventory        *inventory.Inventory `protobuf:"bytes,1,opt,name=inventory,proto3" json:"inventory,omitempty"`
	Bundle           *explorer.Bundle     `protobuf:"bytes,2,opt,name=bundle,proto3" json:"bundle,omitempty"`
	DoRecord         bool                 `protobuf:"varint,20,opt,name=do_record,json=doRecord,proto3" json:"do_record,omitempty"`
	QueryPackFilters []string             `protobuf:"bytes,21,rep,name=query_pack_filters,json=queryPackFilters,proto3" json:"query_pack_filters,omitempty"`
	Props            map[string]string    `protobuf:"bytes,22,rep,name=props,proto3" json:"props,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
}

func (x *Job) Reset() {
	*x = Job{}
	if protoimpl.UnsafeEnabled {
		mi := &file_cnquery_explorer_scan_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Job) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Job) ProtoMessage() {}

func (x *Job) ProtoReflect() protoreflect.Message {
	mi := &file_cnquery_explorer_scan_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Job.ProtoReflect.Descriptor instead.
func (*Job) Descriptor() ([]byte, []int) {
	return file_cnquery_explorer_scan_proto_rawDescGZIP(), []int{0}
}

func (x *Job) GetInventory() *inventory.Inventory {
	if x != nil {
		return x.Inventory
	}
	return nil
}

func (x *Job) GetBundle() *explorer.Bundle {
	if x != nil {
		return x.Bundle
	}
	return nil
}

func (x *Job) GetDoRecord() bool {
	if x != nil {
		return x.DoRecord
	}
	return false
}

func (x *Job) GetQueryPackFilters() []string {
	if x != nil {
		return x.QueryPackFilters
	}
	return nil
}

func (x *Job) GetProps() map[string]string {
	if x != nil {
		return x.Props
	}
	return nil
}

var File_cnquery_explorer_scan_proto protoreflect.FileDescriptor

var file_cnquery_explorer_scan_proto_rawDesc = []byte{
	0x0a, 0x1b, 0x63, 0x6e, 0x71, 0x75, 0x65, 0x72, 0x79, 0x5f, 0x65, 0x78, 0x70, 0x6c, 0x6f, 0x72,
	0x65, 0x72, 0x5f, 0x73, 0x63, 0x61, 0x6e, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x15, 0x63,
	0x6e, 0x71, 0x75, 0x65, 0x72, 0x79, 0x2e, 0x65, 0x78, 0x70, 0x6c, 0x6f, 0x72, 0x65, 0x72, 0x2e,
	0x73, 0x63, 0x61, 0x6e, 0x1a, 0x1f, 0x65, 0x78, 0x70, 0x6c, 0x6f, 0x72, 0x65, 0x72, 0x2f, 0x63,
	0x6e, 0x71, 0x75, 0x65, 0x72, 0x79, 0x5f, 0x65, 0x78, 0x70, 0x6c, 0x6f, 0x72, 0x65, 0x72, 0x2e,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x2a, 0x70, 0x72, 0x6f, 0x76, 0x69, 0x64, 0x65, 0x72, 0x73,
	0x2d, 0x73, 0x64, 0x6b, 0x2f, 0x76, 0x31, 0x2f, 0x69, 0x6e, 0x76, 0x65, 0x6e, 0x74, 0x6f, 0x72,
	0x79, 0x2f, 0x69, 0x6e, 0x76, 0x65, 0x6e, 0x74, 0x6f, 0x72, 0x79, 0x2e, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x22, 0xb8, 0x02, 0x0a, 0x03, 0x4a, 0x6f, 0x62, 0x12, 0x3d, 0x0a, 0x09, 0x69, 0x6e, 0x76,
	0x65, 0x6e, 0x74, 0x6f, 0x72, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1f, 0x2e, 0x63,
	0x6e, 0x71, 0x75, 0x65, 0x72, 0x79, 0x2e, 0x70, 0x72, 0x6f, 0x76, 0x69, 0x64, 0x65, 0x72, 0x73,
	0x2e, 0x76, 0x31, 0x2e, 0x49, 0x6e, 0x76, 0x65, 0x6e, 0x74, 0x6f, 0x72, 0x79, 0x52, 0x09, 0x69,
	0x6e, 0x76, 0x65, 0x6e, 0x74, 0x6f, 0x72, 0x79, 0x12, 0x30, 0x0a, 0x06, 0x62, 0x75, 0x6e, 0x64,
	0x6c, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x18, 0x2e, 0x63, 0x6e, 0x71, 0x75, 0x65,
	0x72, 0x79, 0x2e, 0x65, 0x78, 0x70, 0x6c, 0x6f, 0x72, 0x65, 0x72, 0x2e, 0x42, 0x75, 0x6e, 0x64,
	0x6c, 0x65, 0x52, 0x06, 0x62, 0x75, 0x6e, 0x64, 0x6c, 0x65, 0x12, 0x1b, 0x0a, 0x09, 0x64, 0x6f,
	0x5f, 0x72, 0x65, 0x63, 0x6f, 0x72, 0x64, 0x18, 0x14, 0x20, 0x01, 0x28, 0x08, 0x52, 0x08, 0x64,
	0x6f, 0x52, 0x65, 0x63, 0x6f, 0x72, 0x64, 0x12, 0x2c, 0x0a, 0x12, 0x71, 0x75, 0x65, 0x72, 0x79,
	0x5f, 0x70, 0x61, 0x63, 0x6b, 0x5f, 0x66, 0x69, 0x6c, 0x74, 0x65, 0x72, 0x73, 0x18, 0x15, 0x20,
	0x03, 0x28, 0x09, 0x52, 0x10, 0x71, 0x75, 0x65, 0x72, 0x79, 0x50, 0x61, 0x63, 0x6b, 0x46, 0x69,
	0x6c, 0x74, 0x65, 0x72, 0x73, 0x12, 0x3b, 0x0a, 0x05, 0x70, 0x72, 0x6f, 0x70, 0x73, 0x18, 0x16,
	0x20, 0x03, 0x28, 0x0b, 0x32, 0x25, 0x2e, 0x63, 0x6e, 0x71, 0x75, 0x65, 0x72, 0x79, 0x2e, 0x65,
	0x78, 0x70, 0x6c, 0x6f, 0x72, 0x65, 0x72, 0x2e, 0x73, 0x63, 0x61, 0x6e, 0x2e, 0x4a, 0x6f, 0x62,
	0x2e, 0x50, 0x72, 0x6f, 0x70, 0x73, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x52, 0x05, 0x70, 0x72, 0x6f,
	0x70, 0x73, 0x1a, 0x38, 0x0a, 0x0a, 0x50, 0x72, 0x6f, 0x70, 0x73, 0x45, 0x6e, 0x74, 0x72, 0x79,
	0x12, 0x10, 0x0a, 0x03, 0x6b, 0x65, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x6b,
	0x65, 0x79, 0x12, 0x14, 0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28,
	0x09, 0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x3a, 0x02, 0x38, 0x01, 0x42, 0x29, 0x5a, 0x27,
	0x67, 0x6f, 0x2e, 0x6d, 0x6f, 0x6e, 0x64, 0x6f, 0x6f, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x63, 0x6e,
	0x71, 0x75, 0x65, 0x72, 0x79, 0x2f, 0x76, 0x31, 0x31, 0x2f, 0x65, 0x78, 0x70, 0x6c, 0x6f, 0x72,
	0x65, 0x72, 0x2f, 0x73, 0x63, 0x61, 0x6e, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_cnquery_explorer_scan_proto_rawDescOnce sync.Once
	file_cnquery_explorer_scan_proto_rawDescData = file_cnquery_explorer_scan_proto_rawDesc
)

func file_cnquery_explorer_scan_proto_rawDescGZIP() []byte {
	file_cnquery_explorer_scan_proto_rawDescOnce.Do(func() {
		file_cnquery_explorer_scan_proto_rawDescData = protoimpl.X.CompressGZIP(file_cnquery_explorer_scan_proto_rawDescData)
	})
	return file_cnquery_explorer_scan_proto_rawDescData
}

var file_cnquery_explorer_scan_proto_msgTypes = make([]protoimpl.MessageInfo, 2)
var file_cnquery_explorer_scan_proto_goTypes = []interface{}{
	(*Job)(nil),                 // 0: cnquery.explorer.scan.Job
	nil,                         // 1: cnquery.explorer.scan.Job.PropsEntry
	(*inventory.Inventory)(nil), // 2: cnquery.providers.v1.Inventory
	(*explorer.Bundle)(nil),     // 3: cnquery.explorer.Bundle
}
var file_cnquery_explorer_scan_proto_depIdxs = []int32{
	2, // 0: cnquery.explorer.scan.Job.inventory:type_name -> cnquery.providers.v1.Inventory
	3, // 1: cnquery.explorer.scan.Job.bundle:type_name -> cnquery.explorer.Bundle
	1, // 2: cnquery.explorer.scan.Job.props:type_name -> cnquery.explorer.scan.Job.PropsEntry
	3, // [3:3] is the sub-list for method output_type
	3, // [3:3] is the sub-list for method input_type
	3, // [3:3] is the sub-list for extension type_name
	3, // [3:3] is the sub-list for extension extendee
	0, // [0:3] is the sub-list for field type_name
}

func init() { file_cnquery_explorer_scan_proto_init() }
func file_cnquery_explorer_scan_proto_init() {
	if File_cnquery_explorer_scan_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_cnquery_explorer_scan_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Job); i {
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
			RawDescriptor: file_cnquery_explorer_scan_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   2,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_cnquery_explorer_scan_proto_goTypes,
		DependencyIndexes: file_cnquery_explorer_scan_proto_depIdxs,
		MessageInfos:      file_cnquery_explorer_scan_proto_msgTypes,
	}.Build()
	File_cnquery_explorer_scan_proto = out.File
	file_cnquery_explorer_scan_proto_rawDesc = nil
	file_cnquery_explorer_scan_proto_goTypes = nil
	file_cnquery_explorer_scan_proto_depIdxs = nil
}
