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
	event := panicEvent("cnspec", "12.0.0", "abc123", "boom", []byte("stacktrace"), nil)

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

func TestPanicEventIncludesQueryTags(t *testing.T) {
	tags := QueryPanicTags("zLKUfd9hgBY=", "groups.map(name).containsAll(suRestrictedGroups)")
	event := panicEvent("cnspec", "12.0.0", "abc123", "boom", []byte("stacktrace"), tags)

	assert.Equal(t, "zLKUfd9hgBY=", event.Tags[TagQueryCodeID])
	assert.Equal(t, "groups.map(name).containsAll(suRestrictedGroups)", event.Tags[TagQuerySource])
	// baseline platform tags are preserved
	assert.Equal(t, runtime.GOOS, event.Tags["os"])
	assert.Equal(t, runtime.GOARCH, event.Tags["arch"])
}

func TestPanicEventPlatformTagsWin(t *testing.T) {
	event := panicEvent("cnspec", "12.0.0", "abc123", "boom", []byte("st"),
		map[string]string{"os": "spoofed", "arch": "spoofed"})

	assert.Equal(t, runtime.GOOS, event.Tags["os"])
	assert.Equal(t, runtime.GOARCH, event.Tags["arch"])
}

func TestQueryPanicTags(t *testing.T) {
	assert.Nil(t, QueryPanicTags("", ""))

	tags := QueryPanicTags("abc=", "")
	assert.Equal(t, map[string]string{TagQueryCodeID: "abc="}, tags)

	long := make([]rune, querySourceMax+100)
	for i := range long {
		long[i] = 'x'
	}
	tags = QueryPanicTags("", string(long))
	assert.Len(t, []rune(tags[TagQuerySource]), querySourceMax)
}
