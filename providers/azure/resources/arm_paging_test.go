// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testToken() (azcore.AccessToken, error) {
	return azcore.AccessToken{Token: "test-token"}, nil
}

// collectPages is the decode callback the real call sites use: accumulate the
// page's values and hand back the cursor.
func collectPages(into *[]string) func([]byte) (string, error) {
	return func(raw []byte) (string, error) {
		var page struct {
			Value    []string `json:"value"`
			NextLink string   `json:"nextLink"`
		}
		if err := json.Unmarshal(raw, &page); err != nil {
			return "", err
		}
		*into = append(*into, page.Value...)
		return page.NextLink, nil
	}
}

func TestFetchArmPagesFollowsTheCursor(t *testing.T) {
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
		switch r.URL.Path {
		case "/page1":
			fmt.Fprintf(w, `{"value":["a","b"],"nextLink":%q}`, srv.URL+"/page2")
		case "/page2":
			fmt.Fprintf(w, `{"value":["c"],"nextLink":%q}`, srv.URL+"/page3")
		case "/page3":
			fmt.Fprint(w, `{"value":["d"]}`)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	var got []string
	require.NoError(t, fetchArmPages(context.Background(), testToken, srv.URL+"/page1", "things", collectPages(&got)))
	assert.Equal(t, []string{"a", "b", "c", "d"}, got, "every page must reach the caller")
}

func TestFetchArmPagesStopsWithoutACursor(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"value":["only"]}`)
	}))
	defer srv.Close()

	var got []string
	require.NoError(t, fetchArmPages(context.Background(), testToken, srv.URL, "things", collectPages(&got)))
	assert.Equal(t, []string{"only"}, got)
}

// A cursor that points back at a page already fetched would loop forever and
// duplicate every row it returned. The walk has to notice rather than hang the
// scan.
func TestFetchArmPagesRejectsACycle(t *testing.T) {
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// always points back at itself
		fmt.Fprintf(w, `{"value":["x"],"nextLink":%q}`, srv.URL+"/loop")
	}))
	defer srv.Close()

	var got []string
	err := fetchArmPages(context.Background(), testToken, srv.URL+"/loop", "things", collectPages(&got))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cycled")
	assert.Len(t, got, 1, "the cycle must be caught on the second visit, not walked")
}

// A service that keeps handing back a fresh cursor forever must not grow the
// result without bound.
func TestFetchArmPagesIsBounded(t *testing.T) {
	var srv *httptest.Server
	page := 0
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page++
		fmt.Fprintf(w, `{"value":["x"],"nextLink":"%s/p%d"}`, srv.URL, page)
	}))
	defer srv.Close()

	var got []string
	err := fetchArmPages(context.Background(), testToken, srv.URL+"/p0", "things", collectPages(&got))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "stopped after")
	assert.Len(t, got, maxArmPages)
}

// An error body must surface as an error, never be decoded as a zero-length
// page -- that would report an empty collection as fact.
func TestFetchArmPagesSurfacesNon2xx(t *testing.T) {
	for _, status := range []int{http.StatusForbidden, http.StatusTooManyRequests, http.StatusInternalServerError} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(status)
				fmt.Fprint(w, `{"value":[]}`)
			}))
			defer srv.Close()

			var got []string
			err := fetchArmPages(context.Background(), testToken, srv.URL, "things", collectPages(&got))
			require.Error(t, err)
			assert.Contains(t, err.Error(), "failed to fetch things")
			assert.Empty(t, got)
		})
	}
}

// A non-2xx on a later page must fail the whole call rather than quietly
// returning the pages already collected, which would look like a short list.
func TestFetchArmPagesFailsOnALaterPage(t *testing.T) {
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/page1" {
			fmt.Fprintf(w, `{"value":["a"],"nextLink":%q}`, srv.URL+"/page2")
			return
		}
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	var got []string
	err := fetchArmPages(context.Background(), testToken, srv.URL+"/page1", "things", collectPages(&got))
	require.Error(t, err)
}

func TestFetchArmPagesPropagatesDecodeErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `not json`)
	}))
	defer srv.Close()

	var got []string
	require.Error(t, fetchArmPages(context.Background(), testToken, srv.URL, "things", collectPages(&got)))
}

func TestFetchArmPagesPropagatesTokenErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"value":[]}`)
	}))
	defer srv.Close()

	badToken := func() (azcore.AccessToken, error) {
		return azcore.AccessToken{}, assert.AnError
	}
	var got []string
	require.Error(t, fetchArmPages(context.Background(), badToken, srv.URL, "things", collectPages(&got)))
}

func TestFetchArmPagesHonoursContextCancellation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"value":[]}`)
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var got []string
	require.Error(t, fetchArmPages(ctx, testToken, srv.URL, "things", collectPages(&got)))
}
