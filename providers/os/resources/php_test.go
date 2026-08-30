// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/providers/os/resources/languages"
	"go.mondoo.com/mql/utils/syncx"
)

// The composer.lock and installed.json extractors read a license, but until
// php.package carried the field there was nowhere for it to land: the value was
// set on languages.Package and the resource never mapped it, so `php.packages`
// answered `cannot find field or resource 'license'`. This pins the mapping, so
// the extractor work stays reachable from MQL rather than only from the Go API.
func TestPhpPackageCarriesLicenseAndDescription(t *testing.T) {
	runtime := &plugin.Runtime{
		Resources: &syncx.Map[plugin.Resource]{},
		Callback:  &providerCallbacks{},
	}

	pkg, err := newPhpPackage(runtime, &languages.Package{
		Name:    "dual/licensed",
		Version: "1.0.0",
		Purl:    "pkg:composer/dual/licensed@1.0.0",
		// What LicenseExpression renders for a composer.lock list: the package
		// is offered under either, and the consumer chooses.
		License:     "(LGPL-2.1-only OR GPL-3.0-or-later)",
		Description: "Two licenses",
	})
	require.NoError(t, err)

	require.Equal(t, "(LGPL-2.1-only OR GPL-3.0-or-later)", pkg.License.Data)
	require.Equal(t, "Two licenses", pkg.Description.Data)
}

// A package whose manifest declares neither reports empty rather than carrying
// a value from somewhere else, and the fields are still set so a query reads ""
// instead of failing.
func TestPhpPackageWithoutLicenseIsEmpty(t *testing.T) {
	runtime := &plugin.Runtime{
		Resources: &syncx.Map[plugin.Resource]{},
		Callback:  &providerCallbacks{},
	}

	pkg, err := newPhpPackage(runtime, &languages.Package{
		Name:    "bare/package",
		Version: "2.0.0",
		Purl:    "pkg:composer/bare/package@2.0.0",
	})
	require.NoError(t, err)

	require.Empty(t, pkg.License.Data)
	require.Empty(t, pkg.Description.Data)
	require.True(t, pkg.License.IsSet(), "the field must be set, so a query reads \"\" rather than erroring")
}
