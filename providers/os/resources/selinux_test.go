// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1
package resources

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseSelinuxConfig(t *testing.T) {
	t.Run("standard config", func(t *testing.T) {
		content := `# This file controls the state of SELinux on the system.
# SELINUX= can take one of these three values:
#     enforcing - SELinux security policy is enforced.
#     permissive - SELinux prints warnings instead of enforcing.
#     disabled - No SELinux policy is loaded.
SELINUX=enforcing
# SELINUXTYPE= can take one of three values:
#     targeted - Targeted processes are protected,
#     minimum - Modification of targeted policy. Only selected processes are protected.
#     mls - Multi Level Security protection.
SELINUXTYPE=targeted
`
		mode, policyType := ParseSelinuxConfig(content)
		require.Equal(t, "enforcing", mode)
		require.Equal(t, "targeted", policyType)
	})

	t.Run("permissive mls", func(t *testing.T) {
		content := `SELINUX=permissive
SELINUXTYPE=mls
`
		mode, policyType := ParseSelinuxConfig(content)
		require.Equal(t, "permissive", mode)
		require.Equal(t, "mls", policyType)
	})

	t.Run("disabled", func(t *testing.T) {
		content := `SELINUX=disabled
SELINUXTYPE=targeted
`
		mode, policyType := ParseSelinuxConfig(content)
		require.Equal(t, "disabled", mode)
		require.Equal(t, "targeted", policyType)
	})

	t.Run("empty content", func(t *testing.T) {
		mode, policyType := ParseSelinuxConfig("")
		require.Equal(t, "", mode)
		require.Equal(t, "", policyType)
	})

	t.Run("comments only", func(t *testing.T) {
		mode, policyType := ParseSelinuxConfig("# SELINUX=enforcing\n# SELINUXTYPE=targeted\n")
		require.Equal(t, "", mode)
		require.Equal(t, "", policyType)
	})
}

func TestParseGetsebool(t *testing.T) {
	t.Run("standard output", func(t *testing.T) {
		output := `abrt_anon_write --> off
abrt_handle_event --> off
httpd_can_network_connect --> on
httpd_enable_cgi --> on
virt_sandbox_use_all_caps --> off
`
		bools := ParseGetsebool(output)
		require.Len(t, bools, 5)

		require.Equal(t, SELinuxBool{Name: "abrt_anon_write", Value: false}, bools[0])
		require.Equal(t, SELinuxBool{Name: "httpd_can_network_connect", Value: true}, bools[2])
		require.Equal(t, SELinuxBool{Name: "httpd_enable_cgi", Value: true}, bools[3])
	})

	t.Run("empty output", func(t *testing.T) {
		bools := ParseGetsebool("")
		require.Nil(t, bools)
	})
}

func TestParseSemodule(t *testing.T) {
	t.Run("simple format", func(t *testing.T) {
		output := `abrt
accountsd
apache
`
		modules := ParseSemodule(output)
		require.Len(t, modules, 3)

		require.Equal(t, SELinuxModule{Name: "abrt", Status: "enabled"}, modules[0])
		require.Equal(t, SELinuxModule{Name: "accountsd", Status: "enabled"}, modules[1])
		require.Equal(t, SELinuxModule{Name: "apache", Status: "enabled"}, modules[2])
	})

	t.Run("priority format", func(t *testing.T) {
		output := `100 abrt
100 accountsd
200 custom_policy
`
		modules := ParseSemodule(output)
		require.Len(t, modules, 3)

		require.Equal(t, SELinuxModule{Name: "abrt", Priority: 100, Status: "enabled"}, modules[0])
		require.Equal(t, SELinuxModule{Name: "custom_policy", Priority: 200, Status: "enabled"}, modules[2])
	})

	t.Run("full format with status", func(t *testing.T) {
		output := `100 abrt enabled
100 accountsd enabled
200 custom_policy disabled
`
		modules := ParseSemodule(output)
		require.Len(t, modules, 3)

		require.Equal(t, SELinuxModule{Name: "abrt", Priority: 100, Status: "enabled"}, modules[0])
		require.Equal(t, SELinuxModule{Name: "custom_policy", Priority: 200, Status: "disabled"}, modules[2])
	})

	t.Run("empty output", func(t *testing.T) {
		modules := ParseSemodule("")
		require.Nil(t, modules)
	})
}
