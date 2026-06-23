// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package mount

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseOptions(t *testing.T) {
	t.Run("flags and simple key=value", func(t *testing.T) {
		res := parseOptions("rw,relatime,seclabel,logbufs=8")
		assert.Equal(t, "", res["rw"])
		assert.Equal(t, "", res["relatime"])
		assert.Equal(t, "", res["seclabel"])
		assert.Equal(t, "8", res["logbufs"])
	})

	t.Run("value containing = is preserved", func(t *testing.T) {
		// An option whose value itself contains `=` must keep its full value
		// rather than being dropped to a valueless flag.
		res := parseOptions("rw,context=a=b=c,nosuid")
		assert.Equal(t, "a=b=c", res["context"])
		assert.Equal(t, "", res["rw"])
		assert.Equal(t, "", res["nosuid"])
	})
}
