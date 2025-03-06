// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package metadata

import (
	"encoding/json"
	"regexp"
	"strings"

	"github.com/rs/zerolog/log"
)

// recursive is the interface passed to `Crawl` to fetch all metadata from an instance
type recursive interface {
	GetMetadataValue(path string) (string, error)
}

// Crawl fetches all metadata from an instance recursively
func Crawl(r recursive, path string) (any, error) {
	return getMetadataRecursively(r, path)
}

func getMetadataRecursively(r recursive, path string) (any, error) {
	log.Trace().Str("path", path).Msg("os.id.metadata> crawling")
	data, err := r.GetMetadataValue(path)
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

			subData, err := getMetadataRecursively(r, subPath)
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

// isJSON checks if a string is valid JSON.
func isJSON(data string) bool {
	var js json.RawMessage
	return json.Unmarshal([]byte(data), &js) == nil
}

// Add any additional paths that should be treated as multiline strings.
//
// Regex are allowed with `*` for single segment and `**` for multiple, if the
// path is not a regex, it will be considered an exact match.
var multilineStringFields = []string{
	// AWS
	"managed-ssh-keys/signer-cert",
	// GCP
	"instance/service-accounts/*/scopes",
	"instance/attributes/ssh-keys",
}

// isMultilineString checks if a path should be treated as a raw multiline string.
func isMultilineString(path string) bool {
	for _, pattern := range multilineStringFields {
		if matchRegex(pattern, path) {
			return true
		}
	}

	return false
}

// matchRegex accepts a pattern considered a regex and the path to match.
func matchRegex(pattern, path string) bool {
	re, err := regexp.Compile(patternToRegex(pattern))
	if err != nil {
		log.Trace().Err(err).
			Str("path", path).
			Str("pattern", pattern).
			Msg("os.id.metadata> failed to compile pattern")
		return false
	}
	return re.MatchString(path)
}

// Convert wildcard pattern to regex, or return exact match regex.
func patternToRegex(pattern string) string {
	if !strings.ContainsAny(pattern, "*?") {
		// No wildcards → Match exact string
		return "^" + regexp.QuoteMeta(pattern) + "$"
	}

	// Replace ** with .* (matches multiple path segments)
	pattern = strings.ReplaceAll(pattern, "**", ".*")

	// Replace * with [^/]+ (matches a single path segment)
	pattern = strings.ReplaceAll(pattern, "*", "[^/]+")

	// Ensure full match
	return "^" + pattern + "$"
}
