// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestClient points a client at a handler, so behavior is asserted against
// responses the HTTP stack actually produced rather than hand-built structs.
func newTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return NewClient(srv.Client(), srv.URL, "NRAK-TEST-KEY")
}

func TestQueryDecodesData(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"data":{"actor":{"organization":{"id":"org-1","name":"Example"}}}}`)
	})

	var resp struct {
		Actor struct {
			Organization apiOrganizationForTest `json:"organization"`
		} `json:"actor"`
	}
	require.NoError(t, client.Query(context.Background(), "query{}", nil, &resp))
	assert.Equal(t, "org-1", resp.Actor.Organization.ID)
	assert.Equal(t, "Example", resp.Actor.Organization.Name)
}

type apiOrganizationForTest struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func TestQuerySendsKeyAndVariables(t *testing.T) {
	var seenKey, seenContentType string
	var body map[string]any

	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		seenKey = r.Header.Get("API-Key")
		seenContentType = r.Header.Get("Content-Type")
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &body)
		_, _ = io.WriteString(w, `{"data":{"ok":true}}`)
	})

	require.NoError(t, client.Query(context.Background(), "query($accountId: Int!){}", map[string]any{"accountId": 42}, nil))

	assert.Equal(t, "NRAK-TEST-KEY", seenKey, "the key travels in the API-Key header, not the URL")
	assert.Equal(t, "application/json", seenContentType)
	assert.Equal(t, "query($accountId: Int!){}", body["query"])
	assert.Equal(t, map[string]any{"accountId": float64(42)}, body["variables"])
}

// A NerdGraph response routinely carries data and errors together. Taking the
// data half would report a truncated collection as a complete one, so the call
// has to fail.
func TestQueryPartialDataWithErrorsFails(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{
			"data": {"actor": {"organization": {"id": "org-1"}}},
			"errors": [{"message":"nope","extensions":{"errorClass":"FORBIDDEN"}}]
		}`)
	})

	var resp map[string]any
	err := client.Query(context.Background(), "query{}", nil, &resp)
	require.Error(t, err)
	assert.True(t, IsForbidden(err))
	assert.Empty(t, resp, "no partial data may reach the caller")
}

func TestQueryHTTPStatusIsPreserved(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = io.WriteString(w, `{"error":"not authorized"}`)
	})

	err := client.Query(context.Background(), "query{}", nil, nil)
	require.Error(t, err)

	var statusErr *HTTPStatusError
	require.True(t, errors.As(err, &statusErr))
	assert.Equal(t, http.StatusForbidden, statusErr.StatusCode)
}

func TestQueryEmptyDataIsAnError(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{}`)
	})

	err := client.Query(context.Background(), "query{}", nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no data and no error")
}

func TestQueryMalformedBodyIsAnError(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `not json`)
	})

	err := client.Query(context.Background(), "query{}", nil, nil)
	require.Error(t, err)
	assert.False(t, IsForbidden(err), "a decode failure says nothing about permissions")
	assert.False(t, IsNotFound(err), "a decode failure is not an absence")
}

// transportFailure produces the error a dropped connection yields, by pointing
// a client at a server that has already been shut down.
func transportFailure(t *testing.T) error {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	client := NewClient(srv.Client(), srv.URL, "NRAK-TEST-KEY")
	srv.Close()

	err := client.Query(context.Background(), "query{}", nil, nil)
	require.Error(t, err)
	return err
}

func graphQLErr(classes ...string) error {
	entries := make([]GraphQLError, 0, len(classes))
	for _, class := range classes {
		entries = append(entries, GraphQLError{
			Message:    "something happened",
			Extensions: GraphQLErrorExtensions{ErrorClass: class},
		})
	}
	return &QueryError{Errors: entries}
}

func TestIsForbidden(t *testing.T) {
	t.Run("nil is not a definitive answer", func(t *testing.T) {
		assert.False(t, IsForbidden(nil))
	})

	// This is the case the classifier exists to get right. A network blip that
	// was read as "not permitted" would degrade the field to null and let an
	// audit pass on data nobody ever read.
	t.Run("transport failure is not a definitive answer", func(t *testing.T) {
		assert.False(t, IsForbidden(transportFailure(t)))
		assert.False(t, IsForbidden(errors.New("dial tcp: connection refused")))
		assert.False(t, IsForbidden(errors.New("403 Forbidden")), "the message text is not evidence")
	})

	t.Run("http statuses", func(t *testing.T) {
		assert.True(t, IsForbidden(&HTTPStatusError{StatusCode: http.StatusUnauthorized}))
		assert.True(t, IsForbidden(&HTTPStatusError{StatusCode: http.StatusForbidden}))
		assert.False(t, IsForbidden(&HTTPStatusError{StatusCode: http.StatusNotFound}))
		assert.False(t, IsForbidden(&HTTPStatusError{StatusCode: http.StatusInternalServerError}))
		assert.False(t, IsForbidden(&HTTPStatusError{StatusCode: http.StatusTooManyRequests}))
	})

	t.Run("graphql error classes", func(t *testing.T) {
		assert.True(t, IsForbidden(graphQLErr("UNAUTHORIZED")))
		assert.True(t, IsForbidden(graphQLErr("FORBIDDEN")))
		assert.True(t, IsForbidden(graphQLErr("VALIDATION_ERROR", "FORBIDDEN")))
		assert.False(t, IsForbidden(graphQLErr("NOT_FOUND")))
		assert.False(t, IsForbidden(graphQLErr("INTERNAL_SERVER_ERROR")))
		assert.False(t, IsForbidden(graphQLErr("")))
	})

	t.Run("wrapped errors still classify", func(t *testing.T) {
		assert.True(t, IsForbidden(fmt.Errorf("listing keys: %w", graphQLErr("FORBIDDEN"))))
		assert.True(t, IsForbidden(fmt.Errorf("listing keys: %w", &HTTPStatusError{StatusCode: 401})))
	})
}

func TestIsNotFound(t *testing.T) {
	assert.False(t, IsNotFound(nil))
	assert.False(t, IsNotFound(transportFailure(t)), "a dropped connection is not an absence")
	assert.False(t, IsNotFound(errors.New("404 not found")), "the message text is not evidence")

	assert.True(t, IsNotFound(&HTTPStatusError{StatusCode: http.StatusNotFound}))
	assert.False(t, IsNotFound(&HTTPStatusError{StatusCode: http.StatusForbidden}))

	assert.True(t, IsNotFound(graphQLErr("NOT_FOUND")))
	assert.False(t, IsNotFound(graphQLErr("FORBIDDEN")))
	assert.True(t, IsNotFound(fmt.Errorf("wrapped: %w", graphQLErr("NOT_FOUND"))))
}

func TestQueryErrorMessage(t *testing.T) {
	err := &QueryError{Errors: []GraphQLError{
		{Message: "no access", Extensions: GraphQLErrorExtensions{ErrorClass: "FORBIDDEN"}},
		{Message: "bad field"},
	}}
	assert.Equal(t, "the New Relic API reported: FORBIDDEN: no access; bad field", err.Error())
	assert.Equal(t, []string{"FORBIDDEN", ""}, err.Classes())

	empty := &QueryError{}
	assert.Contains(t, empty.Error(), "unspecified error")
}

func TestNormalizeRegion(t *testing.T) {
	tests := []struct {
		in       string
		region   string
		endpoint string
		wantErr  bool
	}{
		{"", RegionUS, endpointUS, false},
		{"us", RegionUS, endpointUS, false},
		{"US", RegionUS, endpointUS, false},
		{"  us  ", RegionUS, endpointUS, false},
		{"eu", RegionEU, endpointEU, false},
		{"EU", RegionEU, endpointEU, false},
		// An unknown region must not fall back to the US host. An EU account
		// read against the US host answers with an empty organization, which
		// reads as an account with no users and no keys.
		{"apac", "", "", true},
		{"us-east-1", "", "", true},
		{"eu-central", "", "", true},
	}

	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			region, endpoint, err := NormalizeRegion(tc.in)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.region, region)
			assert.Equal(t, tc.endpoint, endpoint)
		})
	}
}

func TestNewAccountIdentifier(t *testing.T) {
	assert.Equal(t,
		"//platformid.api.mondoo.app/runtime/newrelic/region/us/account/1234567",
		NewAccountIdentifier("us", 1234567))

	// The same account number in the two regions is two different accounts, so
	// the identifiers must differ or one asset would overwrite the other.
	assert.NotEqual(t, NewAccountIdentifier("us", 1234567), NewAccountIdentifier("eu", 1234567))
}

func TestNewAccountPlatform(t *testing.T) {
	pf := NewAccountPlatform("eu", 42)
	require.NotNil(t, pf)
	assert.Equal(t, "newrelic", pf.Name)
	assert.Equal(t, []string{"saas", "newrelic", "region", "eu", "account", "42"}, pf.TechnologyUrlSegments)
}

func TestMemoizeRunsOnce(t *testing.T) {
	conn := &NewrelicConnection{}

	var calls atomic.Int64
	fetch := func() (any, error) {
		calls.Add(1)
		return "value", nil
	}

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			val, err := conn.Memoize("k", fetch)
			assert.NoError(t, err)
			assert.Equal(t, "value", val)
		}()
	}
	wg.Wait()

	assert.Equal(t, int64(1), calls.Load())
}

// A failure is cached too. Without that, a permission failure would be retried
// by every field that touches the list, turning one refusal into a burst.
func TestMemoizeCachesFailure(t *testing.T) {
	conn := &NewrelicConnection{}

	var calls atomic.Int64
	fetch := func() (any, error) {
		calls.Add(1)
		return nil, errors.New("boom")
	}

	for i := 0; i < 5; i++ {
		_, err := conn.Memoize("k", fetch)
		require.Error(t, err)
	}
	assert.Equal(t, int64(1), calls.Load())
}

func TestMemoizeKeysAreIndependent(t *testing.T) {
	conn := &NewrelicConnection{}

	a, err := conn.Memoize("a", func() (any, error) { return 1, nil })
	require.NoError(t, err)
	b, err := conn.Memoize("b", func() (any, error) { return 2, nil })
	require.NoError(t, err)

	assert.Equal(t, 1, a)
	assert.Equal(t, 2, b)
}
