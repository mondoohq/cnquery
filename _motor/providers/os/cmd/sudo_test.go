// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSudo(t *testing.T) {
	s := NewSudo()
	cmd := s.Build("echo")
	assert.Equal(t, "sudo echo", cmd)
}
