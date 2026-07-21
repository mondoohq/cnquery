// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"fmt"

	"github.com/cockroachdb/errors"
	betaodataerrors "github.com/microsoftgraph/msgraph-beta-sdk-go/models/odataerrors"
	"github.com/microsoftgraph/msgraph-sdk-go/models/odataerrors"
)

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
