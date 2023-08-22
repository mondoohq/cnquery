// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package php_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mondoo.com/cnquery/resources/packs/core/php"
	"go.mondoo.com/cnquery/upstream/mvd"
)

func TestComposerLockParser(t *testing.T) {
	data, err := os.Open("./testdata/drupal-composer.lock")
	if err != nil {
		t.Fatal(err)
	}

	pkgs, err := php.ParseComposerLock(data)
	assert.Nil(t, err)
	assert.Equal(t, 51, len(pkgs))

	assert.Contains(t, pkgs, &mvd.Package{
		Name:      "asm89/stack-cors",
		Version:   "1.2.0",
		Format:    "php",
		Namespace: "php",
	})

	assert.Contains(t, pkgs, &mvd.Package{
		Name:      "zendframework/zend-stdlib",
		Version:   "3.0.1",
		Format:    "php",
		Namespace: "php",
	})
}
