// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestImageSignatureFromProperties(t *testing.T) {
	t.Run("nil properties returns empty string", func(t *testing.T) {
		assert.Equal(t, "", imageSignatureFromProperties(nil))
	})
	t.Run("non-map properties returns empty string", func(t *testing.T) {
		assert.Equal(t, "", imageSignatureFromProperties("not-a-map"))
	})
	t.Run("missing img_signature returns empty string", func(t *testing.T) {
		props := map[string]any{"os_distro": "ubuntu"}
		assert.Equal(t, "", imageSignatureFromProperties(props))
	})
	t.Run("null img_signature returns empty string", func(t *testing.T) {
		props := map[string]any{"img_signature": nil}
		assert.Equal(t, "", imageSignatureFromProperties(props))
	})
	t.Run("non-string img_signature returns empty string", func(t *testing.T) {
		props := map[string]any{"img_signature": 42}
		assert.Equal(t, "", imageSignatureFromProperties(props))
	})
	t.Run("string img_signature is returned", func(t *testing.T) {
		props := map[string]any{"img_signature": "c2lnbmF0dXJl"}
		assert.Equal(t, "c2lnbmF0dXJl", imageSignatureFromProperties(props))
	})
	t.Run("empty img_signature is returned as empty string", func(t *testing.T) {
		props := map[string]any{"img_signature": ""}
		assert.Equal(t, "", imageSignatureFromProperties(props))
	})
}
