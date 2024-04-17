// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package reporter

import (
	"bytes"
	"encoding/csv"

	"go.mondoo.com/cnquery/v11/explorer"
	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/mrn"
	"go.mondoo.com/cnquery/v11/shared"
)

type csvStruct struct {
	AssetMrn    string
	AssetId     string
	AssetName   string
	QueryMrn    string
	QueryTitle  string
	MQL         string
	QueryResult string
}

func (c csvStruct) toSlice() []string {
	return []string{c.AssetMrn, c.AssetId, c.AssetName, c.QueryMrn, c.QueryTitle, c.MQL, c.QueryResult}
}

// ConvertToCSV writes the given report collection to the given output directory
func ConvertToCSV(data *explorer.ReportCollection, out shared.OutputHelper) error {
	w := csv.NewWriter(out)

	// write header
	err := w.Write(csvStruct{
		"Asset Mrn",
		"Asset ID",
		"Asset Name",
		"Query Mrn",
		"Query Title",
		"MQL",
		"Query Result",
	}.toSlice())
	if err != nil {
		return err
	}

	queryMrnIdx := map[string]*explorer.Mquery{}
	if data.Bundle != nil {
		for i := range data.Bundle.Packs {
			pack := data.Bundle.Packs[i]
			for j := range pack.Queries {
				query := pack.Queries[j]
				queryMrnIdx[query.CodeId] = query
			}
		}
	}

	for i := range data.Assets {
		asset := data.Assets[i]
		parsedMrn, err := mrn.NewMRN(asset.Mrn)
		if err != nil {
			return err
		}
		assetId, err := parsedMrn.ResourceID("assets")
		if err != nil {
			return err
		}

		if data.Errors != nil {
			errStatus, ok := data.Errors[asset.Mrn]
			if ok {
				err := w.Write(csvStruct{
					AssetMrn:    asset.Mrn,
					AssetId:     assetId,
					AssetName:   asset.Name,
					QueryMrn:    "",
					QueryTitle:  "",
					MQL:         "",
					QueryResult: errStatus.Message,
				}.toSlice())
				if err != nil {
					return err
				}
			}
		}

		if data.Reports != nil {
			report, ok := data.Reports[asset.Mrn]
			if ok {
				results := report.RawResults()
				resolvedPack := data.Resolved[asset.Mrn]
				if resolvedPack != nil && resolvedPack.ExecutionJob != nil {
					for qid, query := range resolvedPack.ExecutionJob.Queries {
						buf := &bytes.Buffer{}
						resultWriter := &shared.IOWriter{Writer: buf}
						err := ResultsToCsvEntry(query.Code, results, resultWriter)
						if err != nil {
							return err
						}

						var queryMrn string
						var queryTitle string
						mQuery := queryMrnIdx[qid]
						if mQuery != nil {
							queryMrn = mQuery.Mrn
							queryTitle = mQuery.Title
						}

						entry := csvStruct{
							AssetMrn:    asset.Mrn,
							AssetId:     assetId,
							AssetName:   asset.Name,
							QueryMrn:    queryMrn,
							QueryTitle:  queryTitle,
							MQL:         query.Query,
							QueryResult: string(buf.Bytes()),
						}

						err = w.Write(entry.toSlice())
						if err != nil {
							return err
						}
					}
				}
			}
		}
	}

	w.Flush()
	return w.Error()
}

func ResultsToCsvEntry(code *llx.CodeBundle, results map[string]*llx.RawResult, out shared.OutputHelper) error {
	var checksums []string
	eps := code.CodeV2.Entrypoints()
	checksums = make([]string, len(eps))
	for i, ref := range eps {
		checksums[i] = code.CodeV2.Checksums[ref]
	}

	// We try to flatten the information as much as possible. If we have multiple checksums we have no choice but to use
	// a full json out, otherwise we can use a simple value without the labels.
	if len(checksums) == 1 {
		checksum := checksums[0]
		result := results[checksum]
		if result != nil {
			jsonData := result.Data.JSON(checksum, code)
			out.Write(jsonData)
		}
		return nil
	}

	return CodeBundleToJSON(code, results, out)
}
