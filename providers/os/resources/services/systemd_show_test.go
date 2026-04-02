// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package services

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseShowProperties(t *testing.T) {
	t.Run("basic key=value", func(t *testing.T) {
		input := strings.NewReader("Id=foo.socket\nDescription=Foo Socket\n")
		props, err := parseShowProperties(input)
		require.NoError(t, err)
		assert.Equal(t, "foo.socket", props["Id"])
		assert.Equal(t, "Foo Socket", props["Description"])
	})

	t.Run("duplicate keys are newline-joined", func(t *testing.T) {
		input := strings.NewReader(strings.Join([]string{
			"Listen=/run/foo.sock (Stream)",
			"Listen=[::]:80 (Stream)",
			"Accept=no",
			"",
		}, "\n"))
		props, err := parseShowProperties(input)
		require.NoError(t, err)
		assert.Equal(t, "/run/foo.sock (Stream)\n[::]:80 (Stream)", props["Listen"])
		assert.Equal(t, "no", props["Accept"])
	})
}
