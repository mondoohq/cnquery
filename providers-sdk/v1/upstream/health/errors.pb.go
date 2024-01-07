// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.32.0
// 	protoc        v4.25.1
// source: errors.proto

package health

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

type SendErrorReq struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	ServiceAccountMrn string       `protobuf:"bytes,1,opt,name=service_account_mrn,json=serviceAccountMrn,proto3" json:"service_account_mrn,omitempty"`
	AgentMrn          string       `protobuf:"bytes,2,opt,name=agent_mrn,json=agentMrn,proto3" json:"agent_mrn,omitempty"`
	Product           *ProductInfo `protobuf:"bytes,3,opt,name=product,proto3" json:"product,omitempty"`
	Error             *ErrorInfo   `protobuf:"bytes,4,opt,name=error,proto3" json:"error,omitempty"`
}

func (x *SendErrorReq) Reset() {
	*x = SendErrorReq{}
	if protoimpl.UnsafeEnabled {
		mi := &file_errors_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *SendErrorReq) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*SendErrorReq) ProtoMessage() {}

func (x *SendErrorReq) ProtoReflect() protoreflect.Message {
	mi := &file_errors_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use SendErrorReq.ProtoReflect.Descriptor instead.
func (*SendErrorReq) Descriptor() ([]byte, []int) {
	return file_errors_proto_rawDescGZIP(), []int{0}
}

func (x *SendErrorReq) GetServiceAccountMrn() string {
	if x != nil {
		return x.ServiceAccountMrn
	}
	return ""
}

func (x *SendErrorReq) GetAgentMrn() string {
	if x != nil {
		return x.AgentMrn
	}
	return ""
}

func (x *SendErrorReq) GetProduct() *ProductInfo {
	if x != nil {
		return x.Product
	}
	return nil
}

func (x *SendErrorReq) GetError() *ErrorInfo {
	if x != nil {
		return x.Error
	}
	return nil
}

type ProductInfo struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Name    string `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	Version string `protobuf:"bytes,2,opt,name=version,proto3" json:"version,omitempty"`
	Build   string `protobuf:"bytes,3,opt,name=build,proto3" json:"build,omitempty"`
}

func (x *ProductInfo) Reset() {
	*x = ProductInfo{}
	if protoimpl.UnsafeEnabled {
		mi := &file_errors_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ProductInfo) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ProductInfo) ProtoMessage() {}

func (x *ProductInfo) ProtoReflect() protoreflect.Message {
	mi := &file_errors_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ProductInfo.ProtoReflect.Descriptor instead.
func (*ProductInfo) Descriptor() ([]byte, []int) {
	return file_errors_proto_rawDescGZIP(), []int{1}
}

func (x *ProductInfo) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *ProductInfo) GetVersion() string {
	if x != nil {
		return x.Version
	}
	return ""
}

func (x *ProductInfo) GetBuild() string {
	if x != nil {
		return x.Build
	}
	return ""
}

type ErrorInfo struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Message    string `protobuf:"bytes,1,opt,name=message,proto3" json:"message,omitempty"`
	Stacktrace string `protobuf:"bytes,2,opt,name=stacktrace,proto3" json:"stacktrace,omitempty"`
}

func (x *ErrorInfo) Reset() {
	*x = ErrorInfo{}
	if protoimpl.UnsafeEnabled {
		mi := &file_errors_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ErrorInfo) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ErrorInfo) ProtoMessage() {}

func (x *ErrorInfo) ProtoReflect() protoreflect.Message {
	mi := &file_errors_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ErrorInfo.ProtoReflect.Descriptor instead.
func (*ErrorInfo) Descriptor() ([]byte, []int) {
	return file_errors_proto_rawDescGZIP(), []int{2}
}

func (x *ErrorInfo) GetMessage() string {
	if x != nil {
		return x.Message
	}
	return ""
}

func (x *ErrorInfo) GetStacktrace() string {
	if x != nil {
		return x.Stacktrace
	}
	return ""
}

type SendErrorResp struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
}

func (x *SendErrorResp) Reset() {
	*x = SendErrorResp{}
	if protoimpl.UnsafeEnabled {
		mi := &file_errors_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *SendErrorResp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*SendErrorResp) ProtoMessage() {}

func (x *SendErrorResp) ProtoReflect() protoreflect.Message {
	mi := &file_errors_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use SendErrorResp.ProtoReflect.Descriptor instead.
func (*SendErrorResp) Descriptor() ([]byte, []int) {
	return file_errors_proto_rawDescGZIP(), []int{3}
}

var File_errors_proto protoreflect.FileDescriptor

var file_errors_proto_rawDesc = []byte{
	0x0a, 0x0c, 0x65, 0x72, 0x72, 0x6f, 0x72, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x13,
	0x67, 0x6f, 0x2e, 0x6d, 0x6f, 0x6e, 0x64, 0x6f, 0x6f, 0x2e, 0x68, 0x65, 0x61, 0x6c, 0x74, 0x68,
	0x2e, 0x76, 0x31, 0x22, 0xcd, 0x01, 0x0a, 0x0c, 0x53, 0x65, 0x6e, 0x64, 0x45, 0x72, 0x72, 0x6f,
	0x72, 0x52, 0x65, 0x71, 0x12, 0x2e, 0x0a, 0x13, 0x73, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x5f,
	0x61, 0x63, 0x63, 0x6f, 0x75, 0x6e, 0x74, 0x5f, 0x6d, 0x72, 0x6e, 0x18, 0x01, 0x20, 0x01, 0x28,
	0x09, 0x52, 0x11, 0x73, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x41, 0x63, 0x63, 0x6f, 0x75, 0x6e,
	0x74, 0x4d, 0x72, 0x6e, 0x12, 0x1b, 0x0a, 0x09, 0x61, 0x67, 0x65, 0x6e, 0x74, 0x5f, 0x6d, 0x72,
	0x6e, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x08, 0x61, 0x67, 0x65, 0x6e, 0x74, 0x4d, 0x72,
	0x6e, 0x12, 0x3a, 0x0a, 0x07, 0x70, 0x72, 0x6f, 0x64, 0x75, 0x63, 0x74, 0x18, 0x03, 0x20, 0x01,
	0x28, 0x0b, 0x32, 0x20, 0x2e, 0x67, 0x6f, 0x2e, 0x6d, 0x6f, 0x6e, 0x64, 0x6f, 0x6f, 0x2e, 0x68,
	0x65, 0x61, 0x6c, 0x74, 0x68, 0x2e, 0x76, 0x31, 0x2e, 0x50, 0x72, 0x6f, 0x64, 0x75, 0x63, 0x74,
	0x49, 0x6e, 0x66, 0x6f, 0x52, 0x07, 0x70, 0x72, 0x6f, 0x64, 0x75, 0x63, 0x74, 0x12, 0x34, 0x0a,
	0x05, 0x65, 0x72, 0x72, 0x6f, 0x72, 0x18, 0x04, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1e, 0x2e, 0x67,
	0x6f, 0x2e, 0x6d, 0x6f, 0x6e, 0x64, 0x6f, 0x6f, 0x2e, 0x68, 0x65, 0x61, 0x6c, 0x74, 0x68, 0x2e,
	0x76, 0x31, 0x2e, 0x45, 0x72, 0x72, 0x6f, 0x72, 0x49, 0x6e, 0x66, 0x6f, 0x52, 0x05, 0x65, 0x72,
	0x72, 0x6f, 0x72, 0x22, 0x51, 0x0a, 0x0b, 0x50, 0x72, 0x6f, 0x64, 0x75, 0x63, 0x74, 0x49, 0x6e,
	0x66, 0x6f, 0x12, 0x12, 0x0a, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09,
	0x52, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x12, 0x18, 0x0a, 0x07, 0x76, 0x65, 0x72, 0x73, 0x69, 0x6f,
	0x6e, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x07, 0x76, 0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e,
	0x12, 0x14, 0x0a, 0x05, 0x62, 0x75, 0x69, 0x6c, 0x64, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52,
	0x05, 0x62, 0x75, 0x69, 0x6c, 0x64, 0x22, 0x45, 0x0a, 0x09, 0x45, 0x72, 0x72, 0x6f, 0x72, 0x49,
	0x6e, 0x66, 0x6f, 0x12, 0x18, 0x0a, 0x07, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x18, 0x01,
	0x20, 0x01, 0x28, 0x09, 0x52, 0x07, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x12, 0x1e, 0x0a,
	0x0a, 0x73, 0x74, 0x61, 0x63, 0x6b, 0x74, 0x72, 0x61, 0x63, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28,
	0x09, 0x52, 0x0a, 0x73, 0x74, 0x61, 0x63, 0x6b, 0x74, 0x72, 0x61, 0x63, 0x65, 0x22, 0x0f, 0x0a,
	0x0d, 0x53, 0x65, 0x6e, 0x64, 0x45, 0x72, 0x72, 0x6f, 0x72, 0x52, 0x65, 0x73, 0x70, 0x32, 0x64,
	0x0a, 0x0e, 0x45, 0x72, 0x72, 0x6f, 0x72, 0x52, 0x65, 0x70, 0x6f, 0x72, 0x74, 0x69, 0x6e, 0x67,
	0x12, 0x52, 0x0a, 0x09, 0x53, 0x65, 0x6e, 0x64, 0x45, 0x72, 0x72, 0x6f, 0x72, 0x12, 0x21, 0x2e,
	0x67, 0x6f, 0x2e, 0x6d, 0x6f, 0x6e, 0x64, 0x6f, 0x6f, 0x2e, 0x68, 0x65, 0x61, 0x6c, 0x74, 0x68,
	0x2e, 0x76, 0x31, 0x2e, 0x53, 0x65, 0x6e, 0x64, 0x45, 0x72, 0x72, 0x6f, 0x72, 0x52, 0x65, 0x71,
	0x1a, 0x22, 0x2e, 0x67, 0x6f, 0x2e, 0x6d, 0x6f, 0x6e, 0x64, 0x6f, 0x6f, 0x2e, 0x68, 0x65, 0x61,
	0x6c, 0x74, 0x68, 0x2e, 0x76, 0x31, 0x2e, 0x53, 0x65, 0x6e, 0x64, 0x45, 0x72, 0x72, 0x6f, 0x72,
	0x52, 0x65, 0x73, 0x70, 0x42, 0x1d, 0x5a, 0x1b, 0x67, 0x6f, 0x2e, 0x6d, 0x6f, 0x6e, 0x64, 0x6f,
	0x6f, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x6d, 0x6f, 0x6e, 0x64, 0x6f, 0x6f, 0x2f, 0x68, 0x65, 0x61,
	0x6c, 0x74, 0x68, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_errors_proto_rawDescOnce sync.Once
	file_errors_proto_rawDescData = file_errors_proto_rawDesc
)

func file_errors_proto_rawDescGZIP() []byte {
	file_errors_proto_rawDescOnce.Do(func() {
		file_errors_proto_rawDescData = protoimpl.X.CompressGZIP(file_errors_proto_rawDescData)
	})
	return file_errors_proto_rawDescData
}

var file_errors_proto_msgTypes = make([]protoimpl.MessageInfo, 4)
var file_errors_proto_goTypes = []interface{}{
	(*SendErrorReq)(nil),  // 0: go.mondoo.health.v1.SendErrorReq
	(*ProductInfo)(nil),   // 1: go.mondoo.health.v1.ProductInfo
	(*ErrorInfo)(nil),     // 2: go.mondoo.health.v1.ErrorInfo
	(*SendErrorResp)(nil), // 3: go.mondoo.health.v1.SendErrorResp
}
var file_errors_proto_depIdxs = []int32{
	1, // 0: go.mondoo.health.v1.SendErrorReq.product:type_name -> go.mondoo.health.v1.ProductInfo
	2, // 1: go.mondoo.health.v1.SendErrorReq.error:type_name -> go.mondoo.health.v1.ErrorInfo
	0, // 2: go.mondoo.health.v1.ErrorReporting.SendError:input_type -> go.mondoo.health.v1.SendErrorReq
	3, // 3: go.mondoo.health.v1.ErrorReporting.SendError:output_type -> go.mondoo.health.v1.SendErrorResp
	3, // [3:4] is the sub-list for method output_type
	2, // [2:3] is the sub-list for method input_type
	2, // [2:2] is the sub-list for extension type_name
	2, // [2:2] is the sub-list for extension extendee
	0, // [0:2] is the sub-list for field type_name
}

func init() { file_errors_proto_init() }
func file_errors_proto_init() {
	if File_errors_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_errors_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*SendErrorReq); i {
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
		file_errors_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ProductInfo); i {
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
		file_errors_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ErrorInfo); i {
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
		file_errors_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*SendErrorResp); i {
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
			RawDescriptor: file_errors_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   4,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_errors_proto_goTypes,
		DependencyIndexes: file_errors_proto_depIdxs,
		MessageInfos:      file_errors_proto_msgTypes,
	}.Build()
	File_errors_proto = out.File
	file_errors_proto_rawDesc = nil
	file_errors_proto_goTypes = nil
	file_errors_proto_depIdxs = nil
}
