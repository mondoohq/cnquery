package packages_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/providers/os/connection/mock"
	"go.mondoo.com/cnquery/providers/os/resources/packages"
)

func TestOpkgListCommandParser(t *testing.T) {
	pkgList := `base-files - 169-50072
busybox - 1.24.2-1
dnsmasq - 2.78-1
dropbear - 2017.75-1
firewall - 2016-11-29-1`

	m := packages.ParseOpkgListPackagesCommand(strings.NewReader(pkgList))

	assert.Equal(t, 5, len(m), "detected the right amount of packages")
	var p packages.Package
	p = packages.Package{
		Name:    "busybox",
		Version: "1.24.2-1",
		Format:  packages.OpkgPkgFormat,
	}
	assert.Contains(t, m, p, "pkg detected")

	p = packages.Package{
		Name:    "dnsmasq",
		Version: "2.78-1",
		Format:  packages.OpkgPkgFormat,
	}
	assert.Contains(t, m, p, "pkg detected")

	p = packages.Package{
		Name:    "firewall",
		Version: "2016-11-29-1",
		Format:  packages.OpkgPkgFormat,
	}
	assert.Contains(t, m, p, "pkg detected")
}

func TestOpkgStatusParser(t *testing.T) {
	mock, err := mock.New("./testdata/packages_opkg_statusfile.toml", nil)
	require.NoError(t, err)
	f, err := mock.FileSystem().Open("/usr/lib/opkg/status")
	require.NoError(t, err)
	defer f.Close()

	m, err := packages.ParseOpkgPackages(f)
	require.NoError(t, err)
	assert.Equal(t, 8, len(m), "detected the right amount of packages")

	p := packages.Package{
		Name:        "libuci20130104",
		Version:     "2023-03-05-04d0c46c-1",
		Arch:        "x86_64",
		Status:      "install user installed",
		Origin:      "",
		Description: "",
		Format:      "opkg",
	}
	assert.Contains(t, m, p, "libuci20130104 detected")
}

func TestOpkgManager(t *testing.T) {
	filepath, _ := filepath.Abs("./testdata/packages_opkg.toml")
	conn, err := mock.New(filepath, &inventory.Asset{
		Platform: &inventory.Platform{
			Name: "openwrt",
		},
	})
	require.NoError(t, err)

	pkgManager, err := packages.ResolveSystemPkgManager(conn)
	require.NoError(t, err)

	pkgList, err := pkgManager.List()
	require.NoError(t, err)

	assert.Equal(t, 66, len(pkgList))
	p := packages.Package{
		Name:    "libjson-script",
		Version: "2016-11-29-77a629375d7387a33a59509d9d751a8798134cab",
		Format:  "opkg",
	}
	assert.Contains(t, pkgList, p, "pkg detected")
}
