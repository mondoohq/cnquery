// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package sdk

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// pagedRoundTripper serves two pages for any Okta collection GET: the first
// request (no `after` cursor) returns page one plus a `Link: rel="next"`
// header pointing at the same path with `after=p2`; the follow-up request
// returns page two with no next link. It records how many requests it saw so a
// test can prove the client actually followed the next link rather than
// stopping after page one.
type pagedRoundTripper struct {
	page1     string
	page2     string
	nextQuery string // e.g. "after=p2"
	calls     int
}

func (rt *pagedRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	rt.calls++
	body := rt.page1
	header := http.Header{}
	if strings.Contains(req.URL.RawQuery, rt.nextQuery) {
		body = rt.page2
	} else {
		next := "https://" + req.URL.Host + req.URL.Path + "?" + rt.nextQuery
		header.Set("Link", "<"+next+`>; rel="next"`)
	}
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     header,
		Body:       io.NopCloser(bytes.NewBufferString(body)),
		Request:    req,
	}, nil
}

// fakeClient returns an ApiExtension whose transport is the given round-tripper,
// so pagination is exercised without touching any global http.Client.
func fakeClient(rt http.RoundTripper) *ApiExtension {
	return &ApiExtension{Host: "x.okta.com", Token: "t", HTTPClient: &http.Client{Transport: rt}}
}

// TestListPolicyRulesFollowsPagination guards the ACCESS_POLICY rule fetch
// against the single-page truncation regression: it must follow the
// `Link: rel="next"` header and return rules from every page.
func TestListPolicyRulesFollowsPagination(t *testing.T) {
	t.Parallel()
	rt := &pagedRoundTripper{
		page1:     `[{"id":"rule1"},{"id":"rule2"}]`,
		page2:     `[{"id":"rule3"}]`,
		nextQuery: "after=p2",
	}

	m := fakeClient(rt)
	rules, err := m.ListPolicyRules(context.Background(), "pol1", 200)
	require.NoError(t, err)
	require.Len(t, rules, 3, "expected rules from both pages, not just the first")
	assert.Equal(t, 2, rt.calls, "expected the client to follow the next-page link once")
}

// TestListPoliciesFollowsPagination guards the same behavior for the policy
// listing itself across all policy types.
func TestListPoliciesFollowsPagination(t *testing.T) {
	t.Parallel()
	rt := &pagedRoundTripper{
		page1:     `[{"id":"p1"},{"id":"p2"}]`,
		page2:     `[{"id":"p3"}]`,
		nextQuery: "after=p2",
	}

	m := fakeClient(rt)
	policies, _, err := m.ListPolicies(context.Background(), "ACCESS_POLICY", 200)
	require.NoError(t, err)
	require.Len(t, policies, 3, "expected policies from both pages, not just the first")
	assert.Equal(t, 2, rt.calls)
}

// cyclingRoundTripper always returns a `next` link pointing back at the same
// path, simulating a malformed Okta response that would otherwise loop forever.
type cyclingRoundTripper struct{ calls int }

func (rt *cyclingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	rt.calls++
	header := http.Header{}
	header.Set("Link", "<https://"+req.URL.Host+req.URL.Path+`?after=loop>; rel="next"`)
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     header,
		Body:       io.NopCloser(bytes.NewBufferString(`[{"id":"x"}]`)),
		Request:    req,
	}, nil
}

// TestPaginationStopsOnCyclingLink proves the maxPages guard terminates a
// self-referential `Link: rel="next"` cycle instead of hanging the scan.
func TestPaginationStopsOnCyclingLink(t *testing.T) {
	t.Parallel()
	rt := &cyclingRoundTripper{}
	m := fakeClient(rt)

	_, err := m.ListPolicyRules(context.Background(), "pol1", 200)
	require.NoError(t, err)
	assert.Equal(t, maxPages, rt.calls, "loop should stop at the maxPages bound")
}
