// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testClient(t *testing.T, handler http.HandlerFunc) (*Client, *httptest.Server) {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return NewClient(server.URL, "token-abc:secret", server.Client()), server
}

func TestListSinglePage(t *testing.T) {
	client, _ := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer token-abc:secret", r.Header.Get("Authorization"))
		assert.Equal(t, "1000", r.URL.Query().Get("limit"),
			"the walk asks for an explicit page size")
		_, _ = w.Write([]byte(`{"type":"collection","data":[{"id":"a"},{"id":"b"}]}`))
	})

	records, err := client.List(context.Background(), "/v3/clusters")
	require.NoError(t, err)
	assert.Len(t, records, 2)
}

func TestListFollowsPagination(t *testing.T) {
	var server *httptest.Server
	var calls int32

	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page := atomic.AddInt32(&calls, 1)
		switch page {
		case 1:
			fmt.Fprintf(w, `{"data":[{"id":"a"}],"pagination":{"limit":1,"next":%q}}`,
				server.URL+"/v3/clusters?limit=1&marker=m2")
		case 2:
			fmt.Fprintf(w, `{"data":[{"id":"b"}],"pagination":{"limit":1,"next":%q}}`,
				server.URL+"/v3/clusters?limit=1&marker=m3")
		default:
			fmt.Fprint(w, `{"data":[{"id":"c"}],"pagination":{"limit":1}}`)
		}
	}))
	t.Cleanup(server.Close)

	client := NewClient(server.URL, "t", server.Client())
	records, err := client.List(context.Background(), "/v3/clusters")
	require.NoError(t, err)

	require.Len(t, records, 3)
	assert.Equal(t, int32(3), atomic.LoadInt32(&calls))

	ids := []string{}
	for _, record := range records {
		var entry struct {
			ID string `json:"id"`
		}
		require.NoError(t, json.Unmarshal(record, &entry))
		ids = append(ids, entry.ID)
	}
	assert.Equal(t, []string{"a", "b", "c"}, ids)
}

func TestListStuckCursorFails(t *testing.T) {
	// A server that ignores its own marker hands back the page it already gave.
	// Walking it would repeat every record until the page cap, so the listing
	// has to fail loudly. Truncating instead would produce a shorter list that
	// satisfies every assertion written against it.
	var server *httptest.Server
	var calls int32

	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		fmt.Fprintf(w, `{"data":[{"id":"a"}],"pagination":{"next":%q}}`,
			server.URL+"/v3/clusters?limit=1000&marker=stuck")
	}))
	t.Cleanup(server.Close)

	client := NewClient(server.URL, "t", server.Client())
	_, err := client.List(context.Background(), "/v3/clusters")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "pagination repeated")
	assert.LessOrEqual(t, atomic.LoadInt32(&calls), int32(3),
		"the repeat must be caught on the first echo, not after the page cap")
}

func TestListSelfReferencingFirstPageFails(t *testing.T) {
	// The degenerate case: the first page names itself as the next page.
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{"data":[{"id":"a"}],"pagination":{"next":%q}}`,
			server.URL+r.URL.String())
	}))
	t.Cleanup(server.Close)

	client := NewClient(server.URL, "t", server.Client())
	_, err := client.List(context.Background(), "/v3/clusters")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "pagination repeated")
}

func TestListRefusesPaginationToAnotherHost(t *testing.T) {
	// The request carries the API token. A next link naming another host would
	// send that credential somewhere the operator never configured.
	client, _ := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"data":[{"id":"a"}],"pagination":{"next":"https://attacker.example.com/v3/clusters"}}`)
	})

	_, err := client.List(context.Background(), "/v3/clusters")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "attacker.example.com")
	assert.NotContains(t, err.Error(), "token-abc")
}

func TestListEmptyCollection(t *testing.T) {
	client, _ := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"type":"collection","data":[],"pagination":{"limit":1000,"total":0}}`)
	})

	records, err := client.List(context.Background(), "/v3/clusterTemplates")
	require.NoError(t, err)
	assert.Empty(t, records)
}

func TestListPageCap(t *testing.T) {
	// Every page hands back a fresh marker, so the repeat guard never fires and
	// only the page cap ends the walk.
	var server *httptest.Server
	var calls int32

	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page := atomic.AddInt32(&calls, 1)
		fmt.Fprintf(w, `{"data":[{"id":"x"}],"pagination":{"next":%q}}`,
			fmt.Sprintf("%s/v3/clusters?limit=1000&marker=%d", server.URL, page))
	}))
	t.Cleanup(server.Close)

	client := NewClient(server.URL, "t", server.Client())
	_, err := client.List(context.Background(), "/v3/clusters")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "did not finish within")
}

func TestErrorClassifiers(t *testing.T) {
	tests := []struct {
		name        string
		status      int
		body        string
		wantNotFnd  bool
		wantForbidn bool
	}{
		{"not found", http.StatusNotFound, `{"type":"error","status":404,"code":"NotFound","message":"clusterTemplates not found"}`, true, false},
		{"forbidden", http.StatusForbidden, `{"type":"error","status":403,"code":"Forbidden","message":"denied"}`, false, true},
		{"unauthorized", http.StatusUnauthorized, `{"type":"error","status":401,"code":"Unauthorized"}`, false, true},
		{"server error", http.StatusInternalServerError, `{"type":"error","status":500}`, false, false},
		{"gateway timeout", http.StatusGatewayTimeout, ``, false, false},
		{"method not allowed", http.StatusMethodNotAllowed, ``, false, false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client, _ := testClient(t, func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(test.status)
				_, _ = w.Write([]byte(test.body))
			})

			_, err := client.List(context.Background(), "/v3/clusterTemplates")
			require.Error(t, err)
			assert.Equal(t, test.wantNotFnd, IsNotFound(err))
			assert.Equal(t, test.wantForbidn, IsForbidden(err))
		})
	}
}

func TestTransportErrorIsNeverClassifiedAsAbsent(t *testing.T) {
	// The failure mode this guards against: a network blip degrading into "the
	// feature is not present", which turns an unreachable Rancher into a clean
	// audit pass.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	address := server.URL
	server.Close()

	client := NewClient(address, "t", &http.Client{Timeout: 2 * time.Second})
	_, err := client.List(context.Background(), "/v3/clusterTemplates")

	require.Error(t, err)
	assert.False(t, IsNotFound(err), "a connection failure is not a 404")
	assert.False(t, IsForbidden(err), "a connection failure is not a refusal")

	var apiErr *APIError
	assert.False(t, errors.As(err, &apiErr), "no API answer was received at all")
}

func TestTimeoutIsNeverClassifiedAsAbsent(t *testing.T) {
	client, _ := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(300 * time.Millisecond)
	})
	client.http = &http.Client{Timeout: 20 * time.Millisecond}

	_, err := client.List(context.Background(), "/v3/clusters")
	require.Error(t, err)
	assert.False(t, IsNotFound(err))
	assert.False(t, IsForbidden(err))
}

func TestTruncatedBodyIsAnError(t *testing.T) {
	// A body that is not a collection must fail rather than decoding to an
	// empty listing, which would report a server with clusters as having none.
	client, _ := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"type":"collection","data":[{"id":`)
	})

	_, err := client.List(context.Background(), "/v3/clusters")
	require.Error(t, err)
}

func TestAPIErrorMessageOmitsAnHTMLBody(t *testing.T) {
	client, _ := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		fmt.Fprint(w, "<html><body>502 Bad Gateway</body></html>")
	})

	_, err := client.List(context.Background(), "/v3/clusters")
	require.Error(t, err)
	assert.NotContains(t, err.Error(), "<html>")
	assert.Contains(t, err.Error(), "502")
}

func TestListCachedFetchesOnce(t *testing.T) {
	var calls int32
	client, _ := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		fmt.Fprint(w, `{"data":[{"id":"a"}]}`)
	})

	for i := 0; i < 5; i++ {
		records, err := client.ListCached(context.Background(), "/v3/projects")
		require.NoError(t, err)
		require.Len(t, records, 1)
	}
	assert.Equal(t, int32(1), atomic.LoadInt32(&calls))
}

func TestListCachedRemembersAFailure(t *testing.T) {
	var calls int32
	client, _ := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusForbidden)
	})

	for i := 0; i < 3; i++ {
		_, err := client.ListCached(context.Background(), "/v3/tokens")
		require.Error(t, err)
		assert.True(t, IsForbidden(err), "a refusal stays a refusal on every read")
	}
	assert.Equal(t, int32(1), atomic.LoadInt32(&calls))
}

func TestGetDecodesASingleObject(t *testing.T) {
	client, _ := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/settings/server-version", r.URL.Path)
		fmt.Fprint(w, `{"id":"server-version","name":"server-version","value":"v2.11.2","default":"dev"}`)
	})

	version, err := ServerVersionForTest(client)
	require.NoError(t, err)
	assert.Equal(t, "v2.11.2", version)
}

// ServerVersionForTest reads the server-version setting through the client, so
// the single-object path is covered without importing the resources package.
func ServerVersionForTest(client *Client) (string, error) {
	var record struct {
		Value   string `json:"value"`
		Default string `json:"default"`
	}
	if err := client.Get(context.Background(), "/v3/settings/server-version", &record); err != nil {
		return "", err
	}
	if record.Value != "" {
		return record.Value, nil
	}
	return record.Default, nil
}

func TestWithLimit(t *testing.T) {
	assert.Equal(t, "/v3/clusters?limit=1000", withLimit("/v3/clusters", 1000))
	assert.Equal(t, "/v3/clusters?sort=name&limit=50", withLimit("/v3/clusters?sort=name", 50))
}

func TestRequestPath(t *testing.T) {
	assert.Equal(t, "/v3/clusters", requestPath("https://rancher.example.com/v3/clusters?limit=1000"))
	assert.Equal(t, "not a url at all", requestPath("not a url at all"))
}

func TestSameServer(t *testing.T) {
	client := NewClient("https://rancher.example.com", "t", http.DefaultClient)

	next, err := client.sameServer("https://rancher.example.com/v3/clusters?marker=x")
	require.NoError(t, err)
	assert.Equal(t, "https://rancher.example.com/v3/clusters?marker=x", next)

	// Case differences in the host are the same server.
	next, err = client.sameServer("https://RANCHER.example.com/v3/clusters")
	require.NoError(t, err)
	assert.NotEmpty(t, next)

	// A relative link resolves against the configured server.
	next, err = client.sameServer("/v3/clusters?marker=y")
	require.NoError(t, err)
	assert.Equal(t, "https://rancher.example.com/v3/clusters?marker=y", next)

	_, err = client.sameServer("http://rancher.example.com/v3/clusters")
	assert.Error(t, err, "a downgrade to plaintext is not the same server")

	_, err = client.sameServer("https://elsewhere.example.com/v3/clusters")
	assert.Error(t, err)
}

func TestAbsolute(t *testing.T) {
	client := NewClient("https://rancher.example.com/", "t", http.DefaultClient)
	assert.Equal(t, "https://rancher.example.com/v3/clusters", client.absolute("/v3/clusters"))
	assert.Equal(t, "https://rancher.example.com/v3/clusters", client.absolute("v3/clusters"))
	assert.Equal(t, "https://other.example.com/x", client.absolute("https://other.example.com/x"))
}

func TestRequestCarriesNoTokenInErrors(t *testing.T) {
	client, server := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	_, err := client.List(context.Background(), "/v3/clusters")
	require.Error(t, err)
	assert.NotContains(t, err.Error(), "secret")

	parsed, parseErr := url.Parse(server.URL)
	require.NoError(t, parseErr)
	assert.False(t, strings.Contains(err.Error(), parsed.Host+"?token"))
}
