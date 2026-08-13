// Copyright Mondoo, Inc. 2026
// SPDX-License-Identifier: BUSL-1.1

package main

import (
	"fmt"
	"testing"
	"unicode/utf8"
)

func TestCommitTitle_ShortListsProviders(t *testing.T) {
	confs := updateConfs{
		{name: "aws", version: "13.53.1"},
		{name: "azure", version: "13.34.1"},
	}
	got := confs.commitTitle()
	want := "🎉 aws-13.53.1, azure-13.34.1"
	if got != want {
		t.Fatalf("commitTitle() = %q, want %q", got, want)
	}
}

func TestCommitTitle_LongFallsBackToCount(t *testing.T) {
	// Build enough providers that the enumerated title exceeds GitHub's limit.
	var confs updateConfs
	for i := 0; i < 72; i++ {
		confs = append(confs, &providerConf{
			name:    fmt.Sprintf("provider%02d", i),
			version: "13.0.1",
		})
	}

	got := confs.commitTitle()
	if utf8.RuneCountInString(got) > maxTitleLen {
		t.Fatalf("commitTitle() length = %d chars, exceeds limit of %d: %q",
			utf8.RuneCountInString(got), maxTitleLen, got)
	}
	want := "🎉 Release 72 providers"
	if got != want {
		t.Fatalf("commitTitle() = %q, want %q", got, want)
	}
}
