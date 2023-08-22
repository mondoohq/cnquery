// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

//go:build debugtest
// +build debugtest

package aws

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/motor/providers"
)

func TestAwsTransport(t *testing.T) {
	pCfg := &providers.Config{
		Backend: providers.ProviderType_AWS,
		Options: map[string]string{
			"profile": "example-demo",
		},
	}

	p, err := New(pCfg, TransportOptions(pCfg.Options)...)
	require.NoError(t, err)

	info, err := p.Account()
	require.NoError(t, err)
	assert.NotNil(t, info)
}
