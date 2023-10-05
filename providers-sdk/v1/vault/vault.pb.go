// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.31.0
// 	protoc        v4.24.3
// source: vault.proto

package vault

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

type CredentialType int32

const (
	CredentialType_undefined                CredentialType = 0
	CredentialType_password                 CredentialType = 1
	CredentialType_private_key              CredentialType = 2
	CredentialType_ssh_agent                CredentialType = 3
	CredentialType_bearer                   CredentialType = 4
	CredentialType_credentials_query        CredentialType = 5
	CredentialType_json                     CredentialType = 6
	CredentialType_aws_ec2_instance_connect CredentialType = 7
	CredentialType_aws_ec2_ssm_session      CredentialType = 8
	CredentialType_pkcs12                   CredentialType = 9
)

// Enum value maps for CredentialType.
var (
	CredentialType_name = map[int32]string{
		0: "undefined",
		1: "password",
		2: "private_key",
		3: "ssh_agent",
		4: "bearer",
		5: "credentials_query",
		6: "json",
		7: "aws_ec2_instance_connect",
		8: "aws_ec2_ssm_session",
		9: "pkcs12",
	}
	CredentialType_value = map[string]int32{
		"undefined":                0,
		"password":                 1,
		"private_key":              2,
		"ssh_agent":                3,
		"bearer":                   4,
		"credentials_query":        5,
		"json":                     6,
		"aws_ec2_instance_connect": 7,
		"aws_ec2_ssm_session":      8,
		"pkcs12":                   9,
	}
)

func (x CredentialType) Enum() *CredentialType {
	p := new(CredentialType)
	*p = x
	return p
}

func (x CredentialType) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (CredentialType) Descriptor() protoreflect.EnumDescriptor {
	return file_vault_proto_enumTypes[0].Descriptor()
}

func (CredentialType) Type() protoreflect.EnumType {
	return &file_vault_proto_enumTypes[0]
}

func (x CredentialType) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use CredentialType.Descriptor instead.
func (CredentialType) EnumDescriptor() ([]byte, []int) {
	return file_vault_proto_rawDescGZIP(), []int{0}
}

type SecretEncoding int32

const (
	SecretEncoding_encoding_undefined SecretEncoding = 0
	SecretEncoding_encoding_json      SecretEncoding = 1
	SecretEncoding_encoding_proto     SecretEncoding = 2
	SecretEncoding_encoding_binary    SecretEncoding = 3
)

// Enum value maps for SecretEncoding.
var (
	SecretEncoding_name = map[int32]string{
		0: "encoding_undefined",
		1: "encoding_json",
		2: "encoding_proto",
		3: "encoding_binary",
	}
	SecretEncoding_value = map[string]int32{
		"encoding_undefined": 0,
		"encoding_json":      1,
		"encoding_proto":     2,
		"encoding_binary":    3,
	}
)

func (x SecretEncoding) Enum() *SecretEncoding {
	p := new(SecretEncoding)
	*p = x
	return p
}

func (x SecretEncoding) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (SecretEncoding) Descriptor() protoreflect.EnumDescriptor {
	return file_vault_proto_enumTypes[1].Descriptor()
}

func (SecretEncoding) Type() protoreflect.EnumType {
	return &file_vault_proto_enumTypes[1]
}

func (x SecretEncoding) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use SecretEncoding.Descriptor instead.
func (SecretEncoding) EnumDescriptor() ([]byte, []int) {
	return file_vault_proto_rawDescGZIP(), []int{1}
}

type VaultType int32

const (
	VaultType_None               VaultType = 0
	VaultType_KeyRing            VaultType = 1
	VaultType_LinuxKernelKeyring VaultType = 2
	VaultType_EncryptedFile      VaultType = 3
	VaultType_HashiCorp          VaultType = 4
	VaultType_GCPSecretsManager  VaultType = 5
	VaultType_AWSSecretsManager  VaultType = 6
	VaultType_AWSParameterStore  VaultType = 7
	VaultType_GCPBerglas         VaultType = 8
	VaultType_Memory             VaultType = 9
)

// Enum value maps for VaultType.
var (
	VaultType_name = map[int32]string{
		0: "None",
		1: "KeyRing",
		2: "LinuxKernelKeyring",
		3: "EncryptedFile",
		4: "HashiCorp",
		5: "GCPSecretsManager",
		6: "AWSSecretsManager",
		7: "AWSParameterStore",
		8: "GCPBerglas",
		9: "Memory",
	}
	VaultType_value = map[string]int32{
		"None":               0,
		"KeyRing":            1,
		"LinuxKernelKeyring": 2,
		"EncryptedFile":      3,
		"HashiCorp":          4,
		"GCPSecretsManager":  5,
		"AWSSecretsManager":  6,
		"AWSParameterStore":  7,
		"GCPBerglas":         8,
		"Memory":             9,
	}
)

func (x VaultType) Enum() *VaultType {
	p := new(VaultType)
	*p = x
	return p
}

func (x VaultType) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (VaultType) Descriptor() protoreflect.EnumDescriptor {
	return file_vault_proto_enumTypes[2].Descriptor()
}

func (VaultType) Type() protoreflect.EnumType {
	return &file_vault_proto_enumTypes[2]
}

func (x VaultType) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use VaultType.Descriptor instead.
func (VaultType) EnumDescriptor() ([]byte, []int) {
	return file_vault_proto_rawDescGZIP(), []int{2}
}

type SecretID struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Key string `protobuf:"bytes,1,opt,name=key,proto3" json:"key,omitempty"`
}

func (x *SecretID) Reset() {
	*x = SecretID{}
	if protoimpl.UnsafeEnabled {
		mi := &file_vault_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *SecretID) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*SecretID) ProtoMessage() {}

func (x *SecretID) ProtoReflect() protoreflect.Message {
	mi := &file_vault_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use SecretID.ProtoReflect.Descriptor instead.
func (*SecretID) Descriptor() ([]byte, []int) {
	return file_vault_proto_rawDescGZIP(), []int{0}
}

func (x *SecretID) GetKey() string {
	if x != nil {
		return x.Key
	}
	return ""
}

type Secret struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Key      string         `protobuf:"bytes,1,opt,name=key,proto3" json:"key,omitempty"`
	Label    string         `protobuf:"bytes,2,opt,name=label,proto3" json:"label,omitempty"`
	Data     []byte         `protobuf:"bytes,3,opt,name=data,proto3" json:"data,omitempty"`
	Encoding SecretEncoding `protobuf:"varint,4,opt,name=encoding,proto3,enum=cnquery.providers.v1.SecretEncoding" json:"encoding,omitempty"`
}

func (x *Secret) Reset() {
	*x = Secret{}
	if protoimpl.UnsafeEnabled {
		mi := &file_vault_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Secret) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Secret) ProtoMessage() {}

func (x *Secret) ProtoReflect() protoreflect.Message {
	mi := &file_vault_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Secret.ProtoReflect.Descriptor instead.
func (*Secret) Descriptor() ([]byte, []int) {
	return file_vault_proto_rawDescGZIP(), []int{1}
}

func (x *Secret) GetKey() string {
	if x != nil {
		return x.Key
	}
	return ""
}

func (x *Secret) GetLabel() string {
	if x != nil {
		return x.Label
	}
	return ""
}

func (x *Secret) GetData() []byte {
	if x != nil {
		return x.Data
	}
	return nil
}

func (x *Secret) GetEncoding() SecretEncoding {
	if x != nil {
		return x.Encoding
	}
	return SecretEncoding_encoding_undefined
}

type Empty struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
}

func (x *Empty) Reset() {
	*x = Empty{}
	if protoimpl.UnsafeEnabled {
		mi := &file_vault_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Empty) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Empty) ProtoMessage() {}

func (x *Empty) ProtoReflect() protoreflect.Message {
	mi := &file_vault_proto_msgTypes[2]
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
	return file_vault_proto_rawDescGZIP(), []int{2}
}

type VaultInfo struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Name string `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
}

func (x *VaultInfo) Reset() {
	*x = VaultInfo{}
	if protoimpl.UnsafeEnabled {
		mi := &file_vault_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *VaultInfo) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*VaultInfo) ProtoMessage() {}

func (x *VaultInfo) ProtoReflect() protoreflect.Message {
	mi := &file_vault_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use VaultInfo.ProtoReflect.Descriptor instead.
func (*VaultInfo) Descriptor() ([]byte, []int) {
	return file_vault_proto_rawDescGZIP(), []int{3}
}

func (x *VaultInfo) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

// Credential holds authentication information
type Credential struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	SecretId string         `protobuf:"bytes,1,opt,name=secret_id,json=secretId,proto3" json:"secret_id,omitempty"`
	Type     CredentialType `protobuf:"varint,2,opt,name=type,proto3,enum=cnquery.providers.v1.CredentialType" json:"type,omitempty"`
	User     string         `protobuf:"bytes,3,opt,name=user,proto3" json:"user,omitempty"`
	Secret   []byte         `protobuf:"bytes,4,opt,name=secret,proto3" json:"secret,omitempty"`
	// the following are optional and sugar for defining a secret
	// those values are only allowed for reading in yaml values but not via API calls
	Password string `protobuf:"bytes,21,opt,name=password,proto3" json:"password,omitempty"` // optional, could also be the password for the private key
	// for user convenience we define private_key, this allows yaml/json writers
	// to just embed the string representation, otherwise it would need to be base64 encoded
	PrivateKey string `protobuf:"bytes,22,opt,name=private_key,json=privateKey,proto3" json:"private_key,omitempty"`
	// for user convenience we define private_key_path which loads a local file into the
	// secret
	PrivateKeyPath string `protobuf:"bytes,23,opt,name=private_key_path,json=privateKeyPath,proto3" json:"private_key_path,omitempty"`
}

func (x *Credential) Reset() {
	*x = Credential{}
	if protoimpl.UnsafeEnabled {
		mi := &file_vault_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Credential) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Credential) ProtoMessage() {}

func (x *Credential) ProtoReflect() protoreflect.Message {
	mi := &file_vault_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Credential.ProtoReflect.Descriptor instead.
func (*Credential) Descriptor() ([]byte, []int) {
	return file_vault_proto_rawDescGZIP(), []int{4}
}

func (x *Credential) GetSecretId() string {
	if x != nil {
		return x.SecretId
	}
	return ""
}

func (x *Credential) GetType() CredentialType {
	if x != nil {
		return x.Type
	}
	return CredentialType_undefined
}

func (x *Credential) GetUser() string {
	if x != nil {
		return x.User
	}
	return ""
}

func (x *Credential) GetSecret() []byte {
	if x != nil {
		return x.Secret
	}
	return nil
}

func (x *Credential) GetPassword() string {
	if x != nil {
		return x.Password
	}
	return ""
}

func (x *Credential) GetPrivateKey() string {
	if x != nil {
		return x.PrivateKey
	}
	return ""
}

func (x *Credential) GetPrivateKeyPath() string {
	if x != nil {
		return x.PrivateKeyPath
	}
	return ""
}

type VaultConfiguration struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Name    string            `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	Type    VaultType         `protobuf:"varint,2,opt,name=type,proto3,enum=cnquery.providers.v1.VaultType" json:"type,omitempty"`
	Options map[string]string `protobuf:"bytes,3,rep,name=options,proto3" json:"options,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
}

func (x *VaultConfiguration) Reset() {
	*x = VaultConfiguration{}
	if protoimpl.UnsafeEnabled {
		mi := &file_vault_proto_msgTypes[5]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *VaultConfiguration) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*VaultConfiguration) ProtoMessage() {}

func (x *VaultConfiguration) ProtoReflect() protoreflect.Message {
	mi := &file_vault_proto_msgTypes[5]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use VaultConfiguration.ProtoReflect.Descriptor instead.
func (*VaultConfiguration) Descriptor() ([]byte, []int) {
	return file_vault_proto_rawDescGZIP(), []int{5}
}

func (x *VaultConfiguration) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *VaultConfiguration) GetType() VaultType {
	if x != nil {
		return x.Type
	}
	return VaultType_None
}

func (x *VaultConfiguration) GetOptions() map[string]string {
	if x != nil {
		return x.Options
	}
	return nil
}

var File_vault_proto protoreflect.FileDescriptor

var file_vault_proto_rawDesc = []byte{
	0x0a, 0x0b, 0x76, 0x61, 0x75, 0x6c, 0x74, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x14, 0x63,
	0x6e, 0x71, 0x75, 0x65, 0x72, 0x79, 0x2e, 0x70, 0x72, 0x6f, 0x76, 0x69, 0x64, 0x65, 0x72, 0x73,
	0x2e, 0x76, 0x31, 0x22, 0x1c, 0x0a, 0x08, 0x53, 0x65, 0x63, 0x72, 0x65, 0x74, 0x49, 0x44, 0x12,
	0x10, 0x0a, 0x03, 0x6b, 0x65, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x6b, 0x65,
	0x79, 0x22, 0x86, 0x01, 0x0a, 0x06, 0x53, 0x65, 0x63, 0x72, 0x65, 0x74, 0x12, 0x10, 0x0a, 0x03,
	0x6b, 0x65, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x6b, 0x65, 0x79, 0x12, 0x14,
	0x0a, 0x05, 0x6c, 0x61, 0x62, 0x65, 0x6c, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x6c,
	0x61, 0x62, 0x65, 0x6c, 0x12, 0x12, 0x0a, 0x04, 0x64, 0x61, 0x74, 0x61, 0x18, 0x03, 0x20, 0x01,
	0x28, 0x0c, 0x52, 0x04, 0x64, 0x61, 0x74, 0x61, 0x12, 0x40, 0x0a, 0x08, 0x65, 0x6e, 0x63, 0x6f,
	0x64, 0x69, 0x6e, 0x67, 0x18, 0x04, 0x20, 0x01, 0x28, 0x0e, 0x32, 0x24, 0x2e, 0x63, 0x6e, 0x71,
	0x75, 0x65, 0x72, 0x79, 0x2e, 0x70, 0x72, 0x6f, 0x76, 0x69, 0x64, 0x65, 0x72, 0x73, 0x2e, 0x76,
	0x31, 0x2e, 0x53, 0x65, 0x63, 0x72, 0x65, 0x74, 0x45, 0x6e, 0x63, 0x6f, 0x64, 0x69, 0x6e, 0x67,
	0x52, 0x08, 0x65, 0x6e, 0x63, 0x6f, 0x64, 0x69, 0x6e, 0x67, 0x22, 0x07, 0x0a, 0x05, 0x45, 0x6d,
	0x70, 0x74, 0x79, 0x22, 0x1f, 0x0a, 0x09, 0x56, 0x61, 0x75, 0x6c, 0x74, 0x49, 0x6e, 0x66, 0x6f,
	0x12, 0x12, 0x0a, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04,
	0x6e, 0x61, 0x6d, 0x65, 0x22, 0xfc, 0x01, 0x0a, 0x0a, 0x43, 0x72, 0x65, 0x64, 0x65, 0x6e, 0x74,
	0x69, 0x61, 0x6c, 0x12, 0x1b, 0x0a, 0x09, 0x73, 0x65, 0x63, 0x72, 0x65, 0x74, 0x5f, 0x69, 0x64,
	0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x08, 0x73, 0x65, 0x63, 0x72, 0x65, 0x74, 0x49, 0x64,
	0x12, 0x38, 0x0a, 0x04, 0x74, 0x79, 0x70, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0e, 0x32, 0x24,
	0x2e, 0x63, 0x6e, 0x71, 0x75, 0x65, 0x72, 0x79, 0x2e, 0x70, 0x72, 0x6f, 0x76, 0x69, 0x64, 0x65,
	0x72, 0x73, 0x2e, 0x76, 0x31, 0x2e, 0x43, 0x72, 0x65, 0x64, 0x65, 0x6e, 0x74, 0x69, 0x61, 0x6c,
	0x54, 0x79, 0x70, 0x65, 0x52, 0x04, 0x74, 0x79, 0x70, 0x65, 0x12, 0x12, 0x0a, 0x04, 0x75, 0x73,
	0x65, 0x72, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x75, 0x73, 0x65, 0x72, 0x12, 0x16,
	0x0a, 0x06, 0x73, 0x65, 0x63, 0x72, 0x65, 0x74, 0x18, 0x04, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x06,
	0x73, 0x65, 0x63, 0x72, 0x65, 0x74, 0x12, 0x1a, 0x0a, 0x08, 0x70, 0x61, 0x73, 0x73, 0x77, 0x6f,
	0x72, 0x64, 0x18, 0x15, 0x20, 0x01, 0x28, 0x09, 0x52, 0x08, 0x70, 0x61, 0x73, 0x73, 0x77, 0x6f,
	0x72, 0x64, 0x12, 0x1f, 0x0a, 0x0b, 0x70, 0x72, 0x69, 0x76, 0x61, 0x74, 0x65, 0x5f, 0x6b, 0x65,
	0x79, 0x18, 0x16, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0a, 0x70, 0x72, 0x69, 0x76, 0x61, 0x74, 0x65,
	0x4b, 0x65, 0x79, 0x12, 0x28, 0x0a, 0x10, 0x70, 0x72, 0x69, 0x76, 0x61, 0x74, 0x65, 0x5f, 0x6b,
	0x65, 0x79, 0x5f, 0x70, 0x61, 0x74, 0x68, 0x18, 0x17, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0e, 0x70,
	0x72, 0x69, 0x76, 0x61, 0x74, 0x65, 0x4b, 0x65, 0x79, 0x50, 0x61, 0x74, 0x68, 0x4a, 0x04, 0x08,
	0x05, 0x10, 0x06, 0x22, 0xea, 0x01, 0x0a, 0x12, 0x56, 0x61, 0x75, 0x6c, 0x74, 0x43, 0x6f, 0x6e,
	0x66, 0x69, 0x67, 0x75, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x12, 0x12, 0x0a, 0x04, 0x6e, 0x61,
	0x6d, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x12, 0x33,
	0x0a, 0x04, 0x74, 0x79, 0x70, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0e, 0x32, 0x1f, 0x2e, 0x63,
	0x6e, 0x71, 0x75, 0x65, 0x72, 0x79, 0x2e, 0x70, 0x72, 0x6f, 0x76, 0x69, 0x64, 0x65, 0x72, 0x73,
	0x2e, 0x76, 0x31, 0x2e, 0x56, 0x61, 0x75, 0x6c, 0x74, 0x54, 0x79, 0x70, 0x65, 0x52, 0x04, 0x74,
	0x79, 0x70, 0x65, 0x12, 0x4f, 0x0a, 0x07, 0x6f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x18, 0x03,
	0x20, 0x03, 0x28, 0x0b, 0x32, 0x35, 0x2e, 0x63, 0x6e, 0x71, 0x75, 0x65, 0x72, 0x79, 0x2e, 0x70,
	0x72, 0x6f, 0x76, 0x69, 0x64, 0x65, 0x72, 0x73, 0x2e, 0x76, 0x31, 0x2e, 0x56, 0x61, 0x75, 0x6c,
	0x74, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x75, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x2e, 0x4f,
	0x70, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x52, 0x07, 0x6f, 0x70, 0x74,
	0x69, 0x6f, 0x6e, 0x73, 0x1a, 0x3a, 0x0a, 0x0c, 0x4f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x45,
	0x6e, 0x74, 0x72, 0x79, 0x12, 0x10, 0x0a, 0x03, 0x6b, 0x65, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28,
	0x09, 0x52, 0x03, 0x6b, 0x65, 0x79, 0x12, 0x14, 0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18,
	0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x3a, 0x02, 0x38, 0x01,
	0x2a, 0xbd, 0x01, 0x0a, 0x0e, 0x43, 0x72, 0x65, 0x64, 0x65, 0x6e, 0x74, 0x69, 0x61, 0x6c, 0x54,
	0x79, 0x70, 0x65, 0x12, 0x0d, 0x0a, 0x09, 0x75, 0x6e, 0x64, 0x65, 0x66, 0x69, 0x6e, 0x65, 0x64,
	0x10, 0x00, 0x12, 0x0c, 0x0a, 0x08, 0x70, 0x61, 0x73, 0x73, 0x77, 0x6f, 0x72, 0x64, 0x10, 0x01,
	0x12, 0x0f, 0x0a, 0x0b, 0x70, 0x72, 0x69, 0x76, 0x61, 0x74, 0x65, 0x5f, 0x6b, 0x65, 0x79, 0x10,
	0x02, 0x12, 0x0d, 0x0a, 0x09, 0x73, 0x73, 0x68, 0x5f, 0x61, 0x67, 0x65, 0x6e, 0x74, 0x10, 0x03,
	0x12, 0x0a, 0x0a, 0x06, 0x62, 0x65, 0x61, 0x72, 0x65, 0x72, 0x10, 0x04, 0x12, 0x15, 0x0a, 0x11,
	0x63, 0x72, 0x65, 0x64, 0x65, 0x6e, 0x74, 0x69, 0x61, 0x6c, 0x73, 0x5f, 0x71, 0x75, 0x65, 0x72,
	0x79, 0x10, 0x05, 0x12, 0x08, 0x0a, 0x04, 0x6a, 0x73, 0x6f, 0x6e, 0x10, 0x06, 0x12, 0x1c, 0x0a,
	0x18, 0x61, 0x77, 0x73, 0x5f, 0x65, 0x63, 0x32, 0x5f, 0x69, 0x6e, 0x73, 0x74, 0x61, 0x6e, 0x63,
	0x65, 0x5f, 0x63, 0x6f, 0x6e, 0x6e, 0x65, 0x63, 0x74, 0x10, 0x07, 0x12, 0x17, 0x0a, 0x13, 0x61,
	0x77, 0x73, 0x5f, 0x65, 0x63, 0x32, 0x5f, 0x73, 0x73, 0x6d, 0x5f, 0x73, 0x65, 0x73, 0x73, 0x69,
	0x6f, 0x6e, 0x10, 0x08, 0x12, 0x0a, 0x0a, 0x06, 0x70, 0x6b, 0x63, 0x73, 0x31, 0x32, 0x10, 0x09,
	0x2a, 0x64, 0x0a, 0x0e, 0x53, 0x65, 0x63, 0x72, 0x65, 0x74, 0x45, 0x6e, 0x63, 0x6f, 0x64, 0x69,
	0x6e, 0x67, 0x12, 0x16, 0x0a, 0x12, 0x65, 0x6e, 0x63, 0x6f, 0x64, 0x69, 0x6e, 0x67, 0x5f, 0x75,
	0x6e, 0x64, 0x65, 0x66, 0x69, 0x6e, 0x65, 0x64, 0x10, 0x00, 0x12, 0x11, 0x0a, 0x0d, 0x65, 0x6e,
	0x63, 0x6f, 0x64, 0x69, 0x6e, 0x67, 0x5f, 0x6a, 0x73, 0x6f, 0x6e, 0x10, 0x01, 0x12, 0x12, 0x0a,
	0x0e, 0x65, 0x6e, 0x63, 0x6f, 0x64, 0x69, 0x6e, 0x67, 0x5f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x10,
	0x02, 0x12, 0x13, 0x0a, 0x0f, 0x65, 0x6e, 0x63, 0x6f, 0x64, 0x69, 0x6e, 0x67, 0x5f, 0x62, 0x69,
	0x6e, 0x61, 0x72, 0x79, 0x10, 0x03, 0x2a, 0xbd, 0x01, 0x0a, 0x09, 0x56, 0x61, 0x75, 0x6c, 0x74,
	0x54, 0x79, 0x70, 0x65, 0x12, 0x08, 0x0a, 0x04, 0x4e, 0x6f, 0x6e, 0x65, 0x10, 0x00, 0x12, 0x0b,
	0x0a, 0x07, 0x4b, 0x65, 0x79, 0x52, 0x69, 0x6e, 0x67, 0x10, 0x01, 0x12, 0x16, 0x0a, 0x12, 0x4c,
	0x69, 0x6e, 0x75, 0x78, 0x4b, 0x65, 0x72, 0x6e, 0x65, 0x6c, 0x4b, 0x65, 0x79, 0x72, 0x69, 0x6e,
	0x67, 0x10, 0x02, 0x12, 0x11, 0x0a, 0x0d, 0x45, 0x6e, 0x63, 0x72, 0x79, 0x70, 0x74, 0x65, 0x64,
	0x46, 0x69, 0x6c, 0x65, 0x10, 0x03, 0x12, 0x0d, 0x0a, 0x09, 0x48, 0x61, 0x73, 0x68, 0x69, 0x43,
	0x6f, 0x72, 0x70, 0x10, 0x04, 0x12, 0x15, 0x0a, 0x11, 0x47, 0x43, 0x50, 0x53, 0x65, 0x63, 0x72,
	0x65, 0x74, 0x73, 0x4d, 0x61, 0x6e, 0x61, 0x67, 0x65, 0x72, 0x10, 0x05, 0x12, 0x15, 0x0a, 0x11,
	0x41, 0x57, 0x53, 0x53, 0x65, 0x63, 0x72, 0x65, 0x74, 0x73, 0x4d, 0x61, 0x6e, 0x61, 0x67, 0x65,
	0x72, 0x10, 0x06, 0x12, 0x15, 0x0a, 0x11, 0x41, 0x57, 0x53, 0x50, 0x61, 0x72, 0x61, 0x6d, 0x65,
	0x74, 0x65, 0x72, 0x53, 0x74, 0x6f, 0x72, 0x65, 0x10, 0x07, 0x12, 0x0e, 0x0a, 0x0a, 0x47, 0x43,
	0x50, 0x42, 0x65, 0x72, 0x67, 0x6c, 0x61, 0x73, 0x10, 0x08, 0x12, 0x0a, 0x0a, 0x06, 0x4d, 0x65,
	0x6d, 0x6f, 0x72, 0x79, 0x10, 0x09, 0x32, 0xd8, 0x01, 0x0a, 0x05, 0x56, 0x61, 0x75, 0x6c, 0x74,
	0x12, 0x45, 0x0a, 0x05, 0x41, 0x62, 0x6f, 0x75, 0x74, 0x12, 0x1b, 0x2e, 0x63, 0x6e, 0x71, 0x75,
	0x65, 0x72, 0x79, 0x2e, 0x70, 0x72, 0x6f, 0x76, 0x69, 0x64, 0x65, 0x72, 0x73, 0x2e, 0x76, 0x31,
	0x2e, 0x45, 0x6d, 0x70, 0x74, 0x79, 0x1a, 0x1f, 0x2e, 0x63, 0x6e, 0x71, 0x75, 0x65, 0x72, 0x79,
	0x2e, 0x70, 0x72, 0x6f, 0x76, 0x69, 0x64, 0x65, 0x72, 0x73, 0x2e, 0x76, 0x31, 0x2e, 0x56, 0x61,
	0x75, 0x6c, 0x74, 0x49, 0x6e, 0x66, 0x6f, 0x12, 0x43, 0x0a, 0x03, 0x47, 0x65, 0x74, 0x12, 0x1e,
	0x2e, 0x63, 0x6e, 0x71, 0x75, 0x65, 0x72, 0x79, 0x2e, 0x70, 0x72, 0x6f, 0x76, 0x69, 0x64, 0x65,
	0x72, 0x73, 0x2e, 0x76, 0x31, 0x2e, 0x53, 0x65, 0x63, 0x72, 0x65, 0x74, 0x49, 0x44, 0x1a, 0x1c,
	0x2e, 0x63, 0x6e, 0x71, 0x75, 0x65, 0x72, 0x79, 0x2e, 0x70, 0x72, 0x6f, 0x76, 0x69, 0x64, 0x65,
	0x72, 0x73, 0x2e, 0x76, 0x31, 0x2e, 0x53, 0x65, 0x63, 0x72, 0x65, 0x74, 0x12, 0x43, 0x0a, 0x03,
	0x53, 0x65, 0x74, 0x12, 0x1c, 0x2e, 0x63, 0x6e, 0x71, 0x75, 0x65, 0x72, 0x79, 0x2e, 0x70, 0x72,
	0x6f, 0x76, 0x69, 0x64, 0x65, 0x72, 0x73, 0x2e, 0x76, 0x31, 0x2e, 0x53, 0x65, 0x63, 0x72, 0x65,
	0x74, 0x1a, 0x1e, 0x2e, 0x63, 0x6e, 0x71, 0x75, 0x65, 0x72, 0x79, 0x2e, 0x70, 0x72, 0x6f, 0x76,
	0x69, 0x64, 0x65, 0x72, 0x73, 0x2e, 0x76, 0x31, 0x2e, 0x53, 0x65, 0x63, 0x72, 0x65, 0x74, 0x49,
	0x44, 0x42, 0x31, 0x5a, 0x2f, 0x67, 0x6f, 0x2e, 0x6d, 0x6f, 0x6e, 0x64, 0x6f, 0x6f, 0x2e, 0x63,
	0x6f, 0x6d, 0x2f, 0x63, 0x6e, 0x71, 0x75, 0x65, 0x72, 0x79, 0x2f, 0x76, 0x39, 0x2f, 0x70, 0x72,
	0x6f, 0x76, 0x69, 0x64, 0x65, 0x72, 0x73, 0x2d, 0x73, 0x64, 0x6b, 0x2f, 0x76, 0x31, 0x2f, 0x76,
	0x61, 0x75, 0x6c, 0x74, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_vault_proto_rawDescOnce sync.Once
	file_vault_proto_rawDescData = file_vault_proto_rawDesc
)

func file_vault_proto_rawDescGZIP() []byte {
	file_vault_proto_rawDescOnce.Do(func() {
		file_vault_proto_rawDescData = protoimpl.X.CompressGZIP(file_vault_proto_rawDescData)
	})
	return file_vault_proto_rawDescData
}

var file_vault_proto_enumTypes = make([]protoimpl.EnumInfo, 3)
var file_vault_proto_msgTypes = make([]protoimpl.MessageInfo, 7)
var file_vault_proto_goTypes = []interface{}{
	(CredentialType)(0),        // 0: cnquery.providers.v1.CredentialType
	(SecretEncoding)(0),        // 1: cnquery.providers.v1.SecretEncoding
	(VaultType)(0),             // 2: cnquery.providers.v1.VaultType
	(*SecretID)(nil),           // 3: cnquery.providers.v1.SecretID
	(*Secret)(nil),             // 4: cnquery.providers.v1.Secret
	(*Empty)(nil),              // 5: cnquery.providers.v1.Empty
	(*VaultInfo)(nil),          // 6: cnquery.providers.v1.VaultInfo
	(*Credential)(nil),         // 7: cnquery.providers.v1.Credential
	(*VaultConfiguration)(nil), // 8: cnquery.providers.v1.VaultConfiguration
	nil,                        // 9: cnquery.providers.v1.VaultConfiguration.OptionsEntry
}
var file_vault_proto_depIdxs = []int32{
	1, // 0: cnquery.providers.v1.Secret.encoding:type_name -> cnquery.providers.v1.SecretEncoding
	0, // 1: cnquery.providers.v1.Credential.type:type_name -> cnquery.providers.v1.CredentialType
	2, // 2: cnquery.providers.v1.VaultConfiguration.type:type_name -> cnquery.providers.v1.VaultType
	9, // 3: cnquery.providers.v1.VaultConfiguration.options:type_name -> cnquery.providers.v1.VaultConfiguration.OptionsEntry
	5, // 4: cnquery.providers.v1.Vault.About:input_type -> cnquery.providers.v1.Empty
	3, // 5: cnquery.providers.v1.Vault.Get:input_type -> cnquery.providers.v1.SecretID
	4, // 6: cnquery.providers.v1.Vault.Set:input_type -> cnquery.providers.v1.Secret
	6, // 7: cnquery.providers.v1.Vault.About:output_type -> cnquery.providers.v1.VaultInfo
	4, // 8: cnquery.providers.v1.Vault.Get:output_type -> cnquery.providers.v1.Secret
	3, // 9: cnquery.providers.v1.Vault.Set:output_type -> cnquery.providers.v1.SecretID
	7, // [7:10] is the sub-list for method output_type
	4, // [4:7] is the sub-list for method input_type
	4, // [4:4] is the sub-list for extension type_name
	4, // [4:4] is the sub-list for extension extendee
	0, // [0:4] is the sub-list for field type_name
}

func init() { file_vault_proto_init() }
func file_vault_proto_init() {
	if File_vault_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_vault_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*SecretID); i {
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
		file_vault_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Secret); i {
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
		file_vault_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
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
		file_vault_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*VaultInfo); i {
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
		file_vault_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Credential); i {
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
		file_vault_proto_msgTypes[5].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*VaultConfiguration); i {
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
			RawDescriptor: file_vault_proto_rawDesc,
			NumEnums:      3,
			NumMessages:   7,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_vault_proto_goTypes,
		DependencyIndexes: file_vault_proto_depIdxs,
		EnumInfos:         file_vault_proto_enumTypes,
		MessageInfos:      file_vault_proto_msgTypes,
	}.Build()
	File_vault_proto = out.File
	file_vault_proto_rawDesc = nil
	file_vault_proto_goTypes = nil
	file_vault_proto_depIdxs = nil
}
