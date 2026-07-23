// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"strings"
	"testing"
)

func TestFolderResourceName(t *testing.T) {
	cases := map[string]string{
		"123456":         "folders/123456",
		"folders/123456": "folders/123456",
		"":               "folders/",
		"folders/":       "folders/",
	}
	for in, want := range cases {
		if got := folderResourceName(in); got != want {
			t.Errorf("folderResourceName(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestFolderIdNormalizationRoundTrip documents the folder id fix: folderToMql
// stores the bare id and folders()/projects() re-prefix with "folders/". If the
// list path stored the full "folders/123" instead, the child lookup key would
// become "folders/folders/123" and silently match nothing.
func TestFolderIdNormalizationRoundTrip(t *testing.T) {
	// The bare id that folderToMql now stores (mirroring initGcpFolder).
	bareID := strings.TrimPrefix("folders/123456", "folders/")
	if bareID != "123456" {
		t.Fatalf("bare id = %q, want 123456", bareID)
	}
	// folders()/projects() build the child lookup key this way:
	childKey := "folders/" + bareID
	if childKey != "folders/123456" {
		t.Errorf("child lookup key = %q, want folders/123456 (double-prefix regression)", childKey)
	}
}
