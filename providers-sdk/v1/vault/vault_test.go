// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package vault

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/testing/protocmp"
)

func TestSecretCredentialConversion(t *testing.T) {
	cred := &Credential{
		Type:     CredentialType_password,
		User:     "username",
		Password: "pass1",
	}
	cred.PreProcess()

	secret, err := NewSecret(cred, SecretEncoding_encoding_proto)
	require.NoError(t, err)

	cred2, err := secret.Credential()
	require.NoError(t, err)

	if d := cmp.Diff(cred, cred2, protocmp.Transform()); d != "" {
		t.Error("credentials are different", d)
	}
}
