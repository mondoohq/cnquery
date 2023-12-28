// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package recording

import (
	"go.mondoo.com/cnquery/v9/llx"
	"go.mondoo.com/cnquery/v9/providers-sdk/v1/inventory"
)

//go:generate protoc --proto_path=../../../:. --go_out=. --go_opt=paths=source_relative recording.proto

type Recording interface {
	Save() error
	EnsureAsset(asset *inventory.Asset, provider string, connectionID uint32, conf *inventory.Config)
	SetAssetMrn(connectionID uint32, mrn string)
	AddData(connectionID uint32, resource string, id string, field string, data *llx.RawData)
	GetData(connectionID uint32, resource string, id string, field string) (*llx.RawData, bool)
	GetResource(connectionID uint32, resource string, id string) (map[string]*llx.RawData, bool)
	ToProto() *RecordingData
}

type NullRecording struct{}

func (n NullRecording) Save() error {
	return nil
}

func (n NullRecording) EnsureAsset(asset *inventory.Asset, provider string, connectionID uint32, conf *inventory.Config) {
}

func (n NullRecording) SetAssetMrn(connectionID uint32, mrn string) {
}

func (n NullRecording) AddData(connectionID uint32, resource string, id string, field string, data *llx.RawData) {
}

func (n NullRecording) GetData(connectionID uint32, resource string, id string, field string) (*llx.RawData, bool) {
	return nil, false
}

func (n NullRecording) GetResource(connectionID uint32, resource string, id string) (map[string]*llx.RawData, bool) {
	return nil, false
}

func (n NullRecording) ToProto() *RecordingData {
	return nil
}
