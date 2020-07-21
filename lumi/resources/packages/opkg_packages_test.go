package packages_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.io/mondoo/lumi/resources/packages"
	"go.mondoo.io/mondoo/motor"
	"go.mondoo.io/mondoo/motor/transports"
	"go.mondoo.io/mondoo/motor/transports/mock"
)

func TestOpkgParser(t *testing.T) {
	pkgList := `base-files - 169-50072
busybox - 1.24.2-1
dnsmasq - 2.78-1
dropbear - 2017.75-1
firewall - 2016-11-29-1`

	m := packages.ParseOpkgPackages(strings.NewReader(pkgList))

	assert.Equal(t, 5, len(m), "detected the right amount of packages")
	var p packages.Package
	p = packages.Package{
		Name:    "busybox",
		Version: "1.24.2-1",
		Format:  "opkg",
	}
	assert.Contains(t, m, p, "pkg detected")

	p = packages.Package{
		Name:    "dnsmasq",
		Version: "2.78-1",
		Format:  "opkg",
	}
	assert.Contains(t, m, p, "pkg detected")

	p = packages.Package{
		Name:    "firewall",
		Version: "2016-11-29-1",
		Format:  "opkg",
	}
	assert.Contains(t, m, p, "pkg detected")
}

func TestOpkgManager(t *testing.T) {
	filepath, _ := filepath.Abs("./testdata/packages_opkg.toml")
	trans, err := mock.NewFromToml(&transports.TransportConfig{Backend: transports.TransportBackend_CONNECTION_MOCK, Path: filepath})
	require.NoError(t, err)

	m, err := motor.New(trans)
	require.NoError(t, err)

	pkgManager, err := packages.ResolveSystemPkgManager(m)
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
