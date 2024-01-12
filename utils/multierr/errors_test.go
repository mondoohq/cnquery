// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package multierr_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mondoo.com/cnquery/v10/utils/multierr"
)

func TestMultiErr(t *testing.T) {
	t.Run("add nil errors", func(t *testing.T) {
		var e multierr.Errors
		e.Add(nil)
		e.Add(nil, nil, nil)
		assert.Nil(t, e.Deduplicate())
	})

	t.Run("add mixed errors", func(t *testing.T) {
		var e multierr.Errors
		e.Add(errors.New("1"), nil, errors.New("1"))
		var b multierr.Errors
		b.Add(errors.New("1"))
		assert.Equal(t, b.Deduplicate(), e.Deduplicate())
	})

	t.Run("test nil error deduplicate", func(t *testing.T) {
		var e multierr.Errors
		err := e.Deduplicate()
		assert.Nil(t, err)
	})
}
