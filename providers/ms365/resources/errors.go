// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"fmt"

	"github.com/cockroachdb/errors"
	betaodataerrors "github.com/microsoftgraph/msgraph-beta-sdk-go/models/odataerrors"
	"github.com/microsoftgraph/msgraph-sdk-go/models/odataerrors"
)

// graphErrorCode returns the Microsoft Graph error code carried by an
// ODataError -- "Request_ResourceNotFound", "Authorization_RequestDenied" and
// so on -- or "" when the error is not an ODataError or carries no code.
// Both the v1 and the beta SDK model the payload separately, so both are
// checked.
func graphErrorCode(err error) string {
	if err == nil {
		return ""
	}

	var betaOdataErr *betaodataerrors.ODataError
	if errors.As(err, &betaOdataErr) && betaOdataErr != nil {
		if payload := betaOdataErr.GetErrorEscaped(); payload != nil {
			if code := payload.GetCode(); code != nil {
				return *code
			}
		}
	}

	var oDataErr *odataerrors.ODataError
	if errors.As(err, &oDataErr) && oDataErr != nil {
		if payload := oDataErr.GetErrorEscaped(); payload != nil {
			if code := payload.GetCode(); code != nil {
				return *code
			}
		}
	}

	return ""
}

// isResourceNotFound reports whether Graph rejected the request because a
// referenced directory object no longer exists. Graph raises this for the whole
// request when an $expand cannot resolve one of its targets, so callers that
// expand a reference need it to tell a dangling reference apart from a real
// failure.
func isResourceNotFound(err error) bool {
	return graphErrorCode(err) == "Request_ResourceNotFound"
}

func transformError(err error) error {
	var betaOdataErr *betaodataerrors.ODataError
	if errors.As(err, &betaOdataErr) {
		statusCode := betaOdataErr.ResponseStatusCode

		errorPayload := betaOdataErr.GetErrorEscaped()
		if errorPayload != nil && errorPayload.GetMessage() != nil {
			return fmt.Errorf("an API error while performing request Code: %d, Message: %s", statusCode, *errorPayload.GetMessage())
		}

		return fmt.Errorf("an API error occurred with HTTP status code %d", statusCode)
	}

	oDataErr, ok := err.(*odataerrors.ODataError)
	if ok && oDataErr != nil {
		if err := oDataErr.GetErrorEscaped(); err != nil {
			code, msg := "", ""
			if c := err.GetCode(); c != nil {
				code = *c
			}
			if m := err.GetMessage(); m != nil {
				msg = *m
			}
			return errors.Newf("error while performing request. Code: %s, Message: %s", code, msg)
		}
	}
	return err
}
