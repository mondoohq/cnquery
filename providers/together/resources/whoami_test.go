// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/providers/together/connection"
)

// whoamiFixture is shaped like the documented GET /whoami response. The values
// are invented; only the field names have to match the API.
const whoamiFixture = `{
  "api_key_id": "key-0000000000000000",
  "organization_id": "org-1111111111111111",
  "organization_name": "Example Org",
  "project_id": "proj-2222222222222222",
  "project_name": "Example Project",
  "project_slug": "example-project",
  "user_id": "user-3333333333333333"
}`

// newWhoamiTogether builds a together resource whose client talks to a stub
// serving body on /whoami. cliProject is the value the operator typed on the
// command line, which the identity fields must never echo back.
func newWhoamiTogether(t *testing.T, cliProject, body string, status int) (*mqlTogether, *int32) {
	t.Helper()

	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/whoami" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		atomic.AddInt32(&calls, 1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)

	conn, err := connection.NewTogetherConnection(1, &inventory.Asset{}, &inventory.Config{
		Options: map[string]string{
			connection.OptionToken:   "not-a-real-key",
			connection.OptionBaseURL: srv.URL,
			connection.OptionProject: cliProject,
		},
	})
	require.NoError(t, err)

	return &mqlTogether{MqlRuntime: &plugin.Runtime{Connection: conn}}, &calls
}

func TestWhoamiDecodesTheDocumentedResponse(t *testing.T) {
	r, _ := newWhoamiTogether(t, "", whoamiFixture, http.StatusOK)

	who, err := r.identity()
	require.NoError(t, err)

	assert.Equal(t, "key-0000000000000000", who.APIKeyID)
	assert.Equal(t, "org-1111111111111111", who.OrganizationID)
	assert.Equal(t, "Example Org", who.OrganizationName)
	assert.Equal(t, "proj-2222222222222222", who.ProjectID)
	assert.Equal(t, "Example Project", who.ProjectName)
	assert.Equal(t, "user-3333333333333333", who.UserID)
}

func TestIdentityFieldsComeFromTheServerNotTheCLIFlag(t *testing.T) {
	// The operator mistyped --project. Nothing they typed may reach a field.
	const typo = "proj-typo-not-a-real-project"
	r, _ := newWhoamiTogether(t, typo, whoamiFixture, http.StatusOK)

	for name, get := range map[string]func() (string, error){
		"organization":     r.organization,
		"organizationId":   r.organizationId,
		"organizationName": r.organizationName,
		"projectId":        r.projectId,
		"projectName":      r.projectName,
		"apiKeyId":         r.apiKeyId,
		"userId":           r.userId,
	} {
		got, err := get()
		require.NoError(t, err, name)
		assert.NotEqual(t, typo, got, "%s must report server state, not the --project flag", name)
	}

	org, err := r.organization()
	require.NoError(t, err)
	assert.Equal(t, "Example Org", org)

	projectID, err := r.projectId()
	require.NoError(t, err)
	assert.Equal(t, "proj-2222222222222222", projectID)
}

func TestOrganizationFallsBackToTheIdentifier(t *testing.T) {
	r, _ := newWhoamiTogether(t, "", `{
	  "api_key_id": "key-0000000000000000",
	  "organization_id": "org-1111111111111111",
	  "organization_name": "",
	  "project_id": "proj-2222222222222222",
	  "project_name": "Example Project",
	  "project_slug": "example-project"
	}`, http.StatusOK)

	org, err := r.organization()
	require.NoError(t, err)
	assert.Equal(t, "org-1111111111111111", org,
		"an account with no organization name still has to identify itself")

	userID, err := r.userId()
	require.NoError(t, err)
	assert.Empty(t, userID, "user_id is optional and stays empty rather than being invented")
}

func TestIdentityIsFetchedOnce(t *testing.T) {
	r, calls := newWhoamiTogether(t, "", whoamiFixture, http.StatusOK)

	for _, get := range []func() (string, error){
		r.organization, r.organizationId, r.organizationName,
		r.projectId, r.projectName, r.apiKeyId, r.userId,
	} {
		_, err := get()
		require.NoError(t, err)
	}

	assert.Equal(t, int32(1), atomic.LoadInt32(calls),
		"seven identity fields must not cost seven API calls")
}

func TestIdentityFailureIsReportedNotGuessed(t *testing.T) {
	r, _ := newWhoamiTogether(t, "proj-from-the-cli", `{"error":"unauthorized"}`, http.StatusUnauthorized)

	got, err := r.organization()
	assert.Error(t, err, "a key we cannot resolve must fail loudly, never fall back to user input")
	assert.Empty(t, got)
}
