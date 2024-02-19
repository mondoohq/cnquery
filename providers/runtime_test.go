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
	r := &Runtime{
		coordinator: mockC,
		recording:   NullRecording{},
		Provider: &ConnectedProvider{
			Instance: &RunningProvider{
				Name: "test",
			},
		},
	}

	// Make sure the runtime was removed from the coordinator
	mockC.EXPECT().RemoveRuntime(r).Times(1)

	// Close the runtime
	r.Close()

	// Make sure the runtime is closed and the schema is empty
	assert.True(t, r.isClosed)
}
