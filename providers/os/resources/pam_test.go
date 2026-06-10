// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResource_Pam(t *testing.T) {
	t.Run("with missing files", func(t *testing.T) {
		res := x.TestQuery(t, "pam.conf.content")
		assert.NotEmpty(t, res)
		assert.Error(t, res[0].Data.Error, "returned an error")
	})

	t.Run("exists is false without erroring when files are missing", func(t *testing.T) {
		res := x.TestQuery(t, "pam.conf.exists")
		assert.NotEmpty(t, res)
		assert.NoError(t, res[0].Data.Error)
		assert.Equal(t, false, res[0].Data.Value)
	})
}
