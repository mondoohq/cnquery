// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/portainer/client-api-go/v2/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/providers/portainer/connection"
)

func TestUnixTimePtr(t *testing.T) {
	// 0 means "unset" and must map to null (nil) rather than the 1970 epoch.
	assert.Nil(t, unixTimePtr(0))

	got := unixTimePtr(1700000000)
	require.NotNil(t, got)
	assert.Equal(t, int64(1700000000), got.Unix())
	assert.Equal(t, "UTC", got.Location().String())
}

func TestSecuritySettingsToDict(t *testing.T) {
	// nil overrides -> nil dict (field resolves to an empty dict).
	assert.Nil(t, securitySettingsToDict(nil))

	got := securitySettingsToDict(&models.PortainerEndpointSecuritySettings{
		AllowPrivilegedModeForRegularUsers: true,
		AllowSysctlSettingForRegularUsers:  true,
	})
	require.NotNil(t, got)
	assert.Equal(t, true, got["allowPrivilegedModeForRegularUsers"])
	assert.Equal(t, true, got["allowSysctlSettingForRegularUsers"])
	assert.Equal(t, false, got["allowBindMountsForRegularUsers"])
	// every documented override should be present
	assert.Len(t, got, 9)
}

func TestAccessPoliciesToDict(t *testing.T) {
	got := accessPoliciesToDict(map[string]models.PortainerAccessPolicy{
		"5": {RoleID: 1},
		"8": {RoleID: 3},
	})
	assert.Equal(t, int64(1), got["5"])
	assert.Equal(t, int64(3), got["8"])

	assert.Empty(t, accessPoliciesToDict(nil))
}

func TestMatchesEnvironmentTargets(t *testing.T) {
	const (
		dockerType = int64(1) // docker
		k8sType    = int64(5) // kubernetes
		edgeType   = int64(4) // edge-agent-docker
	)

	cases := []struct {
		name    string
		targets []string
		envType int64
		want    bool
	}{
		{"auto matches docker", []string{connection.DiscoveryAuto}, dockerType, true},
		{"all matches kubernetes", []string{connection.DiscoveryAll}, k8sType, true},
		{"environments matches edge", []string{connection.DiscoveryEnvironments}, edgeType, true},
		{"docker target matches docker", []string{connection.DiscoveryDocker}, dockerType, true},
		{"docker target rejects kubernetes", []string{connection.DiscoveryDocker}, k8sType, false},
		{"kubernetes target matches kubernetes", []string{connection.DiscoveryKubernetes}, k8sType, true},
		{"edge target matches edge", []string{connection.DiscoveryEdge}, edgeType, true},
		{"edge target rejects plain docker", []string{connection.DiscoveryEdge}, dockerType, false},
		{"no targets matches nothing", []string{}, dockerType, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, matchesEnvironmentTargets(tc.targets, tc.envType))
		})
	}
}
