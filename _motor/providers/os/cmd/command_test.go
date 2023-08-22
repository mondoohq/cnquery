// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCommandResource(t *testing.T) {
	c := &CommandRunner{}

	if assert.NotNil(t, c) {
		cmd, err := c.Exec("echo", []string{"test"})
		assert.Equal(t, nil, err, "should execute without error")
		assert.Equal(t, "echo test", cmd.Command, "they should be equal")

		stdoutContent, _ := ioutil.ReadAll(cmd.Stdout)
		assert.Equal(t, "test\n", string(stdoutContent), "stdout output should be correct")
		stderrContent, _ := ioutil.ReadAll(cmd.Stderr)
		assert.Equal(t, "", string(stderrContent), "stderr output should be correct")
	}
}
