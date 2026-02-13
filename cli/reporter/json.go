// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package reporter

import (
	"errors"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/utils/iox"
)

// CodeBundleToJSON converts a code bundle and its results to JSON output
func CodeBundleToJSON(code *llx.CodeBundle, results map[string]*llx.RawResult, out iox.OutputHelper) error {
	var checksums []string
	eps := code.CodeV2.Entrypoints()
	checksums = make([]string, len(eps))
	for i, ref := range eps {
		checksums[i] = code.CodeV2.Checksums[ref]
	}

	// since we iterate over checksums, we run into the situation that this could be a slice
	// eg. mql run k8s --query "platform { name } k8s.pod.name" --json

	_ = out.WriteString("{")

	for j, checksum := range checksums {
		result := results[checksum]
		if result == nil {
			llx.JSONerror(errors.New("cannot find result for this query"))
		} else {
			jsonData := result.Data.JSONfield(checksum, code)
			_, _ = out.Write(jsonData)
		}

		if len(checksums) != j+1 {
			_ = out.WriteString(",")
		}
	}

	_ = out.WriteString("}")

	return nil
}
