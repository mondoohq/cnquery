// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestActionsSecretID(t *testing.T) {
	tests := []struct {
		name                          string
		scope, owner, repo, env, secr string
		want                          string
	}{
		{
			name: "organization scope", scope: scopeOrganization,
			owner: "mondoohq", secr: "TOKEN",
			want: "github.actionsSecret/org/mondoohq/TOKEN",
		},
		{
			name: "repository scope", scope: scopeRepository,
			owner: "mondoohq", repo: "cnquery", secr: "TOKEN",
			want: "github.actionsSecret/repo/mondoohq/cnquery/TOKEN",
		},
		{
			name: "environment scope", scope: scopeEnvironment,
			owner: "mondoohq", repo: "cnquery", env: "prod", secr: "TOKEN",
			want: "github.actionsSecret/repo/mondoohq/cnquery/env/prod/TOKEN",
		},
		{
			name: "unknown scope falls back to repo", scope: "weird",
			owner: "mondoohq", repo: "cnquery", secr: "TOKEN",
			want: "github.actionsSecret/repo/mondoohq/cnquery/TOKEN",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, actionsSecretID(tc.scope, tc.owner, tc.repo, tc.env, tc.secr))
		})
	}
}

func TestActionsVariableID(t *testing.T) {
	tests := []struct {
		name                          string
		scope, owner, repo, env, vari string
		want                          string
	}{
		{
			name: "organization scope", scope: scopeOrganization,
			owner: "mondoohq", vari: "REGION",
			want: "github.actionsVariable/org/mondoohq/REGION",
		},
		{
			name: "repository scope", scope: scopeRepository,
			owner: "mondoohq", repo: "cnquery", vari: "REGION",
			want: "github.actionsVariable/repo/mondoohq/cnquery/REGION",
		},
		{
			name: "environment scope", scope: scopeEnvironment,
			owner: "mondoohq", repo: "cnquery", env: "prod", vari: "REGION",
			want: "github.actionsVariable/repo/mondoohq/cnquery/env/prod/REGION",
		},
		{
			name: "unknown scope falls back to repo", scope: "weird",
			owner: "mondoohq", repo: "cnquery", vari: "REGION",
			want: "github.actionsVariable/repo/mondoohq/cnquery/REGION",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, actionsVariableID(tc.scope, tc.owner, tc.repo, tc.env, tc.vari))
		})
	}
}

func TestHandleActionsListErr(t *testing.T) {
	t.Run("access denied is swallowed", func(t *testing.T) {
		res, err := handleActionsListErr(ghErrorResponse(http.StatusForbidden), "org secrets")
		assert.NoError(t, err)
		assert.Nil(t, res)
	})

	t.Run("not found is swallowed", func(t *testing.T) {
		res, err := handleActionsListErr(ghErrorResponse(http.StatusNotFound), "org secrets")
		assert.NoError(t, err)
		assert.Nil(t, res)
	})

	t.Run("real error is propagated", func(t *testing.T) {
		boom := errors.New("rate limited")
		res, err := handleActionsListErr(boom, "org secrets")
		assert.ErrorIs(t, err, boom)
		assert.Nil(t, res)
	})

	t.Run("server error is propagated", func(t *testing.T) {
		res, err := handleActionsListErr(ghErrorResponse(http.StatusInternalServerError), "org secrets")
		assert.Error(t, err)
		assert.Nil(t, res)
	})
}
