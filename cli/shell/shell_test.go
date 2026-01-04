// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package shell_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mondoo.com/cnquery/v12/cli/shell"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/testutils"
)

func localShell() *shell.Shell {
	runtime := testutils.LinuxMock()
	res, err := shell.New(runtime)
	if err != nil {
		panic(err.Error())
	}

	return res
}

func TestShell_RunOnce(t *testing.T) {
	sh := localShell()
	assert.NotPanics(t, func() {
		sh.RunOnce("mondoo.build")
	}, "should not panic on partial queries")

	assert.NotPanics(t, func() {
		sh.RunOnce("mondoo { build version }")
	}, "should not panic on partial queries")

	assert.NotPanics(t, func() {
		sh.RunOnce("mondoo { _.version }")
	}, "should not panic on partial queries")
}

func TestShell_Centos8(t *testing.T) {
	sh := localShell()
	assert.NotPanics(t, func() {
		sh.RunOnce("platform { title name release arch }")
	}, "should not panic on partial queries")
}
