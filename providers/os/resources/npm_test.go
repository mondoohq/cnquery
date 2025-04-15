// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v12/llx"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v12/providers/os/connection/fs"
	"go.mondoo.com/cnquery/v12/types"
	"go.mondoo.com/cnquery/v12/utils/syncx"
)

func TestNpmPackage_unique(t *testing.T) {
	mockFS := afero.NewMemMapFs()
	// create test files and directories
	err := mockFS.MkdirAll("/usr/local/lib/node_modules/generator-code", 0o755)
	require.NoError(t, err)
	err = mockFS.MkdirAll("/usr/local/lib/node_modules/yo", 0o755)
	require.NoError(t, err)

	// Read package.json files from testdata
	yoPkg, err := os.ReadFile(filepath.Join("packages", "testdata", "yo_package.json"))
	require.NoError(t, err)
	require.NotNil(t, yoPkg)

	gcPkg, err := os.ReadFile(filepath.Join("packages", "testdata", "gc_package.json"))
	require.NoError(t, err)
	require.NotNil(t, gcPkg)

	err = afero.WriteFile(mockFS, "/usr/local/lib/node_modules/generator-code/package.json", gcPkg, 0o644)
	require.NoError(t, err)
	err = afero.WriteFile(mockFS, "/usr/local/lib/node_modules/yo/package.json", yoPkg, 0o644)
	require.NoError(t, err)

	conn, err := fs.NewFileSystemConnectionWithFs(0, &inventory.Config{}, &inventory.Asset{}, "", nil, mockFS)
	require.NoError(t, err)

	r := &plugin.Runtime{
		Resources:  &syncx.Map[plugin.Resource]{},
		Connection: conn,
		Callback:   &providerCallbacks{},
	}
	mqlNpm := &mqlNpmPackages{
		MqlRuntime: r,
	}

	// Create resources from filesystem
	err = mqlNpm.gatherData()
	require.NoError(t, err)

	// Check that we have 4 packages
	require.Equal(t, 4, len(mqlNpm.List.Data))

	// Check that the first package is yosay
	pkg2 := mqlNpm.List.Data[2].(*mqlNpmPackage)
	require.Equal(t, "yosay", pkg2.Name.Data)
	require.Equal(t, "^2.0.2", pkg2.Version.Data)

	// Check that the third package is also yosay, but with a different version
	pkg3 := mqlNpm.List.Data[3].(*mqlNpmPackage)
	require.Equal(t, "yosay", pkg3.Name.Data)
	require.Equal(t, "^3.0.0", pkg3.Version.Data)

	// To get the correct data, we need distinct IDs
	require.NotEqual(t, pkg2.MqlID(), pkg3.MqlID())
	require.Equal(t, "yosay/usr/local/lib/node_modules/yo/package.json", pkg2.MqlID())
	require.Equal(t, "yosay/usr/local/lib/node_modules/generator-code/package.json", pkg3.MqlID())
}

// Mock callbacks for testing
// These are needed during calls to CreateSharedResource
type providerCallbacks struct{}

func (p *providerCallbacks) GetData(req *plugin.DataReq) (*plugin.DataRes, error) {
	return &plugin.DataRes{
		Data: &llx.Primitive{
			Type:  string(types.Resource(req.Resource)),
			Value: []byte("not of interest"),
		},
	}, nil
}

func (p *providerCallbacks) GetRecording(req *plugin.DataReq) (*plugin.ResourceData, error) {
	res := plugin.ResourceData{}
	return &res, nil
}

func (p *providerCallbacks) Collect(req *plugin.DataRes) error {
	return nil
}
