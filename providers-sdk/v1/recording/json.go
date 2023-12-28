// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package recording

import (
	"encoding/json"
	"errors"
	"go.mondoo.com/cnquery/v9/llx"
	"go.mondoo.com/cnquery/v9/utils/multierr"
)

//go:generate protoc --proto_path=../../../:. --go_out=. --go_opt=paths=source_relative recording.proto

type recordingResourceJsonData struct {
	Resource string                  `json:"resource,omitempty"`
	Id       string                  `json:"id,omitempty"`
	Fields   map[string]*llx.RawData `json:"fields,omitempty"`
}

func (r *RecordingResourceData) MarshalJSON() ([]byte, error) {

	d := &recordingResourceJsonData{
		Resource: r.Name,
		Id:       r.Id,
		Fields:   make(map[string]*llx.RawData),
	}

	for k, v := range r.Fields {
		d.Fields[k] = v.RawData()
	}
	return json.Marshal(d)
}

func (r *RecordingResourceData) UnmarshalJSON(data []byte) error {
	d := &recordingResourceJsonData{}
	err := json.Unmarshal(data, d)
	if err != nil {
		return err
	}

	r.Name = d.Resource
	r.Id = d.Id

	fields, err := RawDataArgsToResultArgs(d.Fields)
	if err != nil {
		return errors.New("failed to convert raw data to result")
	}
	r.Fields = fields

	return nil
}

func RawDataArgsToResultArgs(args map[string]*llx.RawData) (map[string]*llx.Result, error) {
	all := make(map[string]*llx.Result, len(args))
	var err multierr.Errors
	for k, v := range args {
		res := v.Result()
		if res.Error != "" {
			err.Add(errors.New("failed to convert '" + k + "': " + res.Error))
		} else {
			all[k] = res
		}
	}

	return all, err.Deduplicate()
}

func PrimitiveArgsToResultArgs(args map[string]*llx.Primitive) map[string]*llx.Result {
	res := make(map[string]*llx.Result, len(args))
	for k, v := range args {
		res[k] = &llx.Result{Data: v}
	}
	return res
}
