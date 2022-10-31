package packages

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"
)

func TestParseFreeBSDPackages(t *testing.T) {
	f, err := os.Open("testdata/freebsd-package-info-streaming.json")
	require.NoError(t, err)

	m, err := ParseFreeBSDPackages(f)
	require.Nil(t, err)
	assert.Equal(t, 2, len(m), "detected the right amount of packages")

	var p Package
	p = Package{
		Name:        "pkg",
		Version:     "1.18.4",
		Arch:        "freebsd:13:x86:64",
		Format:      "freebsd",
		Description: "Package management tool\n\nWWW: https://github.com/freebsd/pkg",
		Origin:      "ports-mgmt/pkg",
	}
	assert.Contains(t, m, p)
}
