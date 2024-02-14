// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package providers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestRuntimeClose(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockC := NewMockProvidersCoordinator(ctrl)

	r := &Runtime{coordinator: mockC}
	assert.NotNil(t, r)
}
