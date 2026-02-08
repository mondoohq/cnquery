// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package shell_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mondoo.com/mql/v13/cli/shell"
	"go.mondoo.com/mql/v13/providers-sdk/v1/testutils"
)

func localShell() *shell.ShellProgram {
	runtime := testutils.LinuxMock()
	return shell.NewShell(runtime)
}

func TestShell_RunOnce(t *testing.T) {
	sh := localShell()
	assert.NotPanics(t, func() {
		_, _, _ = sh.RunOnce("mondoo.build")
	}, "should not panic on partial queries")

	assert.NotPanics(t, func() {
		_, _, _ = sh.RunOnce("mondoo { build version }")
	}, "should not panic on partial queries")

	assert.NotPanics(t, func() {
		_, _, _ = sh.RunOnce("mondoo { _.version }")
	}, "should not panic on partial queries")
}

func TestShell_Centos8(t *testing.T) {
	sh := localShell()
	assert.NotPanics(t, func() {
		_, _, _ = sh.RunOnce("platform { title name release arch }")
	}, "should not panic on partial queries")
}
