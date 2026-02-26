// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package windows

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseIntuneDeviceID(t *testing.T) {
	t.Run("parse Intune device ID", func(t *testing.T) {
		data := `{"EnrollmentGUID":"12345678-1234-1234-1234-123456789012","EntDMID":"abcdef12-3456-7890-abcd-ef1234567890"}`

		id, err := ParseIntuneDeviceID(strings.NewReader(data))
		require.NoError(t, err)
		assert.Equal(t, "abcdef12-3456-7890-abcd-ef1234567890", id)
	})

	t.Run("parse empty output", func(t *testing.T) {
		id, err := ParseIntuneDeviceID(strings.NewReader(""))
		require.NoError(t, err)
		assert.Equal(t, "", id)
	})

	t.Run("parse missing EntDMID", func(t *testing.T) {
		data := `{"EnrollmentGUID":"12345678-1234-1234-1234-123456789012","EntDMID":""}`

		id, err := ParseIntuneDeviceID(strings.NewReader(data))
		require.NoError(t, err)
		assert.Equal(t, "", id)
	})
}
