// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"github.com/cockroachdb/errors"
	"github.com/microsoftgraph/msgraph-sdk-go/models/odataerrors"
)

func transformError(err error) error {
	oDataErr, ok := err.(*odataerrors.ODataError)
	if ok && oDataErr != nil {
		if err := oDataErr.GetErrorEscaped(); err != nil {
			return errors.Newf("error while performing request. Code: %s, Message: %s", *err.GetCode(), *err.GetMessage())
		}
	}
	return err
}

func isOdataError(err error) (*odataerrors.ODataError, bool) {
	oDataErr, ok := err.(*odataerrors.ODataError)
	return oDataErr, ok
}
