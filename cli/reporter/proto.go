// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package reporter

import (
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
)

// ToJSON converts the Report to JSON
func (r *Report) ToJSON() ([]byte, error) {
	return protojson.Marshal(r)
}

// JsonValue converts a structpb Value to JSON
func JsonValue(v *structpb.Value) ([]byte, error) {
	return protojson.Marshal(v)
}
