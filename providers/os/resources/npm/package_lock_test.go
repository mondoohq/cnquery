// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package npm

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/upstream/mvd"
)

func TestPackageJsonLockParser(t *testing.T) {
	f, err := os.Open("./testdata/package-lock/workbox-package-lock.json")
	require.NoError(t, err)

	defer f.Close()

	pkgs, err := (&PackageLockParser{}).Parse(f)
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
