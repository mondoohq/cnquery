package php_test

import (
	"go.mondoo.io/mondoo/vadvisor"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mondoo.io/mondoo/lumi/resources/php"
)

func TestComposerLockParser(t *testing.T) {
	data, err := os.Open("./testdata/drupal-composer.lock")
	if err != nil {
		t.Fatal(err)
	}

	pkgs, err := php.ParseComposerLock(data)
	assert.Nil(t, err)
	assert.Equal(t, 51, len(pkgs))

	assert.Contains(t, pkgs, &vadvisor.Package{
		Name:      "asm89/stack-cors",
		Version:   "1.2.0",
		Format:    "php",
		Namespace: "php",
	})

	assert.Contains(t, pkgs, &vadvisor.Package{
		Name:      "zendframework/zend-stdlib",
		Version:   "3.0.1",
		Format:    "php",
		Namespace: "php",
	})
}
