package cmd

import (
	"testing"

	"gotest.tools/assert"
)

func TestSudo(t *testing.T) {
	s := NewSudo()
	cmd := s.Build("echo")
	assert.Equal(t, "sudo echo", cmd)
}
