// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package awsec2

import (
	"encoding/json"
	"strings"

	"github.com/rs/zerolog/log"
)

type recursive struct {
	GetMetadataValueFunc func(path string) (string, error)
}

func (r recursive) Crawl(path string) (any, error) {
	return r.getMetadataRecursively(path)
}

func (r recursive) getMetadataRecursively(path string) (any, error) {
	log.Trace().
		Str("path", path).
		Msg("os.id.awsec2> metadata")
	data, err := r.GetMetadataValueFunc(path)
	if err != nil {
		return nil, err
	}

	// If the response is JSON, parse it
	if isJSON(data) {
		var jsonData interface{}
		if err := json.Unmarshal([]byte(data), &jsonData); err != nil {
			return nil, err
		}
		return jsonData, nil
	}

	// Handle specific paths that return multiline strings (e.g., "managed-ssh-keys/signer-cert")
	if isMultilineString(path) {
		return data, nil // Preserve as a raw string
	}

	lines := strings.Split(data, "\n")

	// If the data contains sub-paths, fetch them recursively
	if len(lines) > 1 || strings.HasSuffix(data, "/") {
		result := make(map[string]any)

		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}

			subPath := path + line

			subData, err := r.getMetadataRecursively(subPath)
			if err != nil {
				log.Trace().Err(err).
					Str("path", path).
					Str("line", line).
					Msg("os.id.awsec2> failed to get sub-path metadata")
				continue
			}

			result[strings.TrimSuffix(line, "/")] = subData
		}

		return result, nil
	}

	// If it's a single value, return it as a string
	return data, nil
}

// isJSON checks if a string is valid JSON
func isJSON(data string) bool {
	var js json.RawMessage
	return json.Unmarshal([]byte(data), &js) == nil
}

// isMultilineString checks if a path should be treated as a raw multiline string
func isMultilineString(path string) bool {
	// Add any additional paths that should be treated as multiline strings
	return path == "managed-ssh-keys/signer-cert"
}
