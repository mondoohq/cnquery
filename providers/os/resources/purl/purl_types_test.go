// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package purl_test

import (
	"testing"

	"github.com/package-url/packageurl-go"
	"github.com/stretchr/testify/assert"
	"go.mondoo.com/cnquery/v11/providers/os/resources/purl"
)

func TestValidType(t *testing.T) {
	t.Run("Valid types should return true", func(t *testing.T) {
		validTypes := []purl.Type{
			purl.TypeWindows, purl.TypeAppx, purl.TypeMacos, purl.TypeGeneric,
			purl.TypeApk, purl.TypeDebian, purl.TypeAlpm, purl.TypeRPM,
			purl.Type_X_Platform,
		}

		for _, validType := range validTypes {
			assert.True(t,
				purl.ValidType(validType),
				"Expected type %s to be valid", validType)
		}
	})

	t.Run("Invalid types should return false", func(t *testing.T) {
		invalidTypes := []purl.Type{"invalid", "unknown", purl.Type("random")}

		for _, invalidType := range invalidTypes {
			assert.False(t,
				purl.ValidType(invalidType),
				"Expected type %s to be invalid", invalidType)
		}
	})

	t.Run("Empty type should return false", func(t *testing.T) {
		assert.False(t,
			purl.ValidType(purl.Type("")),
			"Expected empty type to be invalid")
	})
}

func TestValidTypeString(t *testing.T) {
	t.Run("Valid type strings should return true", func(t *testing.T) {
		validTypes := []string{
			string(purl.TypeWindows), string(purl.TypeAppx), string(purl.TypeMacos),
			packageurl.TypeGeneric, packageurl.TypeApk, packageurl.TypeDebian,
			packageurl.TypeAlpm, packageurl.TypeRPM, "windows", "appx", "macos",
			"platform", string(purl.Type_X_Platform),
		}

		for _, validType := range validTypes {
			assert.True(t,
				purl.ValidTypeString(validType),
				"Expected type string %s to be valid", validType)
		}
	})

	t.Run("Invalid type strings should return false", func(t *testing.T) {
		invalidTypes := []string{"invalid", "unknown", "random"}

		for _, invalidType := range invalidTypes {
			assert.False(t, purl.ValidTypeString(invalidType), "Expected type string %s to be invalid", invalidType)
		}
	})

	t.Run("Empty type string should return false", func(t *testing.T) {
		assert.False(t,
			purl.ValidTypeString(""),
			"Expected empty type string to be invalid")
	})
}
