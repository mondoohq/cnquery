// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.36.2
// 	protoc        v3.12.4
// source: cnquery_report.proto

package reporter

import (
	_struct "github.com/golang/protobuf/ptypes/struct"
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

type Report struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Assets        map[string]*Asset      `protobuf:"bytes,1,rep,name=assets,proto3" json:"assets,omitempty" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"`
	Data          map[string]*DataValues `protobuf:"bytes,2,rep,name=data,proto3" json:"data,omitempty" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"`
	Errors        map[string]string      `protobuf:"bytes,3,rep,name=errors,proto3" json:"errors,omitempty" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Report) Reset() {
	*x = Report{}
	mi := &file_cnquery_report_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Report) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Report) ProtoMessage() {}

func (x *Report) ProtoReflect() protoreflect.Message {
	mi := &file_cnquery_report_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Report.ProtoReflect.Descriptor instead.
func (*Report) Descriptor() ([]byte, []int) {
	return file_cnquery_report_proto_rawDescGZIP(), []int{0}
}

func (x *Report) GetAssets() map[string]*Asset {
	if x != nil {
		return x.Assets
	}
	return nil
}

func (x *Report) GetData() map[string]*DataValues {
	if x != nil {
		return x.Data
	}
	return nil
}

func (x *Report) GetErrors() map[string]string {
	if x != nil {
		return x.Errors
	}
	return nil
}

type Asset struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Mrn           string                 `protobuf:"bytes,1,opt,name=mrn,proto3" json:"mrn,omitempty"`
	Name          string                 `protobuf:"bytes,2,opt,name=name,proto3" json:"name,omitempty"`
	Labels        map[string]string      `protobuf:"bytes,3,rep,name=labels,proto3" json:"labels,omitempty" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"`
	PlatformName  string                 `protobuf:"bytes,20,opt,name=platform_name,json=platformName,proto3" json:"platform_name,omitempty"`
	TraceId       string                 `protobuf:"bytes,21,opt,name=trace_id,json=traceId,proto3" json:"trace_id,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Asset) Reset() {
	*x = Asset{}
	mi := &file_cnquery_report_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Asset) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Asset) ProtoMessage() {}

func (x *Asset) ProtoReflect() protoreflect.Message {
	mi := &file_cnquery_report_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Asset.ProtoReflect.Descriptor instead.
func (*Asset) Descriptor() ([]byte, []int) {
	return file_cnquery_report_proto_rawDescGZIP(), []int{1}
}

func (x *Asset) GetMrn() string {
	if x != nil {
		return x.Mrn
	}
	return ""
}

func (x *Asset) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *Asset) GetLabels() map[string]string {
	if x != nil {
		return x.Labels
	}
	return nil
}

func (x *Asset) GetPlatformName() string {
	if x != nil {
		return x.PlatformName
	}
	return ""
}

func (x *Asset) GetTraceId() string {
	if x != nil {
		return x.TraceId
	}
	return ""
}

type DataValues struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Values        map[string]*DataValue  `protobuf:"bytes,1,rep,name=values,proto3" json:"values,omitempty" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *DataValues) Reset() {
	*x = DataValues{}
	mi := &file_cnquery_report_proto_msgTypes[2]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *DataValues) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DataValues) ProtoMessage() {}

func (x *DataValues) ProtoReflect() protoreflect.Message {
	mi := &file_cnquery_report_proto_msgTypes[2]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use DataValues.ProtoReflect.Descriptor instead.
func (*DataValues) Descriptor() ([]byte, []int) {
	return file_cnquery_report_proto_rawDescGZIP(), []int{2}
}

func (x *DataValues) GetValues() map[string]*DataValue {
	if x != nil {
		return x.Values
	}
	return nil
}

type DataValue struct {
	state protoimpl.MessageState `protogen:"open.v1"`
	// JSON object
	Content       *_struct.Value `protobuf:"bytes,1,opt,name=content,proto3" json:"content,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *DataValue) Reset() {
	*x = DataValue{}
	mi := &file_cnquery_report_proto_msgTypes[3]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *DataValue) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DataValue) ProtoMessage() {}

func (x *DataValue) ProtoReflect() protoreflect.Message {
	mi := &file_cnquery_report_proto_msgTypes[3]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use DataValue.ProtoReflect.Descriptor instead.
func (*DataValue) Descriptor() ([]byte, []int) {
	return file_cnquery_report_proto_rawDescGZIP(), []int{3}
}

func (x *DataValue) GetContent() *_struct.Value {
	if x != nil {
		return x.Content
	}
	return nil
}

var File_cnquery_report_proto protoreflect.FileDescriptor

var file_cnquery_report_proto_rawDesc = []byte{
	0x0a, 0x14, 0x63, 0x6e, 0x71, 0x75, 0x65, 0x72, 0x79, 0x5f, 0x72, 0x65, 0x70, 0x6f, 0x72, 0x74,
	0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x18, 0x6d, 0x6f, 0x6e, 0x64, 0x6f, 0x6f, 0x2e, 0x72,
	0x65, 0x70, 0x6f, 0x72, 0x74, 0x2e, 0x63, 0x6e, 0x71, 0x75, 0x65, 0x72, 0x79, 0x2e, 0x76, 0x31,
	0x1a, 0x1c, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75,
	0x66, 0x2f, 0x73, 0x74, 0x72, 0x75, 0x63, 0x74, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0xca,
	0x03, 0x0a, 0x06, 0x52, 0x65, 0x70, 0x6f, 0x72, 0x74, 0x12, 0x44, 0x0a, 0x06, 0x61, 0x73, 0x73,
	0x65, 0x74, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x2c, 0x2e, 0x6d, 0x6f, 0x6e, 0x64,
	0x6f, 0x6f, 0x2e, 0x72, 0x65, 0x70, 0x6f, 0x72, 0x74, 0x2e, 0x63, 0x6e, 0x71, 0x75, 0x65, 0x72,
	0x79, 0x2e, 0x76, 0x31, 0x2e, 0x52, 0x65, 0x70, 0x6f, 0x72, 0x74, 0x2e, 0x41, 0x73, 0x73, 0x65,
	0x74, 0x73, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x52, 0x06, 0x61, 0x73, 0x73, 0x65, 0x74, 0x73, 0x12,
	0x3e, 0x0a, 0x04, 0x64, 0x61, 0x74, 0x61, 0x18, 0x02, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x2a, 0x2e,
	0x6d, 0x6f, 0x6e, 0x64, 0x6f, 0x6f, 0x2e, 0x72, 0x65, 0x70, 0x6f, 0x72, 0x74, 0x2e, 0x63, 0x6e,
	0x71, 0x75, 0x65, 0x72, 0x79, 0x2e, 0x76, 0x31, 0x2e, 0x52, 0x65, 0x70, 0x6f, 0x72, 0x74, 0x2e,
	0x44, 0x61, 0x74, 0x61, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x52, 0x04, 0x64, 0x61, 0x74, 0x61, 0x12,
	0x44, 0x0a, 0x06, 0x65, 0x72, 0x72, 0x6f, 0x72, 0x73, 0x18, 0x03, 0x20, 0x03, 0x28, 0x0b, 0x32,
	0x2c, 0x2e, 0x6d, 0x6f, 0x6e, 0x64, 0x6f, 0x6f, 0x2e, 0x72, 0x65, 0x70, 0x6f, 0x72, 0x74, 0x2e,
	0x63, 0x6e, 0x71, 0x75, 0x65, 0x72, 0x79, 0x2e, 0x76, 0x31, 0x2e, 0x52, 0x65, 0x70, 0x6f, 0x72,
	0x74, 0x2e, 0x45, 0x72, 0x72, 0x6f, 0x72, 0x73, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x52, 0x06, 0x65,
	0x72, 0x72, 0x6f, 0x72, 0x73, 0x1a, 0x5a, 0x0a, 0x0b, 0x41, 0x73, 0x73, 0x65, 0x74, 0x73, 0x45,
	0x6e, 0x74, 0x72, 0x79, 0x12, 0x10, 0x0a, 0x03, 0x6b, 0x65, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28,
	0x09, 0x52, 0x03, 0x6b, 0x65, 0x79, 0x12, 0x35, 0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18,
	0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1f, 0x2e, 0x6d, 0x6f, 0x6e, 0x64, 0x6f, 0x6f, 0x2e, 0x72,
	0x65, 0x70, 0x6f, 0x72, 0x74, 0x2e, 0x63, 0x6e, 0x71, 0x75, 0x65, 0x72, 0x79, 0x2e, 0x76, 0x31,
	0x2e, 0x41, 0x73, 0x73, 0x65, 0x74, 0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x3a, 0x02, 0x38,
	0x01, 0x1a, 0x5d, 0x0a, 0x09, 0x44, 0x61, 0x74, 0x61, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x12, 0x10,
	0x0a, 0x03, 0x6b, 0x65, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x6b, 0x65, 0x79,
	0x12, 0x3a, 0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32,
	0x24, 0x2e, 0x6d, 0x6f, 0x6e, 0x64, 0x6f, 0x6f, 0x2e, 0x72, 0x65, 0x70, 0x6f, 0x72, 0x74, 0x2e,
	0x63, 0x6e, 0x71, 0x75, 0x65, 0x72, 0x79, 0x2e, 0x76, 0x31, 0x2e, 0x44, 0x61, 0x74, 0x61, 0x56,
	0x61, 0x6c, 0x75, 0x65, 0x73, 0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x3a, 0x02, 0x38, 0x01,
	0x1a, 0x39, 0x0a, 0x0b, 0x45, 0x72, 0x72, 0x6f, 0x72, 0x73, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x12,
	0x10, 0x0a, 0x03, 0x6b, 0x65, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x6b, 0x65,
	0x79, 0x12, 0x14, 0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09,
	0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x3a, 0x02, 0x38, 0x01, 0x22, 0xed, 0x01, 0x0a, 0x05,
	0x41, 0x73, 0x73, 0x65, 0x74, 0x12, 0x10, 0x0a, 0x03, 0x6d, 0x72, 0x6e, 0x18, 0x01, 0x20, 0x01,
	0x28, 0x09, 0x52, 0x03, 0x6d, 0x72, 0x6e, 0x12, 0x12, 0x0a, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x18,
	0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x12, 0x43, 0x0a, 0x06, 0x6c,
	0x61, 0x62, 0x65, 0x6c, 0x73, 0x18, 0x03, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x2b, 0x2e, 0x6d, 0x6f,
	0x6e, 0x64, 0x6f, 0x6f, 0x2e, 0x72, 0x65, 0x70, 0x6f, 0x72, 0x74, 0x2e, 0x63, 0x6e, 0x71, 0x75,
	0x65, 0x72, 0x79, 0x2e, 0x76, 0x31, 0x2e, 0x41, 0x73, 0x73, 0x65, 0x74, 0x2e, 0x4c, 0x61, 0x62,
	0x65, 0x6c, 0x73, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x52, 0x06, 0x6c, 0x61, 0x62, 0x65, 0x6c, 0x73,
	0x12, 0x23, 0x0a, 0x0d, 0x70, 0x6c, 0x61, 0x74, 0x66, 0x6f, 0x72, 0x6d, 0x5f, 0x6e, 0x61, 0x6d,
	0x65, 0x18, 0x14, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0c, 0x70, 0x6c, 0x61, 0x74, 0x66, 0x6f, 0x72,
	0x6d, 0x4e, 0x61, 0x6d, 0x65, 0x12, 0x19, 0x0a, 0x08, 0x74, 0x72, 0x61, 0x63, 0x65, 0x5f, 0x69,
	0x64, 0x18, 0x15, 0x20, 0x01, 0x28, 0x09, 0x52, 0x07, 0x74, 0x72, 0x61, 0x63, 0x65, 0x49, 0x64,
	0x1a, 0x39, 0x0a, 0x0b, 0x4c, 0x61, 0x62, 0x65, 0x6c, 0x73, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x12,
	0x10, 0x0a, 0x03, 0x6b, 0x65, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x6b, 0x65,
	0x79, 0x12, 0x14, 0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09,
	0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x3a, 0x02, 0x38, 0x01, 0x22, 0xb6, 0x01, 0x0a, 0x0a,
	0x44, 0x61, 0x74, 0x61, 0x56, 0x61, 0x6c, 0x75, 0x65, 0x73, 0x12, 0x48, 0x0a, 0x06, 0x76, 0x61,
	0x6c, 0x75, 0x65, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x30, 0x2e, 0x6d, 0x6f, 0x6e,
	0x64, 0x6f, 0x6f, 0x2e, 0x72, 0x65, 0x70, 0x6f, 0x72, 0x74, 0x2e, 0x63, 0x6e, 0x71, 0x75, 0x65,
	0x72, 0x79, 0x2e, 0x76, 0x31, 0x2e, 0x44, 0x61, 0x74, 0x61, 0x56, 0x61, 0x6c, 0x75, 0x65, 0x73,
	0x2e, 0x56, 0x61, 0x6c, 0x75, 0x65, 0x73, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x52, 0x06, 0x76, 0x61,
	0x6c, 0x75, 0x65, 0x73, 0x1a, 0x5e, 0x0a, 0x0b, 0x56, 0x61, 0x6c, 0x75, 0x65, 0x73, 0x45, 0x6e,
	0x74, 0x72, 0x79, 0x12, 0x10, 0x0a, 0x03, 0x6b, 0x65, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09,
	0x52, 0x03, 0x6b, 0x65, 0x79, 0x12, 0x39, 0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x02,
	0x20, 0x01, 0x28, 0x0b, 0x32, 0x23, 0x2e, 0x6d, 0x6f, 0x6e, 0x64, 0x6f, 0x6f, 0x2e, 0x72, 0x65,
	0x70, 0x6f, 0x72, 0x74, 0x2e, 0x63, 0x6e, 0x71, 0x75, 0x65, 0x72, 0x79, 0x2e, 0x76, 0x31, 0x2e,
	0x44, 0x61, 0x74, 0x61, 0x56, 0x61, 0x6c, 0x75, 0x65, 0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65,
	0x3a, 0x02, 0x38, 0x01, 0x22, 0x3d, 0x0a, 0x09, 0x44, 0x61, 0x74, 0x61, 0x56, 0x61, 0x6c, 0x75,
	0x65, 0x12, 0x30, 0x0a, 0x07, 0x63, 0x6f, 0x6e, 0x74, 0x65, 0x6e, 0x74, 0x18, 0x01, 0x20, 0x01,
	0x28, 0x0b, 0x32, 0x16, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x62, 0x75, 0x66, 0x2e, 0x56, 0x61, 0x6c, 0x75, 0x65, 0x52, 0x07, 0x63, 0x6f, 0x6e, 0x74,
	0x65, 0x6e, 0x74, 0x42, 0x28, 0x5a, 0x26, 0x67, 0x6f, 0x2e, 0x6d, 0x6f, 0x6e, 0x64, 0x6f, 0x6f,
	0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x63, 0x6e, 0x71, 0x75, 0x65, 0x72, 0x79, 0x2f, 0x76, 0x31, 0x31,
	0x2f, 0x63, 0x6c, 0x69, 0x2f, 0x72, 0x65, 0x70, 0x6f, 0x72, 0x74, 0x65, 0x72, 0x62, 0x06, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_cnquery_report_proto_rawDescOnce sync.Once
	file_cnquery_report_proto_rawDescData = file_cnquery_report_proto_rawDesc
)

func file_cnquery_report_proto_rawDescGZIP() []byte {
	file_cnquery_report_proto_rawDescOnce.Do(func() {
		file_cnquery_report_proto_rawDescData = protoimpl.X.CompressGZIP(file_cnquery_report_proto_rawDescData)
	})
	return file_cnquery_report_proto_rawDescData
}

var file_cnquery_report_proto_msgTypes = make([]protoimpl.MessageInfo, 9)
var file_cnquery_report_proto_goTypes = []any{
	(*Report)(nil),        // 0: mondoo.report.cnquery.v1.Report
	(*Asset)(nil),         // 1: mondoo.report.cnquery.v1.Asset
	(*DataValues)(nil),    // 2: mondoo.report.cnquery.v1.DataValues
	(*DataValue)(nil),     // 3: mondoo.report.cnquery.v1.DataValue
	nil,                   // 4: mondoo.report.cnquery.v1.Report.AssetsEntry
	nil,                   // 5: mondoo.report.cnquery.v1.Report.DataEntry
	nil,                   // 6: mondoo.report.cnquery.v1.Report.ErrorsEntry
	nil,                   // 7: mondoo.report.cnquery.v1.Asset.LabelsEntry
	nil,                   // 8: mondoo.report.cnquery.v1.DataValues.ValuesEntry
	(*_struct.Value)(nil), // 9: google.protobuf.Value
}
var file_cnquery_report_proto_depIdxs = []int32{
	4, // 0: mondoo.report.cnquery.v1.Report.assets:type_name -> mondoo.report.cnquery.v1.Report.AssetsEntry
	5, // 1: mondoo.report.cnquery.v1.Report.data:type_name -> mondoo.report.cnquery.v1.Report.DataEntry
	6, // 2: mondoo.report.cnquery.v1.Report.errors:type_name -> mondoo.report.cnquery.v1.Report.ErrorsEntry
	7, // 3: mondoo.report.cnquery.v1.Asset.labels:type_name -> mondoo.report.cnquery.v1.Asset.LabelsEntry
	8, // 4: mondoo.report.cnquery.v1.DataValues.values:type_name -> mondoo.report.cnquery.v1.DataValues.ValuesEntry
	9, // 5: mondoo.report.cnquery.v1.DataValue.content:type_name -> google.protobuf.Value
	1, // 6: mondoo.report.cnquery.v1.Report.AssetsEntry.value:type_name -> mondoo.report.cnquery.v1.Asset
	2, // 7: mondoo.report.cnquery.v1.Report.DataEntry.value:type_name -> mondoo.report.cnquery.v1.DataValues
	3, // 8: mondoo.report.cnquery.v1.DataValues.ValuesEntry.value:type_name -> mondoo.report.cnquery.v1.DataValue
	9, // [9:9] is the sub-list for method output_type
	9, // [9:9] is the sub-list for method input_type
	9, // [9:9] is the sub-list for extension type_name
	9, // [9:9] is the sub-list for extension extendee
	0, // [0:9] is the sub-list for field type_name
}

func init() { file_cnquery_report_proto_init() }
func file_cnquery_report_proto_init() {
	if File_cnquery_report_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_cnquery_report_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   9,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_cnquery_report_proto_goTypes,
		DependencyIndexes: file_cnquery_report_proto_depIdxs,
		MessageInfos:      file_cnquery_report_proto_msgTypes,
	}.Build()
	File_cnquery_report_proto = out.File
	file_cnquery_report_proto_rawDesc = nil
	file_cnquery_report_proto_goTypes = nil
	file_cnquery_report_proto_depIdxs = nil
}
