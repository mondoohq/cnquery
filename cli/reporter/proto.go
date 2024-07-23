// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package reporter

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"

	"go.mondoo.com/cnquery/v11/explorer"
	"go.mondoo.com/cnquery/v11/utils/iox"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
)

func ConvertToProto(data *explorer.ReportCollection) (*Report, error) {
	protoReport := &Report{
		Assets: map[string]*Asset{},
		Data:   map[string]*DataValues{},
		Errors: map[string]string{},
	}

	if data == nil {
		return protoReport, nil
	}

	queryMrnIdx := map[string]string{}

	// this case can happen when all assets error out, eg. no query pack is available that matches
	if data.Bundle != nil {
		for i := range data.Bundle.Packs {
			pack := data.Bundle.Packs[i]
			for j := range pack.Queries {
				query := pack.Queries[j]
				queryMrnIdx[query.CodeId] = query.Mrn
			}
		}
	}

	// fill in assets
	for assetMrn, a := range data.Assets {
		pAsset := &Asset{
			Mrn:     a.Mrn,
			Name:    a.Name,
			TraceId: a.TraceId,
			Labels:  a.Labels,
		}
		protoReport.Assets[assetMrn] = pAsset
	}

	// convert the data points to json
	for id, report := range data.Reports {
		assetMrn := prettyPrintString(id)

		resolved, ok := data.Resolved[id]
		if !ok {
			return nil, errors.New("cannot find resolved pack for " + id + " in report")
		}

		results := report.RawResults()
		for qid, query := range resolved.ExecutionJob.Queries {
			printID := queryMrnIdx[qid]
			if printID == "" {
				printID = qid
			}

			queryID := prettyPrintString(printID)

			buf := &bytes.Buffer{}
			w := iox.IOWriter{Writer: buf}
			err := CodeBundleToJSON(query.Code, results, &w)
			if err != nil {
				return nil, err
			}

			var v *structpb.Value
			var jsonStruct map[string]interface{}
			err = json.Unmarshal([]byte(buf.Bytes()), &jsonStruct)
			if err == nil {
				v, err = structpb.NewValue(jsonStruct)
				if err != nil {
					return nil, err
				}
			} else {
				v, err = structpb.NewValue(buf.String())
				if err != nil {
					return nil, err
				}
			}

			if protoReport.Data[assetMrn] == nil {
				protoReport.Data[assetMrn] = &DataValues{
					Values: map[string]*DataValue{},
				}
			}

			protoReport.Data[assetMrn].Values[queryID] = &DataValue{
				Content: v,
			}
		}
	}

	for id, errStatus := range data.Errors {
		assetMrn := prettyPrintString(id)
		errorMsg := errStatus.Message
		protoReport.Errors[assetMrn] = errorMsg
	}

	return protoReport, nil
}

func (r *Report) ToJSON() ([]byte, error) {
	return protojson.Marshal(r)
}

func JsonValue(v *structpb.Value) ([]byte, error) {
	return protojson.Marshal(v)
}

// similar to llx.PrettyPrintString but no double quotes around the string
func prettyPrintString(s string) string {
	res := s
	res = strings.ReplaceAll(res, "\\n", "\n")
	res = strings.ReplaceAll(res, "\\t", "\t")
	return res
}
