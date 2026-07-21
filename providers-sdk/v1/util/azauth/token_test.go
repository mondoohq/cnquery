// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package azauth

import (
	"context"
	"errors"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetWorkloadIdentityToken(t *testing.T) {
	cred, err := GetWorkloadIdentityToken("tid", "cid", "/tmp/x.jwt")
	require.NoError(t, err)
	require.NotNil(t, cred)
}

type fakeCredential struct {
	err error
}

func (f *fakeCredential) GetToken(ctx context.Context, opts policy.TokenRequestOptions) (azcore.AccessToken, error) {
	if f.err != nil {
		return azcore.AccessToken{}, f.err
	}
	return azcore.AccessToken{Token: "token"}, nil
}

func TestGuidedCredential_EnrichesErrors(t *testing.T) {
	// the raw error azidentity's Azure CLI credential returns when the CLI
	// hands back something other than a token
	cliErr := errors.New("invalid character 'N' looking for beginning of value")

	t.Run("default chain with CLI JSON failure", func(t *testing.T) {
		cred := &guidedCredential{inner: &fakeCredential{err: cliErr}, usedDefaultChain: true}
		_, err := cred.GetToken(context.Background(), policy.TokenRequestOptions{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "az login")
		assert.Contains(t, err.Error(), "Azure CLI returned something other than a sign-in token")
		// original error is preserved
		assert.Contains(t, err.Error(), "invalid character 'N'")
	})

	t.Run("default chain with generic failure", func(t *testing.T) {
		cred := &guidedCredential{inner: &fakeCredential{err: errors.New("boom")}, usedDefaultChain: true}
		_, err := cred.GetToken(context.Background(), policy.TokenRequestOptions{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "No credentials were provided")
		assert.NotContains(t, err.Error(), "Azure CLI returned something other than")
		assert.Contains(t, err.Error(), "boom")
	})

	t.Run("explicit credentials failure", func(t *testing.T) {
		cred := &guidedCredential{inner: &fakeCredential{err: errors.New("boom")}, usedDefaultChain: false}
		_, err := cred.GetToken(context.Background(), policy.TokenRequestOptions{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "double-check the tenant ID, client ID")
		assert.Contains(t, err.Error(), "boom")
	})

	t.Run("success passes through untouched", func(t *testing.T) {
		cred := &guidedCredential{inner: &fakeCredential{}, usedDefaultChain: true}
		tk, err := cred.GetToken(context.Background(), policy.TokenRequestOptions{})
		require.NoError(t, err)
		assert.Equal(t, "token", tk.Token)
	})
}
