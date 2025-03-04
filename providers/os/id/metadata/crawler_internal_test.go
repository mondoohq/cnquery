// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package metadata

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsJSON(t *testing.T) {
	assert.True(t, isJSON("{\"key\": \"value\"}"))
	assert.True(t, isJSON("[1,2,3]"))
	assert.False(t, isJSON("not json"))
	assert.False(t, isJSON(""))
	assert.False(t, isJSON("random text"))
	assert.True(t, isJSON("null"))
}

func TestIsMultilineString(t *testing.T) {
	assert.True(t, isMultilineString("managed-ssh-keys/signer-cert"))
	assert.True(t, isMultilineString("instance/service-accounts/default/scopes"))
	assert.False(t, isMultilineString("some/other/path"))
	assert.False(t, isMultilineString("other/path"))
}

func TestMatchRegex(t *testing.T) {
	assert.True(t, matchRegex("instance/service-accounts/*/scopes", "instance/service-accounts/default/scopes"))
	assert.False(t, matchRegex("instance/service-accounts/*/scopes", "instance/service-accounts/scopes"))
	assert.True(t, matchRegex("instance/attributes/ssh-keys", "instance/attributes/ssh-keys"))
	assert.False(t, matchRegex("instance/service-accounts/*/scopes", "instance/service-accounts/scopes"))
	assert.False(t, matchRegex("nonexistent/pattern", "random/path"))
}

func TestPatternToRegex(t *testing.T) {
	assert.Equal(t, "^exact/match$", patternToRegex("exact/match"))
	assert.Equal(t, "^wildcard/match/.[^/]+$", patternToRegex("wildcard/match/**"))
	assert.Equal(t, "^instance/service-accounts/[^/]+/scopes$", patternToRegex("instance/service-accounts/*/scopes"))
	assert.Equal(t, "^exact/path$", patternToRegex("exact/path"))
}
