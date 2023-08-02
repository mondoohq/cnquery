package updates

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mondoo.com/cnquery/motor/providers/mock"
)

// SUSE OS updates
func TestZypperPatchParser(t *testing.T) {
	mock, err := mock.NewFromTomlFile("./testdata/updates_zypper.toml")
	if err != nil {
		t.Fatal(err)
	}
	c, err := mock.RunCommand("zypper -n --xmlout list-updates -t patch")
	if err != nil {
		t.Fatal(err)
	}
	assert.Nil(t, err)

	m, err := ParseZypperPatches(c.Stdout)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(m), "detected the right amount of packages")

	assert.Equal(t, "openSUSE-2018-397", m[0].Name, "update name detected")
	assert.Equal(t, "moderate", m[0].Severity, "severity version detected")
}
