// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

// tofuConfigExtensions are the OpenTofu-specific configuration file extensions.
// OpenTofu loads these *instead of* the identically-named .tf file when both are
// present, so a directory holding them cannot be read correctly by looking at
// .tf files alone: we would miss configuration entirely, or report the .tf file
// that OpenTofu itself ignores.
//
// Test files (.tofutest.hcl / .tofutest.json) are deliberately absent. They
// carry no deployed configuration, and the .tftest.* equivalents are not read
// either.
var tofuConfigExtensions = []string{
	".tofu",
	".tofu.json",
	".tofuvars",
	".tofuvars.json",
}

// isTofuConfigFile reports whether path names an OpenTofu-specific
// configuration file.
func isTofuConfigFile(path string) bool {
	name := filepath.Base(path)
	for _, ext := range tofuConfigExtensions {
		if strings.HasSuffix(name, ext) {
			return true
		}
	}
	return false
}

// UnsupportedTofuFilesError reports OpenTofu-specific configuration files that
// this provider cannot read yet. It is returned instead of silently skipping
// them, because skipping produces a successful scan of an empty configuration:
// every assertion over terraform.resources then passes vacuously, and a policy
// reports compliant on a configuration that was never read.
type UnsupportedTofuFilesError struct {
	// Files holds the offending paths, sorted, as encountered on disk.
	Files []string
}

func (e *UnsupportedTofuFilesError) Error() string {
	files := e.Files
	// keep the message bounded on a large repository
	const maxListed = 5
	listed := files
	suffix := ""
	if len(listed) > maxListed {
		listed = listed[:maxListed]
		suffix = fmt.Sprintf(" (and %d more)", len(files)-maxListed)
	}
	return fmt.Sprintf(
		"found %d OpenTofu configuration file(s) that this provider cannot parse yet: %s%s. "+
			"Scanning would report on an incomplete configuration, so it was stopped instead. "+
			"Only .tf and .tf.json files are supported today",
		len(files), strings.Join(listed, ", "), suffix)
}

// newUnsupportedTofuFilesError builds the error from the collected paths,
// returning nil when there is nothing to report so callers can assign it
// directly.
func newUnsupportedTofuFilesError(files []string) error {
	if len(files) == 0 {
		return nil
	}
	sorted := make([]string, len(files))
	copy(sorted, files)
	sort.Strings(sorted)
	return &UnsupportedTofuFilesError{Files: sorted}
}

// documentKind describes what a JSON document handed to the state or plan
// connection actually is. This provider reads the `terraform show -json`
// representation; every other shape has to be reported rather than parsed into
// an empty result.
type documentKind int

const (
	// documentShowJSON is the `terraform show -json` representation, the only
	// shape this provider can read.
	documentShowJSON documentKind = iota
	// documentEncrypted is an OpenTofu encrypted state or plan envelope.
	documentEncrypted
	// documentNativeState is a raw terraform.tfstate file.
	documentNativeState
	// documentUnrecognized is a JSON object that is neither.
	documentUnrecognized
)

// classifyDocument inspects the top-level keys of a JSON document to determine
// whether it is something this provider can read.
//
// The distinguishing keys:
//   - `encrypted_data` is the OpenTofu encryption envelope.
//   - `format_version` is emitted by every `terraform show -json` run.
//   - `version` (without `format_version`) is the native state file format.
func classifyDocument(data []byte) (documentKind, error) {
	var probe map[string]json.RawMessage
	if err := json.Unmarshal(data, &probe); err != nil {
		return documentUnrecognized, err
	}

	if _, ok := probe["encrypted_data"]; ok {
		return documentEncrypted, nil
	}
	if _, ok := probe["format_version"]; ok {
		return documentShowJSON, nil
	}
	if _, ok := probe["version"]; ok {
		return documentNativeState, nil
	}
	return documentUnrecognized, nil
}

// validateStateDocument returns an actionable error when data is not the
// `terraform show -json` representation of a state file.
func validateStateDocument(data []byte) error {
	kind, err := classifyDocument(data)
	if err != nil {
		return fmt.Errorf("state file is not valid JSON: %w. "+
			"Convert it first with `terraform show -json > state.json`", err)
	}

	switch kind {
	case documentEncrypted:
		return fmt.Errorf("state file is encrypted and cannot be read. " +
			"Decrypt it by running `terraform show -json` (or `tofu show -json`) " +
			"with the encryption key configured, and scan that output")
	case documentNativeState:
		return fmt.Errorf("this is a raw state file, which this provider cannot read. " +
			"Convert it first with `terraform show -json > state.json` and scan that file")
	case documentUnrecognized:
		return fmt.Errorf("state file is missing the `format_version` field, so it is not " +
			"the output of `terraform show -json`. Convert the state file first with " +
			"`terraform show -json > state.json` and scan that file")
	}
	return nil
}

// validatePlanDocument returns an actionable error when data is not the
// `terraform show -json` representation of a plan file.
func validatePlanDocument(data []byte) error {
	kind, err := classifyDocument(data)
	if err != nil {
		return fmt.Errorf("plan file is not valid JSON: %w. "+
			"Binary plan files must be converted first with "+
			"`terraform show -json tfplan > tfplan.json`", err)
	}

	switch kind {
	case documentEncrypted:
		return fmt.Errorf("plan file is encrypted and cannot be read. " +
			"Decrypt it by running `terraform show -json` (or `tofu show -json`) " +
			"with the encryption key configured, and scan that output")
	case documentNativeState, documentUnrecognized:
		return fmt.Errorf("plan file is missing the `format_version` field, so it is not " +
			"the output of `terraform show -json`. Convert the plan file first with " +
			"`terraform show -json tfplan > tfplan.json` and scan that file")
	}
	return nil
}
