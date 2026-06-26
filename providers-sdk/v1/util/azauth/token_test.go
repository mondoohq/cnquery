// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package azauth

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetWorkloadIdentityToken(t *testing.T) {
	cred, err := GetWorkloadIdentityToken("tid", "cid", "/tmp/x.jwt")
	require.NoError(t, err)
	require.NotNil(t, cred)
}
