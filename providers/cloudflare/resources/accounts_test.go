// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAccountRoles(t *testing.T) {
	env := setupTestEnv(t)
	acc := createTestAccount(t, env)

	env.Mux.HandleFunc(fmt.Sprintf("/accounts/%s/roles", testAccountID), func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		if page := r.URL.Query().Get("page"); page != "" && page != "1" {
			jsonResponse(w, `{"result":[],"success":true,"errors":[],"messages":[]}`)
			return
		}
		jsonResponse(w, loadFixture("account_roles"))
	})

	result, err := acc.roles()
	require.NoError(t, err)
	require.Len(t, result, 2)

	admin := result[0].(*mqlCloudflareAccountRole)
	assert.Equal(t, "role-admin", admin.Id.Data)
	assert.Equal(t, "Administrator", admin.Name.Data)
	assert.Equal(t, "Full access to all account resources", admin.Description.Data)

	readonly := result[1].(*mqlCloudflareAccountRole)
	assert.Equal(t, "role-readonly", readonly.Id.Data)
	assert.Equal(t, "Read Only", readonly.Name.Data)
}
