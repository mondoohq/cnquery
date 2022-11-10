package msgraphclient

import (
	"github.com/cockroachdb/errors"
	od "github.com/microsoftgraph/msgraph-beta-sdk-go/models/odataerrors"
)

func TransformODataError(err error) error {
	oDataErr := err.(*od.ODataError)
	if oDataErr != nil {
		if err := oDataErr.GetError(); err != nil {
			return errors.Newf("error while performing request. Code: %s, Message: %s", *err.GetCode(), *err.GetMessage())
		}
	}

	return err
}
