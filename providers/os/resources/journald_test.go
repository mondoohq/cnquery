// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/testutils"
)

func TestResource_JournaldConfig(t *testing.T) {
	x.TestSimpleErrors(t, []testutils.SimpleTest{
		{
			Code:        "journald.config('nopath').sections",
			ResultIndex: 0,
			Expectation: "file 'nopath' not found",
		},
	})

	t.Run("journald file path", func(t *testing.T) {
		res := x.TestQuery(t, "journald.config.file.path")
		assert.NotEmpty(t, res)
		assert.NoError(t, res[0].Data.Error)
	})

	t.Run("journald sections", func(t *testing.T) {
		res := x.TestQuery(t, "journald.config.sections")
		assert.NotEmpty(t, res)
		assert.NoError(t, res[0].Data.Error)
	})

	t.Run("journald params", func(t *testing.T) {
		res := x.TestQuery(t, "journald.config.sections[0].params")
		assert.NotEmpty(t, res)
		assert.NoError(t, res[0].Data.Error)
	})

	t.Run("journald is downcasing relevant params", func(t *testing.T) {
		res := x.TestQuery(t, "journald.config.sections.where(name == 'Journal')[0].params.where(name == 'Compress')[0].value")
		assert.NotEmpty(t, res)
		assert.NoError(t, res[0].Data.Error)
		assert.Equal(t, "yes", res[0].Data.Value)
	})

	t.Run("journald is NOT downcasing other params", func(t *testing.T) {
		res := x.TestQuery(t, "journald.config.sections.where(name == 'Journal')[0].params.where(name == 'Storage')[0].value")
		assert.NotEmpty(t, res)
		assert.NoError(t, res[0].Data.Error)
		assert.Equal(t, "persistent", res[0].Data.Value)
	})
}
