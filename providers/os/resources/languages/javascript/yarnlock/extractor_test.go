// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package yarnlock

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/providers/os/resources/languages"
	"go.mondoo.com/mql/v13/providers/os/resources/languages/javascript"
)

func TestYarnLockExtractor(t *testing.T) {
	f, err := os.Open("./testdata/d3-yarn.lock")
	require.NoError(t, err)
	defer f.Close()

	info, err := (&Extractor{}).Parse(f, "/path/to/yarn.lock")
	assert.Nil(t, err)

	list := info.Transitive()
	assert.Equal(t, 99, len(list))

	evidence := javascript.NewEvidenceList([]string{"/path/to/yarn.lock"})

	p := list.Find("has")
	assert.Equal(t, &languages.Package{
		Name:         "has",
		Version:      "1.0.3",
		Purl:         "pkg:npm/has@1.0.3",
		Cpes:         []string{"cpe:2.3:a:has:has:1.0.3:*:*:*:*:*:*:*"},
		EvidenceList: evidence,
	}, p)

	p = list.Find("iconv-lite")
	assert.Equal(t, &languages.Package{
		Name:         "iconv-lite",
		Version:      "0.4.24",
		Purl:         "pkg:npm/iconv-lite@0.4.24",
		Cpes:         []string{"cpe:2.3:a:iconv-lite:iconv-lite:0.4.24:*:*:*:*:*:*:*"},
		EvidenceList: evidence,
	}, p)
}
