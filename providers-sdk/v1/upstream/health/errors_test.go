// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package health

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPanicEventIncludesPlatform(t *testing.T) {
	event := panicEvent("cnspec", "12.0.0", "abc123", "boom", []byte("stacktrace"))

	require.NotNil(t, event.Product)
	assert.Equal(t, "cnspec", event.Product.Name)
	assert.Equal(t, "12.0.0", event.Product.Version)
	assert.Equal(t, "abc123", event.Product.Build)

	require.NotNil(t, event.Error)
	assert.Equal(t, "panic: boom", event.Error.Message)
	assert.Equal(t, "stacktrace", event.Error.Stacktrace)

	assert.Equal(t, runtime.GOOS, event.Tags["os"])
	assert.Equal(t, runtime.GOARCH, event.Tags["arch"])
}
