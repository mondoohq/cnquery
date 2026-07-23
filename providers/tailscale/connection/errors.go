// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"errors"
	"regexp"
	"strconv"

	tsclient "github.com/tailscale/tailscale-client-go/v2"
)

// The Tailscale client models API failures as tsclient.APIError, but keeps the
// HTTP status on an unexported field and only exposes tsclient.IsNotFound. The
// status is however rendered into the error string by APIError.Error(), which
// formats as "<message> (<status>)". We recover it from there so callers can
// distinguish an authorization failure from a genuine absence.
var apiStatusRe = regexp.MustCompile(`\((\d{3})\)$`)

// APIStatusCode returns the HTTP status code of a Tailscale API error, or 0
// when err is not a Tailscale API error (a transport failure, a context
// cancellation, a JSON decode error).
func APIStatusCode(err error) int {
	if err == nil {
		return 0
	}

	// Only trust the string form once we know this really is an APIError.
	var apiErr tsclient.APIError
	if !errors.As(err, &apiErr) {
		return 0
	}

	match := apiStatusRe.FindStringSubmatch(apiErr.Error())
	if match == nil {
		return 0
	}

	code, convErr := strconv.Atoi(match[1])
	if convErr != nil {
		return 0
	}
	return code
}

// IsAccessDenied reports whether err is a Tailscale API authorization failure:
// either the credential is invalid (401) or it lacks the scope required for the
// endpoint (403). OAuth clients are scoped per resource, so a tailnet the
// caller can otherwise read may still refuse individual endpoints.
func IsAccessDenied(err error) bool {
	switch APIStatusCode(err) {
	case 401, 403:
		return true
	default:
		return false
	}
}

// IsUnavailable reports whether err indicates the endpoint carries no data for
// this tailnet, either because nothing is configured (404) or because the
// tailnet's plan does not include the feature (403). Callers use it to degrade
// an optional collection to empty rather than failing the whole query.
func IsUnavailable(err error) bool {
	return tsclient.IsNotFound(err) || APIStatusCode(err) == 403
}
