package packages_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mondoo.io/mondoo/lumi/resources/packages"
)

func TestPacmanParser(t *testing.T) {
	pkgList := `qpdfview 0.4.17beta1-4.1
usbmuxd 1.1.0+28+g46bdf3e-1
vertex-maia-themes 20171114-1
xfce4-power-manager 1.6.0.41.g9daecb5-1
xfce4-pulseaudio-plugin 0.3.2.r13.g553691a-1
zita-alsa-pcmi 0.2.0-3
zlib 1:1.2.11-2
zziplib 0.13.67-1`

	m := packages.ParsePacmanPackages(strings.NewReader(pkgList))

	assert.Equal(t, 8, len(m), "detected the right amount of packages")
	var p packages.Package
	p = packages.Package{
		Name:    "qpdfview",
		Version: "0.4.17beta1-4.1",
		Format:  "apk",
	}
	assert.Contains(t, m, p, "pkg detected")

	p = packages.Package{
		Name:    "vertex-maia-themes",
		Version: "20171114-1",
		Format:  "apk",
	}
	assert.Contains(t, m, p, "pkg detected")

	p = packages.Package{
		Name:    "xfce4-pulseaudio-plugin",
		Version: "0.3.2.r13.g553691a-1",
		Format:  "apk",
	}
	assert.Contains(t, m, p, "pkg detected")
}
