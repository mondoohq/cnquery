// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

//go:build debugtest
// +build debugtest

package aws

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	aws_provider "go.mondoo.com/cnquery/motor/providers/aws"
)

func TestEC2Discovery(t *testing.T) {
	pCfg := &providers.Config{
		Type: providers.ProviderType_AWS,
		Options: map[string]string{
			"profile": "mondoo-demo",
			"region":  "us-east-1",
		},
	}

	p, err := aws_provider.New(pCfg, aws_provider.TransportOptions(pCfg.Options)...)
	require.NoError(t, err)

	r, err := NewEc2Discovery(p.Config())
	require.NoError(t, err)

	assets, err := r.List()
	require.NoError(t, err)
	assert.True(t, len(assets) > 0)
}
