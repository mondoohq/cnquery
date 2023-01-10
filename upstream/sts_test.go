package upstream

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHash(t *testing.T) {
	claim1 := &Claims{
		Subject:  "foo",
		Resource: "bar",
		Exp:      "baz",
		Iat:      "quux",
	}

	h1a, err := HashClaimsSha256(claim1)
	require.NoError(t, err)
	h1b, err := HashClaimsSha256(claim1)
	require.NoError(t, err)
	require.Equal(t, h1a, h1b)

	claim2 := &Claims{
		Subject:  "f",
		Resource: "oobar",
		Exp:      "b",
		Iat:      "azquux",
	}

	h2, err := HashClaimsSha256(claim2)
	require.NoError(t, err)
	require.NotEqual(t, h1a, h2)
}
