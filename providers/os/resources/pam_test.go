// Copyright (c) Mondoo, Inc.
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
}
