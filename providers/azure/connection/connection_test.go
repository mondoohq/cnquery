// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
)

func TestSelectAzureCredential_WorkloadIdentity(t *testing.T) {
	// Set up an env var pointing to a (non-existent) token file.
	// Construction of WorkloadIdentityCredential does not read the file —
	// it is read lazily at GetToken time — so the file need not exist.
	t.Setenv("AZURE_FEDERATED_TOKEN_FILE", "/tmp/x.jwt")

	conf := &inventory.Config{
		Options: map[string]string{
			"tenant-id":       "tid",
			"client-id":       "cid",
			"subscription-id": "sub",
		},
		Credentials: nil,
	}

	cred, err := selectAzureCredential(conf)
	require.NoError(t, err)
	require.NotNil(t, cred)
}

func TestSelectAzureCredential_WorkloadIdentity_ViaOption(t *testing.T) {
	// Ensure env var is not set so we test the option path exclusively.
	os.Unsetenv("AZURE_FEDERATED_TOKEN_FILE")

	conf := &inventory.Config{
		Options: map[string]string{
			"tenant-id":                 "tid",
			"client-id":                 "cid",
			"subscription-id":           "sub",
			"azure-federated-token-file": "/tmp/x.jwt",
		},
		Credentials: nil,
	}

	cred, err := selectAzureCredential(conf)
	require.NoError(t, err)
	require.NotNil(t, cred)
}
