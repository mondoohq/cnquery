// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/okta/okta-sdk-golang/v6/okta"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
)

// TestAuthorizationServerJwksNilIsNullNotAnError pins the absence of a nil
// guard around the jwks conversion.
//
// Most authorization servers do not encrypt tokens, so entry.Jwks is nil on the
// common path. JsonToDict marshals that to the JSON literal `null` and
// unmarshals it into the result map, which encoding/json treats as a no-op
// rather than an error: the call returns a nil map and a nil error. The field
// then reports null, which is the correct reading for a server that has no
// encryption key set.
//
// Guarding the call would not fix a bug, and coercing the result to an empty
// map would introduce one: `{}` asserts that a key set was returned and was
// empty, which is a different claim from there being no key set at all.
func TestAuthorizationServerJwksNilIsNullNotAnError(t *testing.T) {
	t.Parallel()

	// The exact type and nilness the mapper sees for a server without
	// token encryption configured.
	var absent *okta.ResourceServerJsonWebKeys

	dict, err := convert.JsonToDict(absent)
	require.NoError(t, err,
		"a server without token encryption must not fail the listing")
	assert.Nil(t, dict, "an absent key set must read null, not an empty map")

	// A configured key set still converts.
	kid, kty := "k1", "RSA"
	present := &okta.ResourceServerJsonWebKeys{
		Keys: []okta.ResourceServerJsonWebKey{{Kid: &kid, Kty: &kty}},
	}

	dict, err = convert.JsonToDict(present)
	require.NoError(t, err)
	require.NotNil(t, dict)
	assert.Contains(t, dict, "keys")
}
