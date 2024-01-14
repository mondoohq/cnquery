// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package npm_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/upstream/mvd"
	"go.mondoo.com/cnquery/v10/providers/os/resources/npm"
)

func TestPackageJsonParser(t *testing.T) {
	data, err := os.Open("./testdata/express-package.json")
	if err != nil {
		t.Fatal(err)
	}

	pkgs, err := npm.ParsePackageJson(data)
	assert.Nil(t, err)
	assert.Equal(t, 31, len(pkgs))

	assert.Contains(t, pkgs, &mvd.Package{
		Name:      "path-to-regexp",
		Version:   "0.1.7",
		Format:    "npm",
		Namespace: "nodejs",
	})

	// "range-parser": "~1.2.0",
	// TODO: we need to be better at version ranges
	assert.Contains(t, pkgs, &mvd.Package{
		Name:      "range-parser",
		Version:   "~1.2.0",
		Format:    "npm",
		Namespace: "nodejs",
	})
}

func TestPackageJsonLockParser(t *testing.T) {
	data, err := os.Open("./testdata/workbox-package-lock.json")
	if err != nil {
		t.Fatal(err)
	}

	pkgs, err := npm.ParsePackageJsonLock(data)
	assert.Nil(t, err)
	assert.Equal(t, 1300, len(pkgs))

	assert.Contains(t, pkgs, &mvd.Package{
		Name:      "@babel/generator",
		Version:   "7.0.0",
		Format:    "npm",
		Namespace: "nodejs",
	})

	assert.Contains(t, pkgs, &mvd.Package{
		Name:      "@lerna/changed",
		Version:   "3.3.2",
		Format:    "npm",
		Namespace: "nodejs",
	})
}
