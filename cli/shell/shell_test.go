// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package shell_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mondoo.com/cnquery/v11/cli/shell"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/testutils"
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
	shell := localShell()
	assert.NotPanics(t, func() {
		shell.RunOnce("mondoo.build")
	}, "should not panic on partial queries")

	assert.NotPanics(t, func() {
		shell.RunOnce("mondoo { build version }")
	}, "should not panic on partial queries")

	assert.NotPanics(t, func() {
		shell.RunOnce("mondoo { _.version }")
	}, "should not panic on partial queries")
}

func TestShell_Help(t *testing.T) {
	shell := localShell()
	assert.NotPanics(t, func() {
		shell.ExecCmd("help")
	}, "should not panic on help command")

	assert.NotPanics(t, func() {
		shell.ExecCmd("help platform")
	}, "should not panic on help subcommand")
}

func TestShell_Centos8(t *testing.T) {
	shell := localShell()
	assert.NotPanics(t, func() {
		shell.RunOnce("platform { title name release arch }")
	}, "should not panic on partial queries")
}
