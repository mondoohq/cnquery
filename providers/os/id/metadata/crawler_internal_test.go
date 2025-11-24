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
	assert.True(t, isMultilineString("meta-data/managed-ssh-keys/signer-cert"))
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
	assert.Equal(t, "wildcard/match/.[^/]+$", patternToRegex("wildcard/match/**"))
	assert.Equal(t, "instance/service-accounts/[^/]+/scopes$", patternToRegex("instance/service-accounts/*/scopes"))
	assert.Equal(t, "^exact/path$", patternToRegex("exact/path"))
}

func Test_detectMultilineString(t *testing.T) {
	bashScript := `
#!/bin/bash
exec > /var/log/user-data.log 2>&1
set -eux`

	gcpMetadata := `
attributes/
cpu-platform
credentials/
description
disks/
gce-workload-certificates/
guest-attributes/
hostname
id
image
licenses/
machine-type
maintenance-event
name
network-interfaces/
partner-attributes/`

	htmlErrorPage := `
<!DOCTYPE html>
<html lang=en>
  <meta charset=utf-8>
  <meta name=viewport content="initial-scale=1, minimum-scale=1, width=device-width">
  <title>Error 404 (Not Found)!!1</title>
  <style>
`

	tests := []struct {
		name     string
		data     string
		expected bool
	}{
		{
			name:     "ssh key - ed25519 with user prefix",
			data:     "debian:ssh-ed25519 AAAAExampleKey1234567890ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/=\n\ndebian:ssh-ed25519 AAAAExampleKey1234567890ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/\ndebian:ssh-ed25519 AAAAExampleKey1234567890ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+ Comment\ndebian:ssh-rsa AAAAExampleKey1234567890ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/=ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/=ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/=",
			expected: true,
		},
		{
			name:     "ssh key - rsa without user prefix",
			data:     "ssh-rsa AAAAExampleKey1234567890ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/=\nssh-rsa AAAAExampleKey1234567890ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/\nssh-ed25519 AAAAExampleKey1234567890ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+ Comment",
			expected: true,
		},
		{
			name:     "html error page",
			data:     htmlErrorPage,
			expected: true,
		},
		{
			name:     "bash script",
			data:     bashScript,
			expected: true,
		},

		{
			name:     "example GCP metadata",
			data:     gcpMetadata,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detectMultilineString(tt.data)
			assert.Equal(t, tt.expected, result, "DetectMultilineString returned %v, expected %v", result, tt.expected)
		})
	}
}
