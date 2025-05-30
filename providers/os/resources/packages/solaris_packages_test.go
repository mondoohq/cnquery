// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package packages

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v11/providers/os/connection/mock"
)

func TestFmriParser(t *testing.T) {
	// pkg://solaris/entire@0.5.11-0.175.2.0.0.34.0:20140303T182643Z
	// Name: entire
	// Publisher: solaris
	// Version: 0.5.11
	// Branch: 0.175.2.0.0.34.0

	sp, err := ParseSolarisFmri("pkg://solaris/entire@0.5.11,5.11-0.175.1.0.0.24.2:20120919T190135Z")
	require.NoError(t, err)
	assert.Equal(t, "entire", sp.Name)
	assert.Equal(t, "solaris", sp.Publisher)
	assert.Equal(t, "0.5.11", sp.Version)
	assert.Equal(t, "5.11-0.175.1.0.0.24.2", sp.Branch)

	// pkg://solaris/x11/library/libxscrnsaver@1.2.2,5.11-0.175.1.0.0.24.1317:20120904T180021Z
	// 	 vagrant@solaris-vagrant:~$ pkg info x11/library/libxscrnsaver
	//           Name: x11/library/libxscrnsaver
	//        Summary: libXss - X11 Screen Saver extension client library
	//    Description: Xlib-based client API for the MIT-SCREEN-SAVER extension to the
	//                 X11 protocol
	//       Category: System/X11
	//          State: Installed
	//      Publisher: solaris
	//        Version: 1.2.2
	//  Build Release: 5.11
	//         Branch: 0.175.1.0.0.24.1317
	// Packaging Date: September  4, 2012 06:00:21 PM
	//           Size: 101.36 kB
	//           FMRI: pkg://solaris/x11/library/libxscrnsaver@1.2.2,5.11-0.175.1.0.0.24.1317:20120904T180021Z

	sp, err = ParseSolarisFmri("pkg://solaris/x11/library/libxscrnsaver@1.2.2,5.11-0.175.1.0.0.24.1317:20120904T180021Z")
	require.NoError(t, err)
	assert.Equal(t, "x11/library/libxscrnsaver", sp.Name)
	assert.Equal(t, "solaris", sp.Publisher)
	assert.Equal(t, "1.2.2", sp.Version)
	assert.Equal(t, "5.11-0.175.1.0.0.24.1317", sp.Branch)
}

func TestSolarisPackageParser(t *testing.T) {
	pkgList := `FMRI                                                                         IFO
pkg://solaris/archiver/gnu-tar@1.26,5.11-0.175.1.0.0.24.0:20120904T170545Z   i--
pkg://solaris/compress/bzip2@1.0.6,5.11-0.175.1.0.0.24.0:20120904T170602Z    i--
pkg://solaris/compress/gzip@1.4,5.11-0.175.1.0.0.24.0:20120904T170603Z       i--
pkg://solaris/compress/p7zip@9.20.1,5.11-0.175.1.0.0.24.0:20120904T170605Z   i--`

	m := ParseSolarisPackages(strings.NewReader(pkgList))

	assert.Equal(t, 4, len(m), "detected the right amount of packages")
	p := Package{
		Name:    "archiver/gnu-tar",
		Version: "1.26",
		Format:  SolarisPkgFormat,
	}
	assert.Contains(t, m, p, "pkg detected")

	p = Package{
		Name:    "compress/bzip2",
		Version: "1.0.6",
		Format:  SolarisPkgFormat,
	}
	assert.Contains(t, m, p, "pkg detected")

	p = Package{
		Name:    "compress/p7zip",
		Version: "9.20.1",
		Format:  SolarisPkgFormat,
	}
	assert.Contains(t, m, p, "pkg detected")
}

func TestSolarisManager(t *testing.T) {
	filepath, _ := filepath.Abs("./testdata/packages_solaris11.toml")
	conn, err := mock.New(0, filepath, &inventory.Asset{
		Platform: &inventory.Platform{
			Name: "solaris",
		},
	})
	require.NoError(t, err)

	pms, err := ResolveSystemPkgManagers(conn)
	require.NoError(t, err)
	pkgManager := pms[0]

	pkgList, err := pkgManager.List()
	require.NoError(t, err)

	assert.Equal(t, 146, len(pkgList))
	p := Package{
		Name:    "compress/p7zip",
		Version: "9.20.1",
		Format:  SolarisPkgFormat,
	}
	assert.Contains(t, pkgList, p, "pkg detected")
}
