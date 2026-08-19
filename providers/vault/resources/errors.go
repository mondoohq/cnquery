// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	"net/http"

	vaultapi "github.com/hashicorp/vault/api"
)

// isFeatureAbsent reports whether the server answered that the endpoint does
// not exist, which is how Community edition responds to an Enterprise-only
// path. Only that answer may be turned into an empty result.
//
// A 403 is deliberately excluded. A token that may not read an endpoint tells
// us nothing about what is behind it, so reporting "none" would turn a missing
// permission into a clean audit pass. It must surface as an error instead.
// The classifier matches on the HTTP status the server returned, never on the
// error text, so a transport failure is not mistaken for a definitive answer.
func isFeatureAbsent(err error) bool {
	if err == nil {
		return false
	}

	var respErr *vaultapi.ResponseError
	if !errors.As(err, &respErr) {
		return false
	}

	switch respErr.StatusCode {
	case http.StatusNotFound, http.StatusMethodNotAllowed:
		return true
	default:
		return false
	}
}
