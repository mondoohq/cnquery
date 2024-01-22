// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseAzurePortRange(t *testing.T) {
	entry := "*,80,1024-65535"
	ranges := parseAzureSecurityRulePortRange(entry)
	assert.Equal(t, 3, len(ranges))
	assert.Equal(t, "*", ranges[0].FromPort)
	assert.Equal(t, "*", ranges[0].ToPort)
	assert.Equal(t, "80", ranges[1].FromPort)
	assert.Equal(t, "80", ranges[1].ToPort)
	assert.Equal(t, "1024", ranges[2].FromPort)
	assert.Equal(t, "65535", ranges[2].ToPort)
}
