package gem_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mondoo.io/mondoo/resources/packs/core/gem"
	"go.mondoo.io/mondoo/vadvisor"
)

func TestGemfileLockParser(t *testing.T) {
	data, err := os.Open("./testdata/Gemfile.lock")
	if err != nil {
		t.Fatal(err)
	}

	pkgs, err := gem.ParseGemfileLock(data)
	assert.Nil(t, err)
	assert.Equal(t, 230, len(pkgs))

	assert.Contains(t, pkgs, &vadvisor.Package{
		Name:      "actioncable",
		Version:   "6.0.0.beta3",
		Format:    "gem",
		Namespace: "gem",
	})

	assert.Contains(t, pkgs, &vadvisor.Package{
		Name:      "zeitwerk",
		Version:   "2.0.0",
		Format:    "gem",
		Namespace: "gem",
	})
}

func TestParsePackagename(t *testing.T) {
	var name string
	var version string
	var err error

	name, version, err = gem.ParsePackagename("actioncable (6.0.0.beta3)")
	assert.Nil(t, err)
	assert.Equal(t, "actioncable", name)
	assert.Equal(t, "6.0.0.beta3", version)

	name, version, err = gem.ParsePackagename("activerecord-jdbcsqlite3-adapter (52.1-java)")
	assert.Nil(t, err)
	assert.Equal(t, "activerecord-jdbcsqlite3-adapter", name)
	assert.Equal(t, "52.1-java", version)

	name, version, err = gem.ParsePackagename(" aws-sdk-kms (1.11.0)")
	assert.Nil(t, err)
	assert.Equal(t, "aws-sdk-kms", name)
	assert.Equal(t, "1.11.0", version)
}
