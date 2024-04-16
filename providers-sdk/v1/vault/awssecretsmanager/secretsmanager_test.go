// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

//go:build debugtest
// +build debugtest

package awssecretsmanager

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/vault"
)

func TestAwsSecretsManager(t *testing.T) {
	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(ctx)
	require.NoError(t, err)
	v := New(cfg, WithKmsKey("alias/aws/secretsmanager"))

	cred := &vault.Secret{
		Data: []byte("my-secret-data"),
		Key:  "ivan-test-secret-2",
	}
	s, err := v.Set(ctx, cred)
	require.NoError(t, err)
	get, err := v.Get(ctx, &vault.SecretID{Key: s.Key})
	require.NoError(t, err)
	assert.Equal(t, cred.Data, get.Data)
}
