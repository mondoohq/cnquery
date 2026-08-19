// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	mondoovault "go.mondoo.com/mql/v13/providers-sdk/v1/vault"
	"go.mondoo.com/mql/v13/providers/confluent/connection"
)

// newAuditLogRoot builds a root resource whose connection points at a test
// server answering the account endpoint with the given status and body. The
// assertions below read the field state off the resource rather than off a
// helper, because that state is what the runtime hands a policy.
func newAuditLogRoot(t *testing.T, status int, body string) *mqlConfluent {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, auditLogPath, r.URL.Path)
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)

	conn, err := connection.NewConfluentConnection(1, &inventory.Asset{}, &inventory.Config{
		Options: map[string]string{
			connection.OptionAPIKey:  "cloud-key",
			connection.OptionBaseURL: srv.URL,
		},
		Credentials: []*mondoovault.Credential{
			{Type: mondoovault.CredentialType_password, Secret: []byte("cloud-secret")},
		},
	})
	require.NoError(t, err)

	runtime := plugin.NewRuntime(conn, nil, false, CreateResource, NewResource, GetData, SetData, nil)
	return &mqlConfluent{MqlRuntime: runtime}
}

// A payload carrying no audit log block has not answered the question. The
// field must come back resolved and null, never false: `enabled: false` on an
// organization that is audited is a clean pass on something nobody checked, and
// it fails in the direction nothing else would notice.
func TestAuditLogUnknownIsNullNotFalse(t *testing.T) {
	bodies := map[string]string{
		"no organization block":                    `{}`,
		"organization present, no audit log block": `{"organization":{"id":1,"name":"Acme"}}`,
		"null organization":                        `{"organization":null}`,
		"user with an empty organization":          `{"user":{"organization":{}}}`,
	}

	for name, body := range bodies {
		t.Run(name, func(t *testing.T) {
			root := newAuditLogRoot(t, http.StatusOK, body)

			config, err := root.auditLog()
			require.NoError(t, err)
			require.NotNil(t, config)

			assert.True(t, config.Enabled.IsSet(), "the field must be resolved")
			assert.True(t, config.Enabled.IsNull(), "an unanswered question must be null")
			assert.False(t, config.Enabled.Data, "and must not carry a value that reads as a real false")

			assert.True(t, config.TopicName.IsNull(), "the topic is unknown alongside the state")
		})
	}
}

// The API affirmatively reporting no writing service account is the only thing
// that may read as false.
func TestAuditLogAnsweredFalseIsFalse(t *testing.T) {
	root := newAuditLogRoot(t, http.StatusOK,
		`{"organization":{"audit_log":{"service_account_id":0,"topic_name":""}}}`)

	config, err := root.auditLog()
	require.NoError(t, err)
	require.NotNil(t, config)

	assert.True(t, config.Enabled.IsSet())
	assert.False(t, config.Enabled.IsNull(), "an answered question must not be null")
	assert.False(t, config.Enabled.Data)
	assert.False(t, config.TopicName.IsNull())
	assert.Equal(t, "", config.TopicName.Data)
}

func TestAuditLogAnsweredTrueIsTrue(t *testing.T) {
	root := newAuditLogRoot(t, http.StatusOK, `{"organization":{"audit_log":{
      "cluster_id":"lkc-audit",
      "account_id":"env-audit",
      "topic_name":"confluent-audit-log-events",
      "service_account_id":98765,
      "service_account_resource_id":"sa-audit"
    }}}`)

	config, err := root.auditLog()
	require.NoError(t, err)
	require.NotNil(t, config)

	assert.True(t, config.Enabled.IsSet())
	assert.False(t, config.Enabled.IsNull())
	assert.True(t, config.Enabled.Data)
	assert.Equal(t, "confluent-audit-log-events", config.TopicName.Data)
	assert.Equal(t, "lkc-audit", config.cachedClusterID)
	assert.Equal(t, "env-audit", config.cachedEnvironmentID)
	assert.Equal(t, "sa-audit", config.cachedServiceAccountID)
}

// A permission failure says the caller may not look, which is not evidence that
// there is nothing to see. The same applies to a server error or a 404 on an
// endpoint that is not part of the versioned API surface. None of them may
// become `enabled: false`.
func TestAuditLogFailedReadIsAnError(t *testing.T) {
	statuses := []int{
		http.StatusUnauthorized,
		http.StatusForbidden,
		http.StatusNotFound,
		http.StatusTooManyRequests,
		http.StatusInternalServerError,
		http.StatusBadGateway,
	}

	for _, status := range statuses {
		t.Run(http.StatusText(status), func(t *testing.T) {
			root := newAuditLogRoot(t, status, `{"errors":[{"code":"forbidden","detail":"nope"}]}`)

			config, err := root.auditLog()
			require.Error(t, err, "a failed read must not be turned into an answer")
			assert.Nil(t, config)
		})
	}
}

// A body that is not JSON at all, which is what a login redirect looks like, is
// a failed read rather than an organization without auditing.
func TestAuditLogUnparseableBodyIsAnError(t *testing.T) {
	root := newAuditLogRoot(t, http.StatusOK, `<html>login</html>`)

	config, err := root.auditLog()
	require.Error(t, err)
	assert.Nil(t, config)
}

// A numeric legacy account identifier names no environment this provider can
// resolve, so it must not be cached as one.
func TestAuditLogLegacyAccountIdIsNotAnEnvironment(t *testing.T) {
	root := newAuditLogRoot(t, http.StatusOK,
		`{"organization":{"audit_log":{"account_id":"12345","service_account_id":1,"topic_name":"t"}}}`)

	config, err := root.auditLog()
	require.NoError(t, err)
	require.NotNil(t, config)
	assert.Empty(t, config.cachedEnvironmentID)
}
