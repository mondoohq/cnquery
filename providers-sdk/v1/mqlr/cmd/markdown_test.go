// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSanitizeComments(t *testing.T) {
	comments := []string{
		"",
		"// ",
		"// normal comment",
		"// normal comment | delimiter",
		"// normal comment \\| pre-escaped delimiter",
	}
	expected := []string{
		"",
		"",
		"normal comment",
		"normal comment \\| delimiter",
		"normal comment \\| pre-escaped delimiter",
	}

	actual := sanitizeComments(comments)
	assert.ElementsMatch(t, expected, actual)
}
