package npm_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mondoo.com/cnquery/resources/packs/os/npm"
	"go.mondoo.com/cnquery/upstream/mvd"
)

func TestYarnParser(t *testing.T) {
	data, err := os.Open("./testdata/d3-yarn.lock")
	if err != nil {
		t.Fatal(err)
	}

	pkgs, err := npm.ParseYarnLock(data)
	assert.Nil(t, err)
	assert.Equal(t, 99, len(pkgs))

	assert.Contains(t, pkgs, &mvd.Package{
		Name:      "has",
		Version:   "1.0.3",
		Format:    "npm",
		Namespace: "nodejs",
	})

	assert.Contains(t, pkgs, &mvd.Package{
		Name:      "iconv-lite",
		Version:   "0.4.24",
		Format:    "npm",
		Namespace: "nodejs",
	})
}

func TestParsePackagename(t *testing.T) {
	var name string
	var version string
	var err error

	name, version, err = npm.ParseYarnPackageName("source-map-support@~0.5.10")
	assert.Nil(t, err)
	assert.Equal(t, "source-map-support", name)
	assert.Equal(t, "~0.5.10", version)

	name, version, err = npm.ParseYarnPackageName("@types/node@*")
	assert.Nil(t, err)
	assert.Equal(t, "@types/node", name)
	assert.Equal(t, "*", version)

	name, version, err = npm.ParseYarnPackageName("@babel/code-frame@^7.0.0-beta.47")
	assert.Nil(t, err)
	assert.Equal(t, "@babel/code-frame", name)
	assert.Equal(t, "^7.0.0-beta.47", version)

	name, version, err = npm.ParseYarnPackageName("has@^1.0.1, has@^1.0.3, has@~1.0.3")
	assert.Nil(t, err)
	assert.Equal(t, "has", name)
	assert.Equal(t, "^1.0.1", version)
}
