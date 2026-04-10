// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsControlTowerNotConfiguredError(t *testing.T) {
	t.Run("nil error returns false", func(t *testing.T) {
		assert.False(t, isControlTowerNotConfiguredError(nil))
	})

	t.Run("unrelated error returns false", func(t *testing.T) {
		assert.False(t, isControlTowerNotConfiguredError(errors.New("some random error")))
	})

	t.Run("missing AWSControlTowerAdmin role", func(t *testing.T) {
		err := fmt.Errorf("operation error ControlTower: ListEnabledBaselines, https response error StatusCode: 400, ValidationException: AWS Control Tower could not complete the operation because it could not assume the 'AWSControlTowerAdmin' role. Check the configuration for this role and try again.")
		assert.True(t, isControlTowerNotConfiguredError(err))
	})
}
