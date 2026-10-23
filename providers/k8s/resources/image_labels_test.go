// Copyright Mondoo, Inc. 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/providers-sdk/v1/inventory"
)

func TestImageLabelVerdicts(t *testing.T) {
	nsLabels := map[string]map[string]string{
		"team-alpha":     {"my.company.com/team": "alpha", "my.company.com/department": "d-1"},
		"team-beta":      {"my.company.com/team": "beta", "my.company.com/department": "d-1"},
		"platform-tools": {"my.company.com/department": "d-0"},
	}
	images := map[string][]string{
		"team-alpha":     {"nginx@sha256:shared", "web@sha256:alpha-only", "busybox@sha256:tools"},
		"team-beta":      {"nginx@sha256:shared"},
		"platform-tools": {"busybox@sha256:tools"},
	}
	got := imageLabelVerdicts(images, nsLabels, []string{"my.company.com/*"})

	assert.Equal(t, imageLabelVerdict{Labels: map[string]string{"my.company.com/team": "alpha", "my.company.com/department": "d-1"}}, got["web@sha256:alpha-only"])
	// team disagrees, department agrees: one value, marker names only the conflicting key
	assert.Equal(t, imageLabelVerdict{Labels: map[string]string{"my.company.com/department": "d-1"}, Conflicts: []string{"my.company.com/team"}}, got["nginx@sha256:shared"])
	// platform-tools lacks the team key and does not vote on it; it does vote on department
	assert.Equal(t, imageLabelVerdict{Labels: map[string]string{"my.company.com/team": "alpha"}, Conflicts: []string{"my.company.com/department"}}, got["busybox@sha256:tools"])
}

func TestImageLabelVerdictsNoVoters(t *testing.T) {
	got := imageLabelVerdicts(map[string][]string{"ns": {"img"}}, map[string]map[string]string{"ns": {"other": "x"}}, []string{"my.company.com/team"})
	assert.NotContains(t, got, "img")
}

func TestImageLabelVerdictsEncoding(t *testing.T) {
	v := map[string]imageLabelVerdict{"img": {Labels: map[string]string{"k": "v"}, Conflicts: []string{"c"}}}
	raw := encodeImageLabelVerdicts(v)
	got, skip, err := decodeImageLabelVerdicts(raw)
	require.NoError(t, err)
	assert.False(t, skip)
	assert.Equal(t, v, got)

	assert.Equal(t, "{}", encodeImageLabelVerdicts(nil))

	old := imageLabelVerdictsMaxBytes
	imageLabelVerdictsMaxBytes = 8
	defer func() { imageLabelVerdictsMaxBytes = old }()
	assert.Equal(t, imageLabelVerdictsSentinel, encodeImageLabelVerdicts(v))
	_, skip, err = decodeImageLabelVerdicts(imageLabelVerdictsSentinel)
	require.NoError(t, err)
	assert.True(t, skip)
}

func TestVerdictsForImages(t *testing.T) {
	all := map[string]imageLabelVerdict{"a": {Labels: map[string]string{"k": "1"}}, "b": {Labels: map[string]string{"k": "2"}}}
	assert.Equal(t, map[string]imageLabelVerdict{"a": all["a"]}, verdictsForImages(all, []string{"a", "missing"}))
}

func TestApplyImageLabelVerdicts(t *testing.T) {
	image := func(host string) *inventory.Asset {
		return &inventory.Asset{Connections: []*inventory.Config{{Type: registryImageConnectionType, Host: host}}}
	}
	labeled, conflicted, unknown := image("a"), image("b"), image("c")
	workload := &inventory.Asset{Name: "ns/pod", Labels: map[string]string{"app": "x"}, Connections: []*inventory.Config{{Type: "k8s"}}}
	applyImageLabelVerdicts([]*inventory.Asset{labeled, conflicted, unknown, workload}, map[string]imageLabelVerdict{
		"a": {Labels: map[string]string{"my.company.com/team": "alpha"}},
		"b": {Labels: map[string]string{"my.company.com/department": "d-1"}, Conflicts: []string{"my.company.com/dept", "my.company.com/team"}},
	})
	assert.Equal(t, map[string]string{"my.company.com/team": "alpha"}, labeled.Labels)
	assert.Equal(t, map[string]string{"my.company.com/department": "d-1", LabelNamespaceLabelConflict: "my.company.com/dept,my.company.com/team"}, conflicted.Labels)
	assert.Nil(t, unknown.Labels)
	assert.Equal(t, map[string]string{"app": "x"}, workload.Labels)
}
