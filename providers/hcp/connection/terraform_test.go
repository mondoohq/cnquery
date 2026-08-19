// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestClient(t *testing.T, handler http.Handler) (*TfeClient, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	client, err := NewTfeClient(srv.URL, "test-token", srv.Client())
	require.NoError(t, err)
	return client, srv
}

func TestNewTfeClientAddress(t *testing.T) {
	tests := []struct {
		name    string
		address string
		want    string
	}{
		{"default", "", DefaultTfeAddress + tfeAPIPath},
		{"bare host", "tfe.example.com", "https://tfe.example.com" + tfeAPIPath},
		{"full url", "https://tfe.example.com", "https://tfe.example.com" + tfeAPIPath},
		{"trailing slash", "https://tfe.example.com/", "https://tfe.example.com" + tfeAPIPath},
		{"already versioned", "https://tfe.example.com/api/v2", "https://tfe.example.com" + tfeAPIPath},
		{"http scheme kept", "http://tfe.internal", "http://tfe.internal" + tfeAPIPath},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewTfeClient(tt.address, "token", nil)
			require.NoError(t, err)
			assert.Equal(t, tt.want, client.BaseURL())
		})
	}
}

func TestNewTfeClientRequiresToken(t *testing.T) {
	// A missing token must fail loudly. Building a client that silently
	// returns nothing would report an empty estate instead of a missing
	// credential, and an audit over "no workspaces" passes vacuously.
	_, err := NewTfeClient(DefaultTfeAddress, "", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "token required")
}

func TestNewTfeClientRejectsAddressWithoutHost(t *testing.T) {
	_, err := NewTfeClient("https://", "token", nil)
	require.Error(t, err)
}

func TestTfeClientSendsBearerToken(t *testing.T) {
	var gotAuth, gotPath string
	client, _ := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotPath = r.URL.Path
		fmt.Fprint(w, `{"data":{"id":"my-org","type":"organizations","attributes":{"name":"my-org"}}}`)
	}))

	rec, err := client.GetOne(context.Background(), "organizations/my-org", nil)
	require.NoError(t, err)
	require.NotNil(t, rec)
	assert.Equal(t, "Bearer test-token", gotAuth)
	assert.Equal(t, "/api/v2/organizations/my-org", gotPath)
	assert.Equal(t, "my-org", rec.ID)
}

func TestTfeClientGetOneNullData(t *testing.T) {
	client, _ := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"data":null}`)
	}))
	rec, err := client.GetOne(context.Background(), "teams/team-1/authentication-token", nil)
	require.NoError(t, err)
	assert.Nil(t, rec)
}

// pagedHandler serves `pages` pages of `perPage` records, advertising the next
// page in the JSON:API pagination meta.
func pagedHandler(pages, perPage int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		page, _ := strconv.Atoi(r.URL.Query().Get("page[number]"))
		if page == 0 {
			page = 1
		}
		records := make([]map[string]any, 0, perPage)
		for i := 0; i < perPage; i++ {
			records = append(records, map[string]any{
				"id":         fmt.Sprintf("ws-%d-%d", page, i),
				"type":       "workspaces",
				"attributes": map[string]any{"name": fmt.Sprintf("ws-%d-%d", page, i)},
			})
		}
		meta := map[string]any{"current-page": page, "total-pages": pages}
		if page < pages {
			meta["next-page"] = page + 1
		} else {
			meta["next-page"] = nil
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": records,
			"meta": map[string]any{"pagination": meta},
		})
	}
}

func TestTfeClientListWalksEveryPage(t *testing.T) {
	var seenPages []string
	client, _ := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenPages = append(seenPages, r.URL.Query().Get("page[number]"))
		assert.Equal(t, strconv.Itoa(tfePageSize), r.URL.Query().Get("page[size]"))
		pagedHandler(3, 2)(w, r)
	}))

	records, err := client.List(context.Background(), "organizations/acme/workspaces", nil)
	require.NoError(t, err)
	assert.Len(t, records, 6)
	assert.Equal(t, []string{"1", "2", "3"}, seenPages)
	assert.Equal(t, "ws-1-0", records[0].ID)
	assert.Equal(t, "ws-3-1", records[5].ID)
}

func TestTfeClientListPreservesCallerQuery(t *testing.T) {
	var gotFilter string
	client, _ := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotFilter = r.URL.Query().Get("filter[workspace][id]")
		pagedHandler(1, 1)(w, r)
	}))

	q := url.Values{}
	q.Set("filter[workspace][id]", "ws-abc")
	records, err := client.List(context.Background(), "team-workspaces", q)
	require.NoError(t, err)
	assert.Len(t, records, 1)
	assert.Equal(t, "ws-abc", gotFilter)

	// The caller's values must not have been mutated by the page parameters.
	assert.Equal(t, url.Values{"filter[workspace][id]": {"ws-abc"}}, q)
}

func TestTfeClientListStopsOnStuckCursor(t *testing.T) {
	// An endpoint that ignores page[number] and keeps pointing at page 1 would
	// otherwise return the same records over and over, multiplying every
	// record up to the page cap.
	requests := 0
	client, _ := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{{"id": "ws-1", "type": "workspaces"}},
			"meta": map[string]any{"pagination": map[string]any{
				"current-page": 1, "next-page": 1, "total-pages": 99,
			}},
		})
	}))

	records, err := client.List(context.Background(), "organizations/acme/workspaces", nil)
	require.NoError(t, err)
	assert.Len(t, records, 1)
	assert.Equal(t, 1, requests)
}

func TestTfeClientListStopsOnBackwardCursor(t *testing.T) {
	requests := 0
	client, _ := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		page, _ := strconv.Atoi(r.URL.Query().Get("page[number]"))
		next := 1
		if page == 1 {
			next = 2
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{{"id": fmt.Sprintf("ws-%d", page), "type": "workspaces"}},
			"meta": map[string]any{"pagination": map[string]any{
				"current-page": page, "next-page": next,
			}},
		})
	}))

	records, err := client.List(context.Background(), "organizations/acme/workspaces", nil)
	require.NoError(t, err)
	assert.Len(t, records, 2)
	assert.Equal(t, 2, requests)
}

func TestTfeClientListStopsOnEmptyPageWithNextPage(t *testing.T) {
	requests := 0
	client, _ := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		page, _ := strconv.Atoi(r.URL.Query().Get("page[number]"))
		data := []map[string]any{}
		if page == 1 {
			data = append(data, map[string]any{"id": "ws-1", "type": "workspaces"})
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": data,
			"meta": map[string]any{"pagination": map[string]any{
				"current-page": page, "next-page": page + 1,
			}},
		})
	}))

	records, err := client.List(context.Background(), "organizations/acme/workspaces", nil)
	require.NoError(t, err)
	assert.Len(t, records, 1)
	assert.Equal(t, 2, requests)
}

func TestTfeClientListEnforcesPageCap(t *testing.T) {
	// An endpoint that always advertises a further page with real records must
	// fail rather than loop forever.
	client, _ := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page, _ := strconv.Atoi(r.URL.Query().Get("page[number]"))
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{{"id": fmt.Sprintf("ws-%d", page), "type": "workspaces"}},
			"meta": map[string]any{"pagination": map[string]any{
				"current-page": page, "next-page": page + 1,
			}},
		})
	}))

	_, err := client.List(context.Background(), "organizations/acme/workspaces", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "pagination limit")
}

func TestTfeClientListMissingPaginationMeta(t *testing.T) {
	// Some endpoints return a plain collection with no pagination meta at all.
	requests := 0
	client, _ := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		fmt.Fprint(w, `{"data":[{"id":"ws-1","type":"workspaces"}]}`)
	}))

	records, err := client.List(context.Background(), "workspaces/ws-x/relationships/remote-state-consumers", nil)
	require.NoError(t, err)
	assert.Len(t, records, 1)
	assert.Equal(t, 1, requests)
}

func TestTfeErrorClassifiers(t *testing.T) {
	tests := []struct {
		status      int
		notFound    bool
		forbidden   bool
		unavailable bool
	}{
		{http.StatusNotFound, true, false, true},
		{http.StatusForbidden, false, true, true},
		{http.StatusUnauthorized, false, true, true},
		{http.StatusInternalServerError, false, false, false},
		{http.StatusTooManyRequests, false, false, false},
		{http.StatusBadGateway, false, false, false},
	}
	for _, tt := range tests {
		t.Run(strconv.Itoa(tt.status), func(t *testing.T) {
			client, _ := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.status)
				fmt.Fprintf(w, `{"errors":[{"status":"%d","title":"boom"}]}`, tt.status)
			}))
			_, err := client.GetOne(context.Background(), "organizations/acme", nil)
			require.Error(t, err)
			assert.Equal(t, tt.notFound, IsTfeNotFound(err), "IsTfeNotFound")
			assert.Equal(t, tt.forbidden, IsTfeForbidden(err), "IsTfeForbidden")
			assert.Equal(t, tt.unavailable, IsTfeUnavailable(err), "IsTfeUnavailable")
			assert.Contains(t, err.Error(), "boom")
		})
	}
}

func TestTfeErrorClassifiersRejectTransportErrors(t *testing.T) {
	// This is the important direction: a network failure must never classify
	// as "not found" or "forbidden". If it did, a blip would degrade a field
	// to null or an empty list and an audit would pass on data nobody read.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	client, err := NewTfeClient(srv.URL, "token", srv.Client())
	require.NoError(t, err)
	srv.Close()

	_, err = client.GetOne(context.Background(), "organizations/acme", nil)
	require.Error(t, err)
	assert.False(t, IsTfeNotFound(err), "transport error must not read as not-found")
	assert.False(t, IsTfeForbidden(err), "transport error must not read as forbidden")
	assert.False(t, IsTfeUnavailable(err), "transport error must not read as unavailable")

	// A cancelled context is a transport error too, not a missing record.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = client.List(ctx, "organizations/acme/workspaces", nil)
	require.Error(t, err)
	assert.False(t, IsTfeUnavailable(err))
}

func TestTfeErrorClassifiersRejectNonAPIErrors(t *testing.T) {
	assert.False(t, IsTfeNotFound(nil))
	assert.False(t, IsTfeForbidden(nil))
	assert.False(t, IsTfeUnavailable(nil))
	assert.False(t, IsTfeUnavailable(fmt.Errorf("connection refused")))
	assert.False(t, IsTfeUnavailable(fmt.Errorf("404 not found")))
}

func TestTfeErrorClassifiersUnwrapWrappedErrors(t *testing.T) {
	wrapped := fmt.Errorf("listing teams: %w", &TfeError{StatusCode: http.StatusNotFound})
	assert.True(t, IsTfeNotFound(wrapped))
}

func TestTfeErrorDetail(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{"title and detail", `{"errors":[{"title":"not found","detail":"no such org"}]}`, "not found: no such org"},
		{"title only", `{"errors":[{"title":"forbidden"}]}`, "forbidden"},
		{"detail only", `{"errors":[{"detail":"quota exceeded"}]}`, "quota exceeded"},
		{"two errors", `{"errors":[{"title":"a"},{"title":"b"}]}`, "a; b"},
		{"not json api", `upstream connect error`, "upstream connect error"},
		{"empty", ``, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tfeErrorDetail([]byte(tt.body)))
		})
	}
}

func TestTfeErrorMessage(t *testing.T) {
	assert.Equal(t, "hcp terraform: API returned status 404",
		(&TfeError{StatusCode: 404}).Error())
	assert.Equal(t, "hcp terraform: API returned status 403: forbidden",
		(&TfeError{StatusCode: 403, Detail: "forbidden"}).Error())
}

func TestTfeRelationshipOne(t *testing.T) {
	tests := []struct {
		name string
		data string
		want *TfeRef
	}{
		{"object", `{"id":"team-1","type":"teams"}`, &TfeRef{ID: "team-1", Type: "teams"}},
		{"null", `null`, nil},
		{"absent", ``, nil},
		{"array", `[{"id":"team-1","type":"teams"}]`, nil},
		{"empty object", `{}`, nil},
		{"garbage", `"nope"`, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rel := TfeRelationship{Data: []byte(tt.data)}
			got := rel.One()
			if tt.want == nil {
				assert.Nil(t, got)
				return
			}
			require.NotNil(t, got)
			assert.Equal(t, *tt.want, *got)
		})
	}
}

func TestTfeRelationshipMany(t *testing.T) {
	tests := []struct {
		name string
		data string
		want []string
	}{
		{"array", `[{"id":"ws-1","type":"workspaces"},{"id":"ws-2","type":"workspaces"}]`, []string{"ws-1", "ws-2"}},
		{"empty array", `[]`, []string{}},
		{"null", `null`, []string{}},
		{"absent", ``, []string{}},
		{"single object", `{"id":"ws-1","type":"workspaces"}`, []string{"ws-1"}},
		{"drops idless entries", `[{"id":"","type":"workspaces"},{"id":"ws-2","type":"workspaces"}]`, []string{"ws-2"}},
		{"garbage", `"nope"`, []string{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rel := TfeRelationship{Data: []byte(tt.data)}
			got := rel.Many()
			require.NotNil(t, got, "Many must never return nil")
			ids := make([]string, 0, len(got))
			for _, ref := range got {
				ids = append(ids, ref.ID)
			}
			assert.Equal(t, tt.want, ids)
		})
	}
}

func TestTfeRecordRelAndDecode(t *testing.T) {
	raw := `{
      "id":"ws-1","type":"workspaces",
      "attributes":{"name":"prod","auto-apply":true},
      "relationships":{"organization":{"data":{"id":"acme","type":"organizations"}}}
    }`
	var rec TfeRecord
	require.NoError(t, json.Unmarshal([]byte(raw), &rec))

	assert.Equal(t, "ws-1", rec.ID)
	require.NotNil(t, rec.Rel("organization").One())
	assert.Equal(t, "acme", rec.Rel("organization").One().ID)
	// A relationship the record does not carry yields the zero value, not a
	// panic on a nil map.
	assert.Nil(t, rec.Rel("agent-pool").One())
	assert.Empty(t, rec.Rel("agent-pool").Many())

	var attrs struct {
		Name      string `json:"name"`
		AutoApply bool   `json:"auto-apply"`
	}
	require.NoError(t, rec.DecodeAttributes(&attrs))
	assert.Equal(t, "prod", attrs.Name)
	assert.True(t, attrs.AutoApply)

	// A record with no attributes object must not fail the decode.
	bare := TfeRecord{ID: "ws-2"}
	assert.Nil(t, bare.Rel("organization").One())
	require.NoError(t, bare.DecodeAttributes(&attrs))

	// A malformed attributes object must fail loudly, naming the record type.
	broken := TfeRecord{ID: "ws-3", Type: "workspaces", Attributes: []byte(`"not-an-object"`)}
	require.Error(t, broken.DecodeAttributes(&attrs))
}
