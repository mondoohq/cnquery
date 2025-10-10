// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package lrcore

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/yaml"
)

var defaultLrDocsEntry = &LrDocsEntry{
	Fields:           map[string]*LrDocsField{},
	MinMondooVersion: "9.1.0",
}

func TestPlatformMapping(t *testing.T) {
	res := ensureDefaults("terraform.plan.configuration", defaultLrDocsEntry, "9.1.0", "9.1.0")
	assert.Equal(t, "terraform-plan", res.Platform.Name[0])

	res = ensureDefaults("terraform.plan.proposedChange", defaultLrDocsEntry, "9.1.0", "9.1.0")
	assert.Equal(t, "terraform-plan", res.Platform.Name[0])

	res = ensureDefaults("terraform.state.module", defaultLrDocsEntry, "9.1.0", "9.1.0")
	assert.Equal(t, "terraform-state", res.Platform.Name[0])

	res = ensureDefaults("terraform.block", defaultLrDocsEntry, "9.1.0", "9.1.0")
	assert.Equal(t, "terraform-hcl", res.Platform.Name[0])
}

func TestGenerateDocs(t *testing.T) {
	lrFile, err := os.ReadFile("testdata/new.lr")
	assert.NoError(t, err)
	existingDocs, err := os.ReadFile("testdata/existing-manifest.yaml")
	assert.NoError(t, err)
	var lrDocsData LrDocs
	err = yaml.Unmarshal(existingDocs, &lrDocsData)
	assert.NoError(t, err)
	lr := parse(t, string(lrFile))

	docs, err := lr.GenerateDocs("9.1.0", "9.1.0", lrDocsData)
	assert.NoError(t, err)
	assert.NotNil(t, docs)
	assert.Len(t, docs.Resources, 3)

	names := []string{}
	for k := range docs.Resources {
		names = append(names, k)
	}
	expected := []string{"sshd", "sshd.config", "auditd.config"}
	assert.ElementsMatch(t, expected, names)

	// verify min version for existing resources hasnt changed
	assert.Equal(t, "5.15.0", docs.Resources["sshd"].MinMondooVersion)
	assert.Equal(t, "5.15.0", docs.Resources["sshd.config"].MinMondooVersion)
	// new resource has the default min version
	assert.Equal(t, "9.1.0", docs.Resources["auditd.config"].MinMondooVersion)
}
