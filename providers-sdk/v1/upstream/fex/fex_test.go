// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package fex

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFexToDocument(t *testing.T) {
	f := &FindingExchange{Id: "finding-1"}

	doc := FexToDocument(f)

	require.NotNil(t, doc)
	require.Same(t, f, doc.GetFex())
	assert.Nil(t, doc.GetVex())
}

func TestFexToDocuments(t *testing.T) {
	fexes := []*FindingExchange{{Id: "a"}, {Id: "b"}}

	docs := FexToDocuments(fexes)

	require.Len(t, docs, 2)
	assert.Equal(t, "a", docs[0].GetFex().GetId())
	assert.Equal(t, "b", docs[1].GetFex().GetId())

	assert.Empty(t, FexToDocuments(nil))
}

func TestVexToDocument(t *testing.T) {
	v := &VulnerabilityExchange{Id: "CVE-2026-0001"}

	doc := VexToDocument(v)

	require.NotNil(t, doc)
	require.Same(t, v, doc.GetVex())
	assert.Nil(t, doc.GetFex())
}

func TestVexToDocuments(t *testing.T) {
	vexes := []*VulnerabilityExchange{{Id: "CVE-1"}, {Id: "CVE-2"}}

	docs := VexToDocuments(vexes)

	require.Len(t, docs, 2)
	assert.Equal(t, "CVE-1", docs[0].GetVex().GetId())
	assert.Equal(t, "CVE-2", docs[1].GetVex().GetId())

	assert.Empty(t, VexToDocuments(nil))
}

func TestNewRemediation(t *testing.T) {
	r := NewRemediation("upgrade", "bump to 1.2.3", "https://example.com/advisory")

	require.NotNil(t, r)
	assert.Equal(t, Remediation_Fix, r.GetCategory())
	assert.Equal(t, "upgrade", r.GetSummary())
	assert.Equal(t, "bump to 1.2.3", r.GetDetails())
	assert.Equal(t, "https://example.com/advisory", r.GetUrl())
}
