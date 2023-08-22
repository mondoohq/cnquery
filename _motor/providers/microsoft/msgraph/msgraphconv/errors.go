// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package msgraphconv

import (
	"github.com/cockroachdb/errors"
	"github.com/microsoftgraph/msgraph-sdk-go/models/odataerrors"
)

func TransformError(err error) error {
	oDataErr, ok := err.(*odataerrors.ODataError)
	if ok && oDataErr != nil {
		if err := oDataErr.GetError(); err != nil {
			return errors.Newf("error while performing request. Code: %s, Message: %s", *err.GetCode(), *err.GetMessage())
		}
	}
	return err
}
