// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1
package resources

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	cloudflare "github.com/cloudflare/cloudflare-go/v6"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers/cloudflare/connection"
)

// timeOrNil converts a cloudflare-go v6 time.Time value into MQL time data,
// returning a null value when the timestamp is the zero value. The v6 SDK
// models optional/nullable timestamps as a plain time.Time (a JSON `null`
// decodes to the zero value), whereas the v0 SDK used *time.Time. This helper
// preserves the original null semantics in the MQL schema.
func timeOrNil(t time.Time) *llx.RawData {
	if t.IsZero() {
		return llx.NilData
	}
	tt := t
	return llx.TimeDataPtr(&tt)
}

// cfGetPaged walks a page-numbered Cloudflare list endpoint via the client's
// generic Get, decoding each page's `result` array into T and following
// `result_info.total_pages`. It's used for endpoints whose typed cloudflare-go
// v6 representation is a polymorphic union (or drops fields we expose), where
// decoding the raw payload into our own shape is simpler and preserves the MQL
// schema. uriBase is the path without pagination query params.
func cfGetPaged[T any](conn *connection.CloudflareConnection, uriBase string) ([]T, error) {
	var all []T
	page := 1
	for {
		var env struct {
			Result     []T `json:"result"`
			ResultInfo struct {
				Page       int `json:"page"`
				TotalPages int `json:"total_pages"`
			} `json:"result_info"`
		}
		sep := "?"
		if strings.Contains(uriBase, "?") {
			sep = "&"
		}
		uri := fmt.Sprintf("%s%spage=%d&per_page=100", uriBase, sep, page)
		if err := conn.Cf.Get(context.TODO(), uri, nil, &env); err != nil {
			return nil, err
		}
		all = append(all, env.Result...)
		// Terminate on the local page counter, not the server-echoed
		// ResultInfo.Page: if the API ever returned page:0 with total_pages>0
		// the echoed-page comparison would never satisfy and the loop would
		// spin forever appending empty pages.
		if env.ResultInfo.TotalPages == 0 || page >= env.ResultInfo.TotalPages {
			break
		}
		page++
	}
	return all, nil
}

// degradedList maps an unavailable-resource error (401/403/404, via
// isUnavailable) to an empty list, so a gated add-on or permission-limited
// collection endpoint reads as "nothing here" instead of failing the whole
// query. Other errors propagate unchanged. Callers use it as the error branch
// of a list accessor: `if err != nil { return degradedList(err) }`.
func degradedList(err error) ([]any, error) {
	if isUnavailable(err) {
		return []any{}, nil
	}
	return nil, err
}

// isUnavailable reports whether err is a 401, 403, or 404 from the Cloudflare
// API. These statuses mean the resource isn't available to the calling token —
// an unsupported plan, a missing permission, or an absent resource — which
// callers treat as a null/empty result rather than a hard failure. v6 collapses
// the v0 typed *AuthenticationError/*AuthorizationError/*NotFoundError into a
// single *cloudflare.Error with a StatusCode.
func isUnavailable(err error) bool {
	var apiErr *cloudflare.Error
	if !errors.As(err, &apiErr) {
		return false
	}
	switch apiErr.StatusCode {
	case http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound:
		return true
	}
	return false
}
