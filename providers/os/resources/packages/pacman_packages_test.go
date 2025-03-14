// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package packages_test

import (
	"strings"
	"testing"

	"go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"

	"github.com/stretchr/testify/assert"
	"go.mondoo.com/cnquery/v11/providers/os/resources/packages"
)

func TestPacmanParser(t *testing.T) {
	pf := &inventory.Platform{
		Name:    "arch",
		Version: "",
		Arch:    "x86_64",
		Family:  []string{"arch", "linux", "unix", "os"},
		Labels: map[string]string{
			"distro-id": "arch",
		},
	}

	pkgList := `qpdfview 0.4.17beta1-4.1
usbmuxd 1.1.0+28+g46bdf3e-1
vertex-maia-themes 20171114-1
xfce4-power-manager 1.6.0.41.g9daecb5-1
xfce4-pulseaudio-plugin 0.3.2.r13.g553691a-1
zita-alsa-pcmi 0.2.0-3
zlib 1:1.2.11-2
zziplib 0.13.67-1`

	m := packages.ParsePacmanPackages(pf, strings.NewReader(pkgList))

	assert.Equal(t, 8, len(m), "detected the right amount of packages")
	p := packages.Package{
		Name:    "qpdfview",
		Version: "0.4.17beta1-4.1",
		PUrl:    "pkg:alpm/arch/qpdfview@0.4.17beta1-4.1?arch=x86_64&distro=arch",
		Format:  packages.PacmanPkgFormat,
	}
	assert.Contains(t, m, p, "pkg detected")

	p = packages.Package{
		Name:    "vertex-maia-themes",
		Version: "20171114-1",
		PUrl:    "pkg:alpm/arch/vertex-maia-themes@20171114-1?arch=x86_64&distro=arch",
		Format:  packages.PacmanPkgFormat,
	}
	assert.Contains(t, m, p, "pkg detected")

	p = packages.Package{
		Name:    "xfce4-pulseaudio-plugin",
		Version: "0.3.2.r13.g553691a-1",
		PUrl:    "pkg:alpm/arch/xfce4-pulseaudio-plugin@0.3.2.r13.g553691a-1?arch=x86_64&distro=arch",
		Format:  packages.PacmanPkgFormat,
	}
	assert.Contains(t, m, p, "pkg detected")
}

func TestPacmanWithWarningsParser(t *testing.T) {
	pf := &inventory.Platform{
		Name:    "arch",
		Version: "",
		Arch:    "x86_64",
		Family:  []string{"arch", "linux", "unix", "os"},
		Labels: map[string]string{
			"distro-id": "arch",
		},
	}

	pkgList := `warning: database file for 'core' does not exist (use '-Sy' to download)
warning: database file for 'extra' does not exist (use '-Sy' to download)
warning: database file for 'community' does not exist (use '-Sy' to download)
acl 2.2.53-2
archlinux-keyring 20200108-1
argon2 20190702-2`

	m := packages.ParsePacmanPackages(pf, strings.NewReader(pkgList))

	assert.Equal(t, 3, len(m), "detected the right amount of packages")
	p := packages.Package{
		Name:    "acl",
		Version: "2.2.53-2",
		PUrl:    "pkg:alpm/arch/acl@2.2.53-2?arch=x86_64&distro=arch",
		Format:  packages.PacmanPkgFormat,
	}
	assert.Contains(t, m, p, "pkg detected")
}
