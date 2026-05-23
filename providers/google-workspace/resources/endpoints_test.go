// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/require"
	cloudidentity "google.golang.org/api/cloudidentity/v1"
)

func TestParseRFC3339(t *testing.T) {
	require.Nil(t, parseRFC3339(""))
	require.Nil(t, parseRFC3339("not a date"))

	got := parseRFC3339("2024-07-29T15:04:05Z")
	require.NotNil(t, got)
	require.Equal(t, "2024-07-29T15:04:05Z", got.UTC().Format("2006-01-02T15:04:05Z"))
}

func TestParseEndpointAdditionalSignals(t *testing.T) {
	// nil attrs is safe
	got := parseEndpointAdditionalSignals(nil)
	require.Nil(t, got.AvInstalled)
	require.Nil(t, got.AvEnabled)
	require.Nil(t, got.IsOsNativeFirewallEnabled)
	require.Nil(t, got.IsSecureBootEnabled)
	require.Equal(t, "", got.WindowsDomainName)

	// empty additional signals is safe
	got = parseEndpointAdditionalSignals(&cloudidentity.GoogleAppsCloudidentityDevicesV1EndpointVerificationSpecificAttributes{})
	require.Nil(t, got.AvInstalled)

	// malformed payload collapses to defaults, no panic
	got = parseEndpointAdditionalSignals(&cloudidentity.GoogleAppsCloudidentityDevicesV1EndpointVerificationSpecificAttributes{
		AdditionalSignals: []byte("not-json"),
	})
	require.Nil(t, got.AvInstalled)
	require.Equal(t, "", got.WindowsDomainName)

	// fully populated payload
	got = parseEndpointAdditionalSignals(&cloudidentity.GoogleAppsCloudidentityDevicesV1EndpointVerificationSpecificAttributes{
		AdditionalSignals: []byte(`{"av_installed":true,"av_enabled":false,"is_os_native_firewall_enabled":true,"is_secure_boot_enabled":false,"windows_domain_name":"CORP"}`),
	})
	require.NotNil(t, got.AvInstalled)
	require.True(t, *got.AvInstalled)
	require.NotNil(t, got.AvEnabled)
	require.False(t, *got.AvEnabled)
	require.NotNil(t, got.IsOsNativeFirewallEnabled)
	require.True(t, *got.IsOsNativeFirewallEnabled)
	require.NotNil(t, got.IsSecureBootEnabled)
	require.False(t, *got.IsSecureBootEnabled)
	require.Equal(t, "CORP", got.WindowsDomainName)

	// partial payload leaves missing booleans as nil (not false)
	got = parseEndpointAdditionalSignals(&cloudidentity.GoogleAppsCloudidentityDevicesV1EndpointVerificationSpecificAttributes{
		AdditionalSignals: []byte(`{"av_installed":true,"windows_domain_name":"WORKGROUP"}`),
	})
	require.NotNil(t, got.AvInstalled)
	require.True(t, *got.AvInstalled)
	require.Nil(t, got.AvEnabled, "device did not report av_enabled, should stay nil")
	require.Nil(t, got.IsSecureBootEnabled)
	require.Equal(t, "WORKGROUP", got.WindowsDomainName)
}
