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
	Fields:             map[string]*LrDocsField{},
	MinProviderVersion: "13.0.0",
	MinMondooVersion:   "9.1.0",
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

	docs, err := lr.GenerateDocs("13.0.0", "9.1.0", lrDocsData)
	assert.NoError(t, err)
	assert.NotNil(t, docs)
	assert.Len(t, docs.Resources, 3)

	names := []string{}
	for k := range docs.Resources {
		names = append(names, k)
	}
	expected := []string{"sshd", "sshd.config", "auditd.config"}
	assert.ElementsMatch(t, expected, names)

	// verify min_mondoo_version for existing resources hasn't changed
	assert.Equal(t, "5.15.0", docs.Resources["sshd"].MinMondooVersion)
	assert.Equal(t, "5.15.0", docs.Resources["sshd.config"].MinMondooVersion)
	// new resource gets default min_mondoo_version
	assert.Equal(t, "9.1.0", docs.Resources["auditd.config"].MinMondooVersion)

	// verify min_provider_version is set for all resources (none had it before)
	assert.Equal(t, "13.0.0", docs.Resources["sshd"].MinProviderVersion)
	assert.Equal(t, "13.0.0", docs.Resources["sshd.config"].MinProviderVersion)
	assert.Equal(t, "13.0.0", docs.Resources["auditd.config"].MinProviderVersion)
}

func TestGenerateDocs_PreservesExistingProviderVersion(t *testing.T) {
	lrFile, err := os.ReadFile("testdata/new.lr")
	assert.NoError(t, err)

	// simulate existing manifest that already has min_provider_version set
	existingDocs := LrDocs{
		Resources: map[string]*LrDocsEntry{
			"sshd": {
				Fields:             map[string]*LrDocsField{},
				MinProviderVersion: "12.0.0",
				MinMondooVersion:   "5.15.0",
			},
			"sshd.config": {
				Fields: map[string]*LrDocsField{
					"ciphers": {MinProviderVersion: "11.0.0"},
				},
				MinProviderVersion: "11.0.0",
				MinMondooVersion:   "5.15.0",
			},
		},
	}

	lr := parse(t, string(lrFile))
	docs, err := lr.GenerateDocs("13.0.0", "9.0.0", existingDocs)
	assert.NoError(t, err)

	// existing min_provider_version must not be overwritten
	assert.Equal(t, "12.0.0", docs.Resources["sshd"].MinProviderVersion)
	assert.Equal(t, "11.0.0", docs.Resources["sshd.config"].MinProviderVersion)

	// existing field-level min_provider_version preserved (scrubbed because it matches resource)
	assert.Equal(t, "", docs.Resources["sshd.config"].Fields["ciphers"].MinProviderVersion)

	// new resource gets the current version
	assert.Equal(t, "13.0.0", docs.Resources["auditd.config"].MinProviderVersion)
	assert.Equal(t, "9.0.0", docs.Resources["auditd.config"].MinMondooVersion)
}
