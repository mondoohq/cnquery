// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.28.0
// 	protoc        v3.21.2
// source: service.proto

package service

import (
	resources "go.mondoo.com/cnquery/resources"
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

type Empty struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
}

func (x *Empty) Reset() {
	*x = Empty{}
	if protoimpl.UnsafeEnabled {
		mi := &file_service_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Empty) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Empty) ProtoMessage() {}

func (x *Empty) ProtoReflect() protoreflect.Message {
	mi := &file_service_proto_msgTypes[0]
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
	return file_service_proto_rawDescGZIP(), []int{0}
}

type ResourceList struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Resources []string `protobuf:"bytes,1,rep,name=resources,proto3" json:"resources,omitempty"`
}

func (x *ResourceList) Reset() {
	*x = ResourceList{}
	if protoimpl.UnsafeEnabled {
		mi := &file_service_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ResourceList) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ResourceList) ProtoMessage() {}

func (x *ResourceList) ProtoReflect() protoreflect.Message {
	mi := &file_service_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ResourceList.ProtoReflect.Descriptor instead.
func (*ResourceList) Descriptor() ([]byte, []int) {
	return file_service_proto_rawDescGZIP(), []int{1}
}

func (x *ResourceList) GetResources() []string {
	if x != nil {
		return x.Resources
	}
	return nil
}

type Fields struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Fields map[string]*resources.Field `protobuf:"bytes,1,rep,name=fields,proto3" json:"fields,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
}

func (x *Fields) Reset() {
	*x = Fields{}
	if protoimpl.UnsafeEnabled {
		mi := &file_service_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Fields) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Fields) ProtoMessage() {}

func (x *Fields) ProtoReflect() protoreflect.Message {
	mi := &file_service_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Fields.ProtoReflect.Descriptor instead.
func (*Fields) Descriptor() ([]byte, []int) {
	return file_service_proto_rawDescGZIP(), []int{2}
}

func (x *Fields) GetFields() map[string]*resources.Field {
	if x != nil {
		return x.Fields
	}
	return nil
}

type FieldsQuery struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// resource name
	Name string `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
}

func (x *FieldsQuery) Reset() {
	*x = FieldsQuery{}
	if protoimpl.UnsafeEnabled {
		mi := &file_service_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *FieldsQuery) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*FieldsQuery) ProtoMessage() {}

func (x *FieldsQuery) ProtoReflect() protoreflect.Message {
	mi := &file_service_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use FieldsQuery.ProtoReflect.Descriptor instead.
func (*FieldsQuery) Descriptor() ([]byte, []int) {
	return file_service_proto_rawDescGZIP(), []int{3}
}

func (x *FieldsQuery) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

type ResourceArguments struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Name  string            `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	Named map[string]string `protobuf:"bytes,2,rep,name=named,proto3" json:"named,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
}

func (x *ResourceArguments) Reset() {
	*x = ResourceArguments{}
	if protoimpl.UnsafeEnabled {
		mi := &file_service_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ResourceArguments) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ResourceArguments) ProtoMessage() {}

func (x *ResourceArguments) ProtoReflect() protoreflect.Message {
	mi := &file_service_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ResourceArguments.ProtoReflect.Descriptor instead.
func (*ResourceArguments) Descriptor() ([]byte, []int) {
	return file_service_proto_rawDescGZIP(), []int{4}
}

func (x *ResourceArguments) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *ResourceArguments) GetNamed() map[string]string {
	if x != nil {
		return x.Named
	}
	return nil
}

type FieldArguments struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Name  string `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	Id    string `protobuf:"bytes,2,opt,name=id,proto3" json:"id,omitempty"`
	Field string `protobuf:"bytes,3,opt,name=field,proto3" json:"field,omitempty"`
}

func (x *FieldArguments) Reset() {
	*x = FieldArguments{}
	if protoimpl.UnsafeEnabled {
		mi := &file_service_proto_msgTypes[5]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *FieldArguments) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*FieldArguments) ProtoMessage() {}

func (x *FieldArguments) ProtoReflect() protoreflect.Message {
	mi := &file_service_proto_msgTypes[5]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use FieldArguments.ProtoReflect.Descriptor instead.
func (*FieldArguments) Descriptor() ([]byte, []int) {
	return file_service_proto_rawDescGZIP(), []int{5}
}

func (x *FieldArguments) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *FieldArguments) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

func (x *FieldArguments) GetField() string {
	if x != nil {
		return x.Field
	}
	return ""
}

type FieldReturn struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Data []byte `protobuf:"bytes,3,opt,name=data,proto3" json:"data,omitempty"`
}

func (x *FieldReturn) Reset() {
	*x = FieldReturn{}
	if protoimpl.UnsafeEnabled {
		mi := &file_service_proto_msgTypes[6]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *FieldReturn) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*FieldReturn) ProtoMessage() {}

func (x *FieldReturn) ProtoReflect() protoreflect.Message {
	mi := &file_service_proto_msgTypes[6]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use FieldReturn.ProtoReflect.Descriptor instead.
func (*FieldReturn) Descriptor() ([]byte, []int) {
	return file_service_proto_rawDescGZIP(), []int{6}
}

func (x *FieldReturn) GetData() []byte {
	if x != nil {
		return x.Data
	}
	return nil
}

var File_service_proto protoreflect.FileDescriptor

var file_service_proto_rawDesc = []byte{
	0x0a, 0x0d, 0x73, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12,
	0x18, 0x6d, 0x6f, 0x6e, 0x64, 0x6f, 0x6f, 0x2e, 0x72, 0x65, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65,
	0x73, 0x2e, 0x73, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x1a, 0x0f, 0x72, 0x65, 0x73, 0x6f, 0x75,
	0x72, 0x63, 0x65, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x07, 0x0a, 0x05, 0x45, 0x6d,
	0x70, 0x74, 0x79, 0x22, 0x2c, 0x0a, 0x0c, 0x52, 0x65, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x4c,
	0x69, 0x73, 0x74, 0x12, 0x1c, 0x0a, 0x09, 0x72, 0x65, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x73,
	0x18, 0x01, 0x20, 0x03, 0x28, 0x09, 0x52, 0x09, 0x72, 0x65, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65,
	0x73, 0x22, 0xa2, 0x01, 0x0a, 0x06, 0x46, 0x69, 0x65, 0x6c, 0x64, 0x73, 0x12, 0x44, 0x0a, 0x06,
	0x66, 0x69, 0x65, 0x6c, 0x64, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x2c, 0x2e, 0x6d,
	0x6f, 0x6e, 0x64, 0x6f, 0x6f, 0x2e, 0x72, 0x65, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x73, 0x2e,
	0x73, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x2e, 0x46, 0x69, 0x65, 0x6c, 0x64, 0x73, 0x2e, 0x46,
	0x69, 0x65, 0x6c, 0x64, 0x73, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x52, 0x06, 0x66, 0x69, 0x65, 0x6c,
	0x64, 0x73, 0x1a, 0x52, 0x0a, 0x0b, 0x46, 0x69, 0x65, 0x6c, 0x64, 0x73, 0x45, 0x6e, 0x74, 0x72,
	0x79, 0x12, 0x10, 0x0a, 0x03, 0x6b, 0x65, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03,
	0x6b, 0x65, 0x79, 0x12, 0x2d, 0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x02, 0x20, 0x01,
	0x28, 0x0b, 0x32, 0x17, 0x2e, 0x6d, 0x6f, 0x6e, 0x64, 0x6f, 0x6f, 0x2e, 0x72, 0x65, 0x73, 0x6f,
	0x75, 0x72, 0x63, 0x65, 0x73, 0x2e, 0x46, 0x69, 0x65, 0x6c, 0x64, 0x52, 0x05, 0x76, 0x61, 0x6c,
	0x75, 0x65, 0x3a, 0x02, 0x38, 0x01, 0x22, 0x21, 0x0a, 0x0b, 0x46, 0x69, 0x65, 0x6c, 0x64, 0x73,
	0x51, 0x75, 0x65, 0x72, 0x79, 0x12, 0x12, 0x0a, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x01, 0x20,
	0x01, 0x28, 0x09, 0x52, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x22, 0xaf, 0x01, 0x0a, 0x11, 0x52, 0x65,
	0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x41, 0x72, 0x67, 0x75, 0x6d, 0x65, 0x6e, 0x74, 0x73, 0x12,
	0x12, 0x0a, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x6e,
	0x61, 0x6d, 0x65, 0x12, 0x4c, 0x0a, 0x05, 0x6e, 0x61, 0x6d, 0x65, 0x64, 0x18, 0x02, 0x20, 0x03,
	0x28, 0x0b, 0x32, 0x36, 0x2e, 0x6d, 0x6f, 0x6e, 0x64, 0x6f, 0x6f, 0x2e, 0x72, 0x65, 0x73, 0x6f,
	0x75, 0x72, 0x63, 0x65, 0x73, 0x2e, 0x73, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x2e, 0x52, 0x65,
	0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x41, 0x72, 0x67, 0x75, 0x6d, 0x65, 0x6e, 0x74, 0x73, 0x2e,
	0x4e, 0x61, 0x6d, 0x65, 0x64, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x52, 0x05, 0x6e, 0x61, 0x6d, 0x65,
	0x64, 0x1a, 0x38, 0x0a, 0x0a, 0x4e, 0x61, 0x6d, 0x65, 0x64, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x12,
	0x10, 0x0a, 0x03, 0x6b, 0x65, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x6b, 0x65,
	0x79, 0x12, 0x14, 0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09,
	0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x3a, 0x02, 0x38, 0x01, 0x22, 0x4a, 0x0a, 0x0e, 0x46,
	0x69, 0x65, 0x6c, 0x64, 0x41, 0x72, 0x67, 0x75, 0x6d, 0x65, 0x6e, 0x74, 0x73, 0x12, 0x12, 0x0a,
	0x04, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x6e, 0x61, 0x6d,
	0x65, 0x12, 0x0e, 0x0a, 0x02, 0x69, 0x64, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x02, 0x69,
	0x64, 0x12, 0x14, 0x0a, 0x05, 0x66, 0x69, 0x65, 0x6c, 0x64, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09,
	0x52, 0x05, 0x66, 0x69, 0x65, 0x6c, 0x64, 0x22, 0x21, 0x0a, 0x0b, 0x46, 0x69, 0x65, 0x6c, 0x64,
	0x52, 0x65, 0x74, 0x75, 0x72, 0x6e, 0x12, 0x12, 0x0a, 0x04, 0x64, 0x61, 0x74, 0x61, 0x18, 0x03,
	0x20, 0x01, 0x28, 0x0c, 0x52, 0x04, 0x64, 0x61, 0x74, 0x61, 0x32, 0xb8, 0x03, 0x0a, 0x03, 0x4d,
	0x51, 0x4c, 0x12, 0x58, 0x0a, 0x0d, 0x4c, 0x69, 0x73, 0x74, 0x52, 0x65, 0x73, 0x6f, 0x75, 0x72,
	0x63, 0x65, 0x73, 0x12, 0x1f, 0x2e, 0x6d, 0x6f, 0x6e, 0x64, 0x6f, 0x6f, 0x2e, 0x72, 0x65, 0x73,
	0x6f, 0x75, 0x72, 0x63, 0x65, 0x73, 0x2e, 0x73, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x2e, 0x45,
	0x6d, 0x70, 0x74, 0x79, 0x1a, 0x26, 0x2e, 0x6d, 0x6f, 0x6e, 0x64, 0x6f, 0x6f, 0x2e, 0x72, 0x65,
	0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x73, 0x2e, 0x73, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x2e,
	0x52, 0x65, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x4c, 0x69, 0x73, 0x74, 0x12, 0x46, 0x0a, 0x09,
	0x47, 0x65, 0x74, 0x53, 0x63, 0x68, 0x65, 0x6d, 0x61, 0x12, 0x1f, 0x2e, 0x6d, 0x6f, 0x6e, 0x64,
	0x6f, 0x6f, 0x2e, 0x72, 0x65, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x73, 0x2e, 0x73, 0x65, 0x72,
	0x76, 0x69, 0x63, 0x65, 0x2e, 0x45, 0x6d, 0x70, 0x74, 0x79, 0x1a, 0x18, 0x2e, 0x6d, 0x6f, 0x6e,
	0x64, 0x6f, 0x6f, 0x2e, 0x72, 0x65, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x73, 0x2e, 0x53, 0x63,
	0x68, 0x65, 0x6d, 0x61, 0x12, 0x55, 0x0a, 0x0a, 0x4c, 0x69, 0x73, 0x74, 0x46, 0x69, 0x65, 0x6c,
	0x64, 0x73, 0x12, 0x25, 0x2e, 0x6d, 0x6f, 0x6e, 0x64, 0x6f, 0x6f, 0x2e, 0x72, 0x65, 0x73, 0x6f,
	0x75, 0x72, 0x63, 0x65, 0x73, 0x2e, 0x73, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x2e, 0x46, 0x69,
	0x65, 0x6c, 0x64, 0x73, 0x51, 0x75, 0x65, 0x72, 0x79, 0x1a, 0x20, 0x2e, 0x6d, 0x6f, 0x6e, 0x64,
	0x6f, 0x6f, 0x2e, 0x72, 0x65, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x73, 0x2e, 0x73, 0x65, 0x72,
	0x76, 0x69, 0x63, 0x65, 0x2e, 0x46, 0x69, 0x65, 0x6c, 0x64, 0x73, 0x12, 0x5b, 0x0a, 0x0e, 0x43,
	0x72, 0x65, 0x61, 0x74, 0x65, 0x52, 0x65, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x12, 0x2b, 0x2e,
	0x6d, 0x6f, 0x6e, 0x64, 0x6f, 0x6f, 0x2e, 0x72, 0x65, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x73,
	0x2e, 0x73, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x2e, 0x52, 0x65, 0x73, 0x6f, 0x75, 0x72, 0x63,
	0x65, 0x41, 0x72, 0x67, 0x75, 0x6d, 0x65, 0x6e, 0x74, 0x73, 0x1a, 0x1c, 0x2e, 0x6d, 0x6f, 0x6e,
	0x64, 0x6f, 0x6f, 0x2e, 0x72, 0x65, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x73, 0x2e, 0x52, 0x65,
	0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x49, 0x44, 0x12, 0x5b, 0x0a, 0x08, 0x47, 0x65, 0x74, 0x46,
	0x69, 0x65, 0x6c, 0x64, 0x12, 0x28, 0x2e, 0x6d, 0x6f, 0x6e, 0x64, 0x6f, 0x6f, 0x2e, 0x72, 0x65,
	0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x73, 0x2e, 0x73, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x2e,
	0x46, 0x69, 0x65, 0x6c, 0x64, 0x41, 0x72, 0x67, 0x75, 0x6d, 0x65, 0x6e, 0x74, 0x73, 0x1a, 0x25,
	0x2e, 0x6d, 0x6f, 0x6e, 0x64, 0x6f, 0x6f, 0x2e, 0x72, 0x65, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65,
	0x73, 0x2e, 0x73, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x2e, 0x46, 0x69, 0x65, 0x6c, 0x64, 0x52,
	0x65, 0x74, 0x75, 0x72, 0x6e, 0x42, 0x29, 0x5a, 0x27, 0x67, 0x6f, 0x2e, 0x6d, 0x6f, 0x6e, 0x64,
	0x6f, 0x6f, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x63, 0x6e, 0x71, 0x75, 0x65, 0x72, 0x79, 0x2f, 0x72,
	0x65, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x73, 0x2f, 0x73, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65,
	0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_service_proto_rawDescOnce sync.Once
	file_service_proto_rawDescData = file_service_proto_rawDesc
)

func file_service_proto_rawDescGZIP() []byte {
	file_service_proto_rawDescOnce.Do(func() {
		file_service_proto_rawDescData = protoimpl.X.CompressGZIP(file_service_proto_rawDescData)
	})
	return file_service_proto_rawDescData
}

var file_service_proto_msgTypes = make([]protoimpl.MessageInfo, 9)
var file_service_proto_goTypes = []interface{}{
	(*Empty)(nil),                // 0: mondoo.resources.service.Empty
	(*ResourceList)(nil),         // 1: mondoo.resources.service.ResourceList
	(*Fields)(nil),               // 2: mondoo.resources.service.Fields
	(*FieldsQuery)(nil),          // 3: mondoo.resources.service.FieldsQuery
	(*ResourceArguments)(nil),    // 4: mondoo.resources.service.ResourceArguments
	(*FieldArguments)(nil),       // 5: mondoo.resources.service.FieldArguments
	(*FieldReturn)(nil),          // 6: mondoo.resources.service.FieldReturn
	nil,                          // 7: mondoo.resources.service.Fields.FieldsEntry
	nil,                          // 8: mondoo.resources.service.ResourceArguments.NamedEntry
	(*resources.Field)(nil),      // 9: mondoo.resources.Field
	(*resources.Schema)(nil),     // 10: mondoo.resources.Schema
	(*resources.ResourceID)(nil), // 11: mondoo.resources.ResourceID
}
var file_service_proto_depIdxs = []int32{
	7,  // 0: mondoo.resources.service.Fields.fields:type_name -> mondoo.resources.service.Fields.FieldsEntry
	8,  // 1: mondoo.resources.service.ResourceArguments.named:type_name -> mondoo.resources.service.ResourceArguments.NamedEntry
	9,  // 2: mondoo.resources.service.Fields.FieldsEntry.value:type_name -> mondoo.resources.Field
	0,  // 3: mondoo.resources.service.MQL.ListResources:input_type -> mondoo.resources.service.Empty
	0,  // 4: mondoo.resources.service.MQL.GetSchema:input_type -> mondoo.resources.service.Empty
	3,  // 5: mondoo.resources.service.MQL.ListFields:input_type -> mondoo.resources.service.FieldsQuery
	4,  // 6: mondoo.resources.service.MQL.CreateResource:input_type -> mondoo.resources.service.ResourceArguments
	5,  // 7: mondoo.resources.service.MQL.GetField:input_type -> mondoo.resources.service.FieldArguments
	1,  // 8: mondoo.resources.service.MQL.ListResources:output_type -> mondoo.resources.service.ResourceList
	10, // 9: mondoo.resources.service.MQL.GetSchema:output_type -> mondoo.resources.Schema
	2,  // 10: mondoo.resources.service.MQL.ListFields:output_type -> mondoo.resources.service.Fields
	11, // 11: mondoo.resources.service.MQL.CreateResource:output_type -> mondoo.resources.ResourceID
	6,  // 12: mondoo.resources.service.MQL.GetField:output_type -> mondoo.resources.service.FieldReturn
	8,  // [8:13] is the sub-list for method output_type
	3,  // [3:8] is the sub-list for method input_type
	3,  // [3:3] is the sub-list for extension type_name
	3,  // [3:3] is the sub-list for extension extendee
	0,  // [0:3] is the sub-list for field type_name
}

func init() { file_service_proto_init() }
func file_service_proto_init() {
	if File_service_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_service_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
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
		file_service_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ResourceList); i {
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
		file_service_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Fields); i {
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
		file_service_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*FieldsQuery); i {
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
		file_service_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ResourceArguments); i {
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
		file_service_proto_msgTypes[5].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*FieldArguments); i {
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
		file_service_proto_msgTypes[6].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*FieldReturn); i {
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
			RawDescriptor: file_service_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   9,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_service_proto_goTypes,
		DependencyIndexes: file_service_proto_depIdxs,
		MessageInfos:      file_service_proto_msgTypes,
	}.Build()
	File_service_proto = out.File
	file_service_proto_rawDesc = nil
	file_service_proto_goTypes = nil
	file_service_proto_depIdxs = nil
}
