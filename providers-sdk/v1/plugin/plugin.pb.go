// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.30.0
// 	protoc        v4.23.4
// source: plugin.proto

package plugin

import (
	llx "go.mondoo.com/cnquery/llx"
	inventory "go.mondoo.com/cnquery/providers-sdk/v1/inventory"
	upstream "go.mondoo.com/cnquery/providers-sdk/v1/upstream"
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

type ParseCLIReq struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Connector string                    `protobuf:"bytes,1,opt,name=connector,proto3" json:"connector,omitempty"`
	Args      []string                  `protobuf:"bytes,2,rep,name=args,proto3" json:"args,omitempty"`
	Flags     map[string]*llx.Primitive `protobuf:"bytes,3,rep,name=flags,proto3" json:"flags,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
}

func (x *ParseCLIReq) Reset() {
	*x = ParseCLIReq{}
	if protoimpl.UnsafeEnabled {
		mi := &file_plugin_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ParseCLIReq) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ParseCLIReq) ProtoMessage() {}

func (x *ParseCLIReq) ProtoReflect() protoreflect.Message {
	mi := &file_plugin_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ParseCLIReq.ProtoReflect.Descriptor instead.
func (*ParseCLIReq) Descriptor() ([]byte, []int) {
	return file_plugin_proto_rawDescGZIP(), []int{0}
}

func (x *ParseCLIReq) GetConnector() string {
	if x != nil {
		return x.Connector
	}
	return ""
}

func (x *ParseCLIReq) GetArgs() []string {
	if x != nil {
		return x.Args
	}
	return nil
}

func (x *ParseCLIReq) GetFlags() map[string]*llx.Primitive {
	if x != nil {
		return x.Flags
	}
	return nil
}

type ParseCLIRes struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// inventory after parsing of CLI; no connection, no discovery, no resolution
	Asset *inventory.Asset `protobuf:"bytes,1,opt,name=asset,proto3" json:"asset,omitempty"`
}

func (x *ParseCLIRes) Reset() {
	*x = ParseCLIRes{}
	if protoimpl.UnsafeEnabled {
		mi := &file_plugin_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ParseCLIRes) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ParseCLIRes) ProtoMessage() {}

func (x *ParseCLIRes) ProtoReflect() protoreflect.Message {
	mi := &file_plugin_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ParseCLIRes.ProtoReflect.Descriptor instead.
func (*ParseCLIRes) Descriptor() ([]byte, []int) {
	return file_plugin_proto_rawDescGZIP(), []int{1}
}

func (x *ParseCLIRes) GetAsset() *inventory.Asset {
	if x != nil {
		return x.Asset
	}
	return nil
}

type ConnectReq struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Features []byte `protobuf:"bytes,2,opt,name=features,proto3" json:"features,omitempty"`
	// The one primary targeted asset for the connection
	Asset          *inventory.Asset         `protobuf:"bytes,3,opt,name=asset,proto3" json:"asset,omitempty"`
	HasRecording   bool                     `protobuf:"varint,20,opt,name=has_recording,json=hasRecording,proto3" json:"has_recording,omitempty"`
	CallbackServer uint32                   `protobuf:"varint,21,opt,name=callback_server,json=callbackServer,proto3" json:"callback_server,omitempty"`
	Upstream       *upstream.UpstreamConfig `protobuf:"bytes,22,opt,name=upstream,proto3" json:"upstream,omitempty"`
}

func (x *ConnectReq) Reset() {
	*x = ConnectReq{}
	if protoimpl.UnsafeEnabled {
		mi := &file_plugin_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ConnectReq) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ConnectReq) ProtoMessage() {}

func (x *ConnectReq) ProtoReflect() protoreflect.Message {
	mi := &file_plugin_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ConnectReq.ProtoReflect.Descriptor instead.
func (*ConnectReq) Descriptor() ([]byte, []int) {
	return file_plugin_proto_rawDescGZIP(), []int{2}
}

func (x *ConnectReq) GetFeatures() []byte {
	if x != nil {
		return x.Features
	}
	return nil
}

func (x *ConnectReq) GetAsset() *inventory.Asset {
	if x != nil {
		return x.Asset
	}
	return nil
}

func (x *ConnectReq) GetHasRecording() bool {
	if x != nil {
		return x.HasRecording
	}
	return false
}

func (x *ConnectReq) GetCallbackServer() uint32 {
	if x != nil {
		return x.CallbackServer
	}
	return 0
}

func (x *ConnectReq) GetUpstream() *upstream.UpstreamConfig {
	if x != nil {
		return x.Upstream
	}
	return nil
}

type ConnectRes struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Id   uint32 `protobuf:"varint,1,opt,name=id,proto3" json:"id,omitempty"`
	Name string `protobuf:"bytes,2,opt,name=name,proto3" json:"name,omitempty"`
	// the connected asset with additional information from this connection
	Asset *inventory.Asset `protobuf:"bytes,3,opt,name=asset,proto3" json:"asset,omitempty"`
	// inventory of other discovered assets
	Inventory *inventory.Inventory `protobuf:"bytes,4,opt,name=inventory,proto3" json:"inventory,omitempty"`
}

func (x *ConnectRes) Reset() {
	*x = ConnectRes{}
	if protoimpl.UnsafeEnabled {
		mi := &file_plugin_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ConnectRes) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ConnectRes) ProtoMessage() {}

func (x *ConnectRes) ProtoReflect() protoreflect.Message {
	mi := &file_plugin_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ConnectRes.ProtoReflect.Descriptor instead.
func (*ConnectRes) Descriptor() ([]byte, []int) {
	return file_plugin_proto_rawDescGZIP(), []int{3}
}

func (x *ConnectRes) GetId() uint32 {
	if x != nil {
		return x.Id
	}
	return 0
}

func (x *ConnectRes) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *ConnectRes) GetAsset() *inventory.Asset {
	if x != nil {
		return x.Asset
	}
	return nil
}

func (x *ConnectRes) GetInventory() *inventory.Inventory {
	if x != nil {
		return x.Inventory
	}
	return nil
}

type DataReq struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Connection uint32                    `protobuf:"varint,1,opt,name=connection,proto3" json:"connection,omitempty"`
	Resource   string                    `protobuf:"bytes,3,opt,name=resource,proto3" json:"resource,omitempty"`
	ResourceId string                    `protobuf:"bytes,4,opt,name=resource_id,json=resourceId,proto3" json:"resource_id,omitempty"`
	Field      string                    `protobuf:"bytes,5,opt,name=field,proto3" json:"field,omitempty"`
	Args       map[string]*llx.Primitive `protobuf:"bytes,6,rep,name=args,proto3" json:"args,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
}

func (x *DataReq) Reset() {
	*x = DataReq{}
	if protoimpl.UnsafeEnabled {
		mi := &file_plugin_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *DataReq) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DataReq) ProtoMessage() {}

func (x *DataReq) ProtoReflect() protoreflect.Message {
	mi := &file_plugin_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use DataReq.ProtoReflect.Descriptor instead.
func (*DataReq) Descriptor() ([]byte, []int) {
	return file_plugin_proto_rawDescGZIP(), []int{4}
}

func (x *DataReq) GetConnection() uint32 {
	if x != nil {
		return x.Connection
	}
	return 0
}

func (x *DataReq) GetResource() string {
	if x != nil {
		return x.Resource
	}
	return ""
}

func (x *DataReq) GetResourceId() string {
	if x != nil {
		return x.ResourceId
	}
	return ""
}

func (x *DataReq) GetField() string {
	if x != nil {
		return x.Field
	}
	return ""
}

func (x *DataReq) GetArgs() map[string]*llx.Primitive {
	if x != nil {
		return x.Args
	}
	return nil
}

type DataRes struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Data  *llx.Primitive `protobuf:"bytes,1,opt,name=data,proto3" json:"data,omitempty"`
	Error string         `protobuf:"bytes,2,opt,name=error,proto3" json:"error,omitempty"`
	// The ID uniquely identifies this request and all associated callbacks
	Id string `protobuf:"bytes,3,opt,name=id,proto3" json:"id,omitempty"`
}

func (x *DataRes) Reset() {
	*x = DataRes{}
	if protoimpl.UnsafeEnabled {
		mi := &file_plugin_proto_msgTypes[5]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *DataRes) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DataRes) ProtoMessage() {}

func (x *DataRes) ProtoReflect() protoreflect.Message {
	mi := &file_plugin_proto_msgTypes[5]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use DataRes.ProtoReflect.Descriptor instead.
func (*DataRes) Descriptor() ([]byte, []int) {
	return file_plugin_proto_rawDescGZIP(), []int{5}
}

func (x *DataRes) GetData() *llx.Primitive {
	if x != nil {
		return x.Data
	}
	return nil
}

func (x *DataRes) GetError() string {
	if x != nil {
		return x.Error
	}
	return ""
}

func (x *DataRes) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

type CollectRes struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
}

func (x *CollectRes) Reset() {
	*x = CollectRes{}
	if protoimpl.UnsafeEnabled {
		mi := &file_plugin_proto_msgTypes[6]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *CollectRes) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*CollectRes) ProtoMessage() {}

func (x *CollectRes) ProtoReflect() protoreflect.Message {
	mi := &file_plugin_proto_msgTypes[6]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use CollectRes.ProtoReflect.Descriptor instead.
func (*CollectRes) Descriptor() ([]byte, []int) {
	return file_plugin_proto_rawDescGZIP(), []int{6}
}

type StoreReq struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Connection uint32          `protobuf:"varint,1,opt,name=connection,proto3" json:"connection,omitempty"`
	Resources  []*ResourceData `protobuf:"bytes,2,rep,name=resources,proto3" json:"resources,omitempty"`
}

func (x *StoreReq) Reset() {
	*x = StoreReq{}
	if protoimpl.UnsafeEnabled {
		mi := &file_plugin_proto_msgTypes[7]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *StoreReq) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*StoreReq) ProtoMessage() {}

func (x *StoreReq) ProtoReflect() protoreflect.Message {
	mi := &file_plugin_proto_msgTypes[7]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use StoreReq.ProtoReflect.Descriptor instead.
func (*StoreReq) Descriptor() ([]byte, []int) {
	return file_plugin_proto_rawDescGZIP(), []int{7}
}

func (x *StoreReq) GetConnection() uint32 {
	if x != nil {
		return x.Connection
	}
	return 0
}

func (x *StoreReq) GetResources() []*ResourceData {
	if x != nil {
		return x.Resources
	}
	return nil
}

type ResourceData struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Name   string                 `protobuf:"bytes,3,opt,name=name,proto3" json:"name,omitempty"`
	Id     string                 `protobuf:"bytes,4,opt,name=id,proto3" json:"id,omitempty"`
	Fields map[string]*llx.Result `protobuf:"bytes,5,rep,name=fields,proto3" json:"fields,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
}

func (x *ResourceData) Reset() {
	*x = ResourceData{}
	if protoimpl.UnsafeEnabled {
		mi := &file_plugin_proto_msgTypes[8]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ResourceData) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ResourceData) ProtoMessage() {}

func (x *ResourceData) ProtoReflect() protoreflect.Message {
	mi := &file_plugin_proto_msgTypes[8]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ResourceData.ProtoReflect.Descriptor instead.
func (*ResourceData) Descriptor() ([]byte, []int) {
	return file_plugin_proto_rawDescGZIP(), []int{8}
}

func (x *ResourceData) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *ResourceData) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

func (x *ResourceData) GetFields() map[string]*llx.Result {
	if x != nil {
		return x.Fields
	}
	return nil
}

type StoreRes struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
}

func (x *StoreRes) Reset() {
	*x = StoreRes{}
	if protoimpl.UnsafeEnabled {
		mi := &file_plugin_proto_msgTypes[9]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *StoreRes) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*StoreRes) ProtoMessage() {}

func (x *StoreRes) ProtoReflect() protoreflect.Message {
	mi := &file_plugin_proto_msgTypes[9]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use StoreRes.ProtoReflect.Descriptor instead.
func (*StoreRes) Descriptor() ([]byte, []int) {
	return file_plugin_proto_rawDescGZIP(), []int{9}
}

var File_plugin_proto protoreflect.FileDescriptor

var file_plugin_proto_rawDesc = []byte{
	0x0a, 0x0c, 0x70, 0x6c, 0x75, 0x67, 0x69, 0x6e, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x14,
	0x63, 0x6e, 0x71, 0x75, 0x65, 0x72, 0x79, 0x2e, 0x70, 0x72, 0x6f, 0x76, 0x69, 0x64, 0x65, 0x72,
	0x73, 0x2e, 0x76, 0x31, 0x1a, 0x2a, 0x70, 0x72, 0x6f, 0x76, 0x69, 0x64, 0x65, 0x72, 0x73, 0x2d,
	0x73, 0x64, 0x6b, 0x2f, 0x76, 0x31, 0x2f, 0x69, 0x6e, 0x76, 0x65, 0x6e, 0x74, 0x6f, 0x72, 0x79,
	0x2f, 0x69, 0x6e, 0x76, 0x65, 0x6e, 0x74, 0x6f, 0x72, 0x79, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x1a, 0x28, 0x70, 0x72, 0x6f, 0x76, 0x69, 0x64, 0x65, 0x72, 0x73, 0x2d, 0x73, 0x64, 0x6b, 0x2f,
	0x76, 0x31, 0x2f, 0x75, 0x70, 0x73, 0x74, 0x72, 0x65, 0x61, 0x6d, 0x2f, 0x75, 0x70, 0x73, 0x74,
	0x72, 0x65, 0x61, 0x6d, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x0d, 0x6c, 0x6c, 0x78, 0x2f,
	0x6c, 0x6c, 0x78, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0xd5, 0x01, 0x0a, 0x0b, 0x50, 0x61,
	0x72, 0x73, 0x65, 0x43, 0x4c, 0x49, 0x52, 0x65, 0x71, 0x12, 0x1c, 0x0a, 0x09, 0x63, 0x6f, 0x6e,
	0x6e, 0x65, 0x63, 0x74, 0x6f, 0x72, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x09, 0x63, 0x6f,
	0x6e, 0x6e, 0x65, 0x63, 0x74, 0x6f, 0x72, 0x12, 0x12, 0x0a, 0x04, 0x61, 0x72, 0x67, 0x73, 0x18,
	0x02, 0x20, 0x03, 0x28, 0x09, 0x52, 0x04, 0x61, 0x72, 0x67, 0x73, 0x12, 0x42, 0x0a, 0x05, 0x66,
	0x6c, 0x61, 0x67, 0x73, 0x18, 0x03, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x2c, 0x2e, 0x63, 0x6e, 0x71,
	0x75, 0x65, 0x72, 0x79, 0x2e, 0x70, 0x72, 0x6f, 0x76, 0x69, 0x64, 0x65, 0x72, 0x73, 0x2e, 0x76,
	0x31, 0x2e, 0x50, 0x61, 0x72, 0x73, 0x65, 0x43, 0x4c, 0x49, 0x52, 0x65, 0x71, 0x2e, 0x46, 0x6c,
	0x61, 0x67, 0x73, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x52, 0x05, 0x66, 0x6c, 0x61, 0x67, 0x73, 0x1a,
	0x50, 0x0a, 0x0a, 0x46, 0x6c, 0x61, 0x67, 0x73, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x12, 0x10, 0x0a,
	0x03, 0x6b, 0x65, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x6b, 0x65, 0x79, 0x12,
	0x2c, 0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x16,
	0x2e, 0x63, 0x6e, 0x71, 0x75, 0x65, 0x72, 0x79, 0x2e, 0x6c, 0x6c, 0x78, 0x2e, 0x50, 0x72, 0x69,
	0x6d, 0x69, 0x74, 0x69, 0x76, 0x65, 0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x3a, 0x02, 0x38,
	0x01, 0x22, 0x40, 0x0a, 0x0b, 0x50, 0x61, 0x72, 0x73, 0x65, 0x43, 0x4c, 0x49, 0x52, 0x65, 0x73,
	0x12, 0x31, 0x0a, 0x05, 0x61, 0x73, 0x73, 0x65, 0x74, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32,
	0x1b, 0x2e, 0x63, 0x6e, 0x71, 0x75, 0x65, 0x72, 0x79, 0x2e, 0x70, 0x72, 0x6f, 0x76, 0x69, 0x64,
	0x65, 0x72, 0x73, 0x2e, 0x76, 0x31, 0x2e, 0x41, 0x73, 0x73, 0x65, 0x74, 0x52, 0x05, 0x61, 0x73,
	0x73, 0x65, 0x74, 0x22, 0xf1, 0x01, 0x0a, 0x0a, 0x43, 0x6f, 0x6e, 0x6e, 0x65, 0x63, 0x74, 0x52,
	0x65, 0x71, 0x12, 0x1a, 0x0a, 0x08, 0x66, 0x65, 0x61, 0x74, 0x75, 0x72, 0x65, 0x73, 0x18, 0x02,
	0x20, 0x01, 0x28, 0x0c, 0x52, 0x08, 0x66, 0x65, 0x61, 0x74, 0x75, 0x72, 0x65, 0x73, 0x12, 0x31,
	0x0a, 0x05, 0x61, 0x73, 0x73, 0x65, 0x74, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1b, 0x2e,
	0x63, 0x6e, 0x71, 0x75, 0x65, 0x72, 0x79, 0x2e, 0x70, 0x72, 0x6f, 0x76, 0x69, 0x64, 0x65, 0x72,
	0x73, 0x2e, 0x76, 0x31, 0x2e, 0x41, 0x73, 0x73, 0x65, 0x74, 0x52, 0x05, 0x61, 0x73, 0x73, 0x65,
	0x74, 0x12, 0x23, 0x0a, 0x0d, 0x68, 0x61, 0x73, 0x5f, 0x72, 0x65, 0x63, 0x6f, 0x72, 0x64, 0x69,
	0x6e, 0x67, 0x18, 0x14, 0x20, 0x01, 0x28, 0x08, 0x52, 0x0c, 0x68, 0x61, 0x73, 0x52, 0x65, 0x63,
	0x6f, 0x72, 0x64, 0x69, 0x6e, 0x67, 0x12, 0x27, 0x0a, 0x0f, 0x63, 0x61, 0x6c, 0x6c, 0x62, 0x61,
	0x63, 0x6b, 0x5f, 0x73, 0x65, 0x72, 0x76, 0x65, 0x72, 0x18, 0x15, 0x20, 0x01, 0x28, 0x0d, 0x52,
	0x0e, 0x63, 0x61, 0x6c, 0x6c, 0x62, 0x61, 0x63, 0x6b, 0x53, 0x65, 0x72, 0x76, 0x65, 0x72, 0x12,
	0x46, 0x0a, 0x08, 0x75, 0x70, 0x73, 0x74, 0x72, 0x65, 0x61, 0x6d, 0x18, 0x16, 0x20, 0x01, 0x28,
	0x0b, 0x32, 0x2a, 0x2e, 0x6d, 0x6f, 0x6e, 0x64, 0x6f, 0x6f, 0x2e, 0x63, 0x6e, 0x71, 0x75, 0x65,
	0x72, 0x79, 0x2e, 0x75, 0x70, 0x73, 0x74, 0x72, 0x65, 0x61, 0x6d, 0x2e, 0x76, 0x31, 0x2e, 0x55,
	0x70, 0x73, 0x74, 0x72, 0x65, 0x61, 0x6d, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x52, 0x08, 0x75,
	0x70, 0x73, 0x74, 0x72, 0x65, 0x61, 0x6d, 0x22, 0xa2, 0x01, 0x0a, 0x0a, 0x43, 0x6f, 0x6e, 0x6e,
	0x65, 0x63, 0x74, 0x52, 0x65, 0x73, 0x12, 0x0e, 0x0a, 0x02, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01,
	0x28, 0x0d, 0x52, 0x02, 0x69, 0x64, 0x12, 0x12, 0x0a, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x02,
	0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x12, 0x31, 0x0a, 0x05, 0x61, 0x73,
	0x73, 0x65, 0x74, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1b, 0x2e, 0x63, 0x6e, 0x71, 0x75,
	0x65, 0x72, 0x79, 0x2e, 0x70, 0x72, 0x6f, 0x76, 0x69, 0x64, 0x65, 0x72, 0x73, 0x2e, 0x76, 0x31,
	0x2e, 0x41, 0x73, 0x73, 0x65, 0x74, 0x52, 0x05, 0x61, 0x73, 0x73, 0x65, 0x74, 0x12, 0x3d, 0x0a,
	0x09, 0x69, 0x6e, 0x76, 0x65, 0x6e, 0x74, 0x6f, 0x72, 0x79, 0x18, 0x04, 0x20, 0x01, 0x28, 0x0b,
	0x32, 0x1f, 0x2e, 0x63, 0x6e, 0x71, 0x75, 0x65, 0x72, 0x79, 0x2e, 0x70, 0x72, 0x6f, 0x76, 0x69,
	0x64, 0x65, 0x72, 0x73, 0x2e, 0x76, 0x31, 0x2e, 0x49, 0x6e, 0x76, 0x65, 0x6e, 0x74, 0x6f, 0x72,
	0x79, 0x52, 0x09, 0x69, 0x6e, 0x76, 0x65, 0x6e, 0x74, 0x6f, 0x72, 0x79, 0x22, 0x8a, 0x02, 0x0a,
	0x07, 0x44, 0x61, 0x74, 0x61, 0x52, 0x65, 0x71, 0x12, 0x1e, 0x0a, 0x0a, 0x63, 0x6f, 0x6e, 0x6e,
	0x65, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0d, 0x52, 0x0a, 0x63, 0x6f,
	0x6e, 0x6e, 0x65, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x12, 0x1a, 0x0a, 0x08, 0x72, 0x65, 0x73, 0x6f,
	0x75, 0x72, 0x63, 0x65, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x08, 0x72, 0x65, 0x73, 0x6f,
	0x75, 0x72, 0x63, 0x65, 0x12, 0x1f, 0x0a, 0x0b, 0x72, 0x65, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65,
	0x5f, 0x69, 0x64, 0x18, 0x04, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0a, 0x72, 0x65, 0x73, 0x6f, 0x75,
	0x72, 0x63, 0x65, 0x49, 0x64, 0x12, 0x14, 0x0a, 0x05, 0x66, 0x69, 0x65, 0x6c, 0x64, 0x18, 0x05,
	0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x66, 0x69, 0x65, 0x6c, 0x64, 0x12, 0x3b, 0x0a, 0x04, 0x61,
	0x72, 0x67, 0x73, 0x18, 0x06, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x27, 0x2e, 0x63, 0x6e, 0x71, 0x75,
	0x65, 0x72, 0x79, 0x2e, 0x70, 0x72, 0x6f, 0x76, 0x69, 0x64, 0x65, 0x72, 0x73, 0x2e, 0x76, 0x31,
	0x2e, 0x44, 0x61, 0x74, 0x61, 0x52, 0x65, 0x71, 0x2e, 0x41, 0x72, 0x67, 0x73, 0x45, 0x6e, 0x74,
	0x72, 0x79, 0x52, 0x04, 0x61, 0x72, 0x67, 0x73, 0x1a, 0x4f, 0x0a, 0x09, 0x41, 0x72, 0x67, 0x73,
	0x45, 0x6e, 0x74, 0x72, 0x79, 0x12, 0x10, 0x0a, 0x03, 0x6b, 0x65, 0x79, 0x18, 0x01, 0x20, 0x01,
	0x28, 0x09, 0x52, 0x03, 0x6b, 0x65, 0x79, 0x12, 0x2c, 0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65,
	0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x16, 0x2e, 0x63, 0x6e, 0x71, 0x75, 0x65, 0x72, 0x79,
	0x2e, 0x6c, 0x6c, 0x78, 0x2e, 0x50, 0x72, 0x69, 0x6d, 0x69, 0x74, 0x69, 0x76, 0x65, 0x52, 0x05,
	0x76, 0x61, 0x6c, 0x75, 0x65, 0x3a, 0x02, 0x38, 0x01, 0x22, 0x5b, 0x0a, 0x07, 0x44, 0x61, 0x74,
	0x61, 0x52, 0x65, 0x73, 0x12, 0x2a, 0x0a, 0x04, 0x64, 0x61, 0x74, 0x61, 0x18, 0x01, 0x20, 0x01,
	0x28, 0x0b, 0x32, 0x16, 0x2e, 0x63, 0x6e, 0x71, 0x75, 0x65, 0x72, 0x79, 0x2e, 0x6c, 0x6c, 0x78,
	0x2e, 0x50, 0x72, 0x69, 0x6d, 0x69, 0x74, 0x69, 0x76, 0x65, 0x52, 0x04, 0x64, 0x61, 0x74, 0x61,
	0x12, 0x14, 0x0a, 0x05, 0x65, 0x72, 0x72, 0x6f, 0x72, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52,
	0x05, 0x65, 0x72, 0x72, 0x6f, 0x72, 0x12, 0x0e, 0x0a, 0x02, 0x69, 0x64, 0x18, 0x03, 0x20, 0x01,
	0x28, 0x09, 0x52, 0x02, 0x69, 0x64, 0x22, 0x0c, 0x0a, 0x0a, 0x43, 0x6f, 0x6c, 0x6c, 0x65, 0x63,
	0x74, 0x52, 0x65, 0x73, 0x22, 0x6c, 0x0a, 0x08, 0x53, 0x74, 0x6f, 0x72, 0x65, 0x52, 0x65, 0x71,
	0x12, 0x1e, 0x0a, 0x0a, 0x63, 0x6f, 0x6e, 0x6e, 0x65, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x18, 0x01,
	0x20, 0x01, 0x28, 0x0d, 0x52, 0x0a, 0x63, 0x6f, 0x6e, 0x6e, 0x65, 0x63, 0x74, 0x69, 0x6f, 0x6e,
	0x12, 0x40, 0x0a, 0x09, 0x72, 0x65, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x73, 0x18, 0x02, 0x20,
	0x03, 0x28, 0x0b, 0x32, 0x22, 0x2e, 0x63, 0x6e, 0x71, 0x75, 0x65, 0x72, 0x79, 0x2e, 0x70, 0x72,
	0x6f, 0x76, 0x69, 0x64, 0x65, 0x72, 0x73, 0x2e, 0x76, 0x31, 0x2e, 0x52, 0x65, 0x73, 0x6f, 0x75,
	0x72, 0x63, 0x65, 0x44, 0x61, 0x74, 0x61, 0x52, 0x09, 0x72, 0x65, 0x73, 0x6f, 0x75, 0x72, 0x63,
	0x65, 0x73, 0x22, 0xca, 0x01, 0x0a, 0x0c, 0x52, 0x65, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x44,
	0x61, 0x74, 0x61, 0x12, 0x12, 0x0a, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x03, 0x20, 0x01, 0x28,
	0x09, 0x52, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x12, 0x0e, 0x0a, 0x02, 0x69, 0x64, 0x18, 0x04, 0x20,
	0x01, 0x28, 0x09, 0x52, 0x02, 0x69, 0x64, 0x12, 0x46, 0x0a, 0x06, 0x66, 0x69, 0x65, 0x6c, 0x64,
	0x73, 0x18, 0x05, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x2e, 0x2e, 0x63, 0x6e, 0x71, 0x75, 0x65, 0x72,
	0x79, 0x2e, 0x70, 0x72, 0x6f, 0x76, 0x69, 0x64, 0x65, 0x72, 0x73, 0x2e, 0x76, 0x31, 0x2e, 0x52,
	0x65, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x44, 0x61, 0x74, 0x61, 0x2e, 0x46, 0x69, 0x65, 0x6c,
	0x64, 0x73, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x52, 0x06, 0x66, 0x69, 0x65, 0x6c, 0x64, 0x73, 0x1a,
	0x4e, 0x0a, 0x0b, 0x46, 0x69, 0x65, 0x6c, 0x64, 0x73, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x12, 0x10,
	0x0a, 0x03, 0x6b, 0x65, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x6b, 0x65, 0x79,
	0x12, 0x29, 0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32,
	0x13, 0x2e, 0x63, 0x6e, 0x71, 0x75, 0x65, 0x72, 0x79, 0x2e, 0x6c, 0x6c, 0x78, 0x2e, 0x52, 0x65,
	0x73, 0x75, 0x6c, 0x74, 0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x3a, 0x02, 0x38, 0x01, 0x22,
	0x0a, 0x0a, 0x08, 0x53, 0x74, 0x6f, 0x72, 0x65, 0x52, 0x65, 0x73, 0x32, 0xc7, 0x02, 0x0a, 0x0e,
	0x50, 0x72, 0x6f, 0x76, 0x69, 0x64, 0x65, 0x72, 0x50, 0x6c, 0x75, 0x67, 0x69, 0x6e, 0x12, 0x50,
	0x0a, 0x08, 0x50, 0x61, 0x72, 0x73, 0x65, 0x43, 0x4c, 0x49, 0x12, 0x21, 0x2e, 0x63, 0x6e, 0x71,
	0x75, 0x65, 0x72, 0x79, 0x2e, 0x70, 0x72, 0x6f, 0x76, 0x69, 0x64, 0x65, 0x72, 0x73, 0x2e, 0x76,
	0x31, 0x2e, 0x50, 0x61, 0x72, 0x73, 0x65, 0x43, 0x4c, 0x49, 0x52, 0x65, 0x71, 0x1a, 0x21, 0x2e,
	0x63, 0x6e, 0x71, 0x75, 0x65, 0x72, 0x79, 0x2e, 0x70, 0x72, 0x6f, 0x76, 0x69, 0x64, 0x65, 0x72,
	0x73, 0x2e, 0x76, 0x31, 0x2e, 0x50, 0x61, 0x72, 0x73, 0x65, 0x43, 0x4c, 0x49, 0x52, 0x65, 0x73,
	0x12, 0x4d, 0x0a, 0x07, 0x43, 0x6f, 0x6e, 0x6e, 0x65, 0x63, 0x74, 0x12, 0x20, 0x2e, 0x63, 0x6e,
	0x71, 0x75, 0x65, 0x72, 0x79, 0x2e, 0x70, 0x72, 0x6f, 0x76, 0x69, 0x64, 0x65, 0x72, 0x73, 0x2e,
	0x76, 0x31, 0x2e, 0x43, 0x6f, 0x6e, 0x6e, 0x65, 0x63, 0x74, 0x52, 0x65, 0x71, 0x1a, 0x20, 0x2e,
	0x63, 0x6e, 0x71, 0x75, 0x65, 0x72, 0x79, 0x2e, 0x70, 0x72, 0x6f, 0x76, 0x69, 0x64, 0x65, 0x72,
	0x73, 0x2e, 0x76, 0x31, 0x2e, 0x43, 0x6f, 0x6e, 0x6e, 0x65, 0x63, 0x74, 0x52, 0x65, 0x73, 0x12,
	0x47, 0x0a, 0x07, 0x47, 0x65, 0x74, 0x44, 0x61, 0x74, 0x61, 0x12, 0x1d, 0x2e, 0x63, 0x6e, 0x71,
	0x75, 0x65, 0x72, 0x79, 0x2e, 0x70, 0x72, 0x6f, 0x76, 0x69, 0x64, 0x65, 0x72, 0x73, 0x2e, 0x76,
	0x31, 0x2e, 0x44, 0x61, 0x74, 0x61, 0x52, 0x65, 0x71, 0x1a, 0x1d, 0x2e, 0x63, 0x6e, 0x71, 0x75,
	0x65, 0x72, 0x79, 0x2e, 0x70, 0x72, 0x6f, 0x76, 0x69, 0x64, 0x65, 0x72, 0x73, 0x2e, 0x76, 0x31,
	0x2e, 0x44, 0x61, 0x74, 0x61, 0x52, 0x65, 0x73, 0x12, 0x4b, 0x0a, 0x09, 0x53, 0x74, 0x6f, 0x72,
	0x65, 0x44, 0x61, 0x74, 0x61, 0x12, 0x1e, 0x2e, 0x63, 0x6e, 0x71, 0x75, 0x65, 0x72, 0x79, 0x2e,
	0x70, 0x72, 0x6f, 0x76, 0x69, 0x64, 0x65, 0x72, 0x73, 0x2e, 0x76, 0x31, 0x2e, 0x53, 0x74, 0x6f,
	0x72, 0x65, 0x52, 0x65, 0x71, 0x1a, 0x1e, 0x2e, 0x63, 0x6e, 0x71, 0x75, 0x65, 0x72, 0x79, 0x2e,
	0x70, 0x72, 0x6f, 0x76, 0x69, 0x64, 0x65, 0x72, 0x73, 0x2e, 0x76, 0x31, 0x2e, 0x53, 0x74, 0x6f,
	0x72, 0x65, 0x52, 0x65, 0x73, 0x32, 0xfa, 0x01, 0x0a, 0x10, 0x50, 0x72, 0x6f, 0x76, 0x69, 0x64,
	0x65, 0x72, 0x43, 0x61, 0x6c, 0x6c, 0x62, 0x61, 0x63, 0x6b, 0x12, 0x4a, 0x0a, 0x07, 0x43, 0x6f,
	0x6c, 0x6c, 0x65, 0x63, 0x74, 0x12, 0x1d, 0x2e, 0x63, 0x6e, 0x71, 0x75, 0x65, 0x72, 0x79, 0x2e,
	0x70, 0x72, 0x6f, 0x76, 0x69, 0x64, 0x65, 0x72, 0x73, 0x2e, 0x76, 0x31, 0x2e, 0x44, 0x61, 0x74,
	0x61, 0x52, 0x65, 0x73, 0x1a, 0x20, 0x2e, 0x63, 0x6e, 0x71, 0x75, 0x65, 0x72, 0x79, 0x2e, 0x70,
	0x72, 0x6f, 0x76, 0x69, 0x64, 0x65, 0x72, 0x73, 0x2e, 0x76, 0x31, 0x2e, 0x43, 0x6f, 0x6c, 0x6c,
	0x65, 0x63, 0x74, 0x52, 0x65, 0x73, 0x12, 0x51, 0x0a, 0x0c, 0x47, 0x65, 0x74, 0x52, 0x65, 0x63,
	0x6f, 0x72, 0x64, 0x69, 0x6e, 0x67, 0x12, 0x1d, 0x2e, 0x63, 0x6e, 0x71, 0x75, 0x65, 0x72, 0x79,
	0x2e, 0x70, 0x72, 0x6f, 0x76, 0x69, 0x64, 0x65, 0x72, 0x73, 0x2e, 0x76, 0x31, 0x2e, 0x44, 0x61,
	0x74, 0x61, 0x52, 0x65, 0x71, 0x1a, 0x22, 0x2e, 0x63, 0x6e, 0x71, 0x75, 0x65, 0x72, 0x79, 0x2e,
	0x70, 0x72, 0x6f, 0x76, 0x69, 0x64, 0x65, 0x72, 0x73, 0x2e, 0x76, 0x31, 0x2e, 0x52, 0x65, 0x73,
	0x6f, 0x75, 0x72, 0x63, 0x65, 0x44, 0x61, 0x74, 0x61, 0x12, 0x47, 0x0a, 0x07, 0x47, 0x65, 0x74,
	0x44, 0x61, 0x74, 0x61, 0x12, 0x1d, 0x2e, 0x63, 0x6e, 0x71, 0x75, 0x65, 0x72, 0x79, 0x2e, 0x70,
	0x72, 0x6f, 0x76, 0x69, 0x64, 0x65, 0x72, 0x73, 0x2e, 0x76, 0x31, 0x2e, 0x44, 0x61, 0x74, 0x61,
	0x52, 0x65, 0x71, 0x1a, 0x1d, 0x2e, 0x63, 0x6e, 0x71, 0x75, 0x65, 0x72, 0x79, 0x2e, 0x70, 0x72,
	0x6f, 0x76, 0x69, 0x64, 0x65, 0x72, 0x73, 0x2e, 0x76, 0x31, 0x2e, 0x44, 0x61, 0x74, 0x61, 0x52,
	0x65, 0x73, 0x42, 0x2f, 0x5a, 0x2d, 0x67, 0x6f, 0x2e, 0x6d, 0x6f, 0x6e, 0x64, 0x6f, 0x6f, 0x2e,
	0x63, 0x6f, 0x6d, 0x2f, 0x63, 0x6e, 0x71, 0x75, 0x65, 0x72, 0x79, 0x2f, 0x70, 0x72, 0x6f, 0x76,
	0x69, 0x64, 0x65, 0x72, 0x73, 0x2d, 0x73, 0x64, 0x6b, 0x2f, 0x76, 0x31, 0x2f, 0x70, 0x6c, 0x75,
	0x67, 0x69, 0x6e, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_plugin_proto_rawDescOnce sync.Once
	file_plugin_proto_rawDescData = file_plugin_proto_rawDesc
)

func file_plugin_proto_rawDescGZIP() []byte {
	file_plugin_proto_rawDescOnce.Do(func() {
		file_plugin_proto_rawDescData = protoimpl.X.CompressGZIP(file_plugin_proto_rawDescData)
	})
	return file_plugin_proto_rawDescData
}

var file_plugin_proto_msgTypes = make([]protoimpl.MessageInfo, 13)
var file_plugin_proto_goTypes = []interface{}{
	(*ParseCLIReq)(nil),             // 0: cnquery.providers.v1.ParseCLIReq
	(*ParseCLIRes)(nil),             // 1: cnquery.providers.v1.ParseCLIRes
	(*ConnectReq)(nil),              // 2: cnquery.providers.v1.ConnectReq
	(*ConnectRes)(nil),              // 3: cnquery.providers.v1.ConnectRes
	(*DataReq)(nil),                 // 4: cnquery.providers.v1.DataReq
	(*DataRes)(nil),                 // 5: cnquery.providers.v1.DataRes
	(*CollectRes)(nil),              // 6: cnquery.providers.v1.CollectRes
	(*StoreReq)(nil),                // 7: cnquery.providers.v1.StoreReq
	(*ResourceData)(nil),            // 8: cnquery.providers.v1.ResourceData
	(*StoreRes)(nil),                // 9: cnquery.providers.v1.StoreRes
	nil,                             // 10: cnquery.providers.v1.ParseCLIReq.FlagsEntry
	nil,                             // 11: cnquery.providers.v1.DataReq.ArgsEntry
	nil,                             // 12: cnquery.providers.v1.ResourceData.FieldsEntry
	(*inventory.Asset)(nil),         // 13: cnquery.providers.v1.Asset
	(*upstream.UpstreamConfig)(nil), // 14: mondoo.cnquery.upstream.v1.UpstreamConfig
	(*inventory.Inventory)(nil),     // 15: cnquery.providers.v1.Inventory
	(*llx.Primitive)(nil),           // 16: cnquery.llx.Primitive
	(*llx.Result)(nil),              // 17: cnquery.llx.Result
}
var file_plugin_proto_depIdxs = []int32{
	10, // 0: cnquery.providers.v1.ParseCLIReq.flags:type_name -> cnquery.providers.v1.ParseCLIReq.FlagsEntry
	13, // 1: cnquery.providers.v1.ParseCLIRes.asset:type_name -> cnquery.providers.v1.Asset
	13, // 2: cnquery.providers.v1.ConnectReq.asset:type_name -> cnquery.providers.v1.Asset
	14, // 3: cnquery.providers.v1.ConnectReq.upstream:type_name -> mondoo.cnquery.upstream.v1.UpstreamConfig
	13, // 4: cnquery.providers.v1.ConnectRes.asset:type_name -> cnquery.providers.v1.Asset
	15, // 5: cnquery.providers.v1.ConnectRes.inventory:type_name -> cnquery.providers.v1.Inventory
	11, // 6: cnquery.providers.v1.DataReq.args:type_name -> cnquery.providers.v1.DataReq.ArgsEntry
	16, // 7: cnquery.providers.v1.DataRes.data:type_name -> cnquery.llx.Primitive
	8,  // 8: cnquery.providers.v1.StoreReq.resources:type_name -> cnquery.providers.v1.ResourceData
	12, // 9: cnquery.providers.v1.ResourceData.fields:type_name -> cnquery.providers.v1.ResourceData.FieldsEntry
	16, // 10: cnquery.providers.v1.ParseCLIReq.FlagsEntry.value:type_name -> cnquery.llx.Primitive
	16, // 11: cnquery.providers.v1.DataReq.ArgsEntry.value:type_name -> cnquery.llx.Primitive
	17, // 12: cnquery.providers.v1.ResourceData.FieldsEntry.value:type_name -> cnquery.llx.Result
	0,  // 13: cnquery.providers.v1.ProviderPlugin.ParseCLI:input_type -> cnquery.providers.v1.ParseCLIReq
	2,  // 14: cnquery.providers.v1.ProviderPlugin.Connect:input_type -> cnquery.providers.v1.ConnectReq
	4,  // 15: cnquery.providers.v1.ProviderPlugin.GetData:input_type -> cnquery.providers.v1.DataReq
	7,  // 16: cnquery.providers.v1.ProviderPlugin.StoreData:input_type -> cnquery.providers.v1.StoreReq
	5,  // 17: cnquery.providers.v1.ProviderCallback.Collect:input_type -> cnquery.providers.v1.DataRes
	4,  // 18: cnquery.providers.v1.ProviderCallback.GetRecording:input_type -> cnquery.providers.v1.DataReq
	4,  // 19: cnquery.providers.v1.ProviderCallback.GetData:input_type -> cnquery.providers.v1.DataReq
	1,  // 20: cnquery.providers.v1.ProviderPlugin.ParseCLI:output_type -> cnquery.providers.v1.ParseCLIRes
	3,  // 21: cnquery.providers.v1.ProviderPlugin.Connect:output_type -> cnquery.providers.v1.ConnectRes
	5,  // 22: cnquery.providers.v1.ProviderPlugin.GetData:output_type -> cnquery.providers.v1.DataRes
	9,  // 23: cnquery.providers.v1.ProviderPlugin.StoreData:output_type -> cnquery.providers.v1.StoreRes
	6,  // 24: cnquery.providers.v1.ProviderCallback.Collect:output_type -> cnquery.providers.v1.CollectRes
	8,  // 25: cnquery.providers.v1.ProviderCallback.GetRecording:output_type -> cnquery.providers.v1.ResourceData
	5,  // 26: cnquery.providers.v1.ProviderCallback.GetData:output_type -> cnquery.providers.v1.DataRes
	20, // [20:27] is the sub-list for method output_type
	13, // [13:20] is the sub-list for method input_type
	13, // [13:13] is the sub-list for extension type_name
	13, // [13:13] is the sub-list for extension extendee
	0,  // [0:13] is the sub-list for field type_name
}

func init() { file_plugin_proto_init() }
func file_plugin_proto_init() {
	if File_plugin_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_plugin_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ParseCLIReq); i {
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
		file_plugin_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ParseCLIRes); i {
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
		file_plugin_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ConnectReq); i {
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
		file_plugin_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ConnectRes); i {
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
		file_plugin_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*DataReq); i {
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
		file_plugin_proto_msgTypes[5].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*DataRes); i {
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
		file_plugin_proto_msgTypes[6].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*CollectRes); i {
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
		file_plugin_proto_msgTypes[7].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*StoreReq); i {
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
		file_plugin_proto_msgTypes[8].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ResourceData); i {
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
		file_plugin_proto_msgTypes[9].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*StoreRes); i {
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
			RawDescriptor: file_plugin_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   13,
			NumExtensions: 0,
			NumServices:   2,
		},
		GoTypes:           file_plugin_proto_goTypes,
		DependencyIndexes: file_plugin_proto_depIdxs,
		MessageInfos:      file_plugin_proto_msgTypes,
	}.Build()
	File_plugin_proto = out.File
	file_plugin_proto_rawDesc = nil
	file_plugin_proto_goTypes = nil
	file_plugin_proto_depIdxs = nil
}
