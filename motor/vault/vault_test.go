package vault

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
	"go.mondoo.io/mondoo/motor/transports"
	"google.golang.org/protobuf/testing/protocmp"
)

func TestSecretCredentialConversion(t *testing.T) {
	cred := &transports.Credential{
		Type:     transports.CredentialType_password,
		User:     "username",
		Password: "pass1",
	}

	secret, err := NewSecret(cred)
	require.NoError(t, err)

	cred2, err := NewCredential(secret)
	require.NoError(t, err)

	if d := cmp.Diff(cred, cred2, protocmp.Transform()); d != "" {
		t.Error("credentials are different", d)
	}
}
