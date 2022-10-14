package reporter

import (
	"encoding/json"
	"errors"

	"go.mondoo.com/cnquery/explorer"
	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/shared"
)

func BundleResultsToJSON(code *llx.CodeBundle, results map[string]*llx.RawResult, out shared.OutputHelper) error {
	var checksums []string
	eps := code.CodeV2.Entrypoints()
	checksums = make([]string, len(eps))
	for i, ref := range eps {
		checksums[i] = code.CodeV2.Checksums[ref]
	}

	// since we iterate over checksums, we run into the situation that this could be a slice
	// eg. cnquery run k8s --query "platform { name } k8s.pod.name" --json

	out.WriteString("{")

	for j, checksum := range checksums {
		result := results[checksum]
		if result == nil {
			llx.JSONerror(errors.New("cannot find result for this query"))
		} else {
			jsonData := result.Data.JSONfield(checksum, code)
			out.Write(jsonData)
		}

		if len(checksums) != j+1 {
			out.WriteString(",")
		}
	}

	out.WriteString("}")

	return nil
}

func ReportCollectionToJSON(data *explorer.ReportCollection, out shared.OutputHelper) error {
	if data == nil {
		return nil
	}

	queryMrnIdx := map[string]string{}
	for i := range data.Bundle.Packs {
		pack := data.Bundle.Packs[i]
		for j := range pack.Queries {
			query := pack.Queries[j]
			queryMrnIdx[query.CodeId] = query.Mrn
		}
	}

	out.WriteString(
		"{" +
			"\"assets\":")
	assets, err := json.Marshal(data.Assets)
	if err != nil {
		return err
	}
	out.WriteString(string(assets))

	out.WriteString("," +
		"\"data\":" +
		"{")
	pre := ""
	for id, report := range data.Reports {
		out.WriteString(pre + llx.PrettyPrintString(id) + ":{")
		pre = ","

		resolved, ok := data.Resolved[id]
		if !ok {
			return errors.New("cannot find resolved pack for " + id + " in report")
		}

		results := report.RawResults()
		pre2 := ""
		for qid, query := range resolved.ExecutionJob.Queries {
			printID := queryMrnIdx[qid]
			if printID == "" {
				printID = qid
			}

			out.WriteString(pre2 + llx.PrettyPrintString(printID) + ":")
			pre2 = ","

			err := BundleResultsToJSON(query.Code, results, out)
			if err != nil {
				return err
			}
		}
		out.WriteString("}")
	}

	out.WriteString("}," +
		"\"errors\":" +
		"{")
	pre = ""
	for id, err := range data.Errors {
		out.WriteString(pre + llx.PrettyPrintString(id) + ":" + llx.PrettyPrintString(err))
		pre = ","
	}
	out.WriteString("}}")

	return nil
}
