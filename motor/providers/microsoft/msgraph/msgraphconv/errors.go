package msgraphconv

import (
	"errors"
	"fmt"

	"github.com/microsoftgraph/msgraph-sdk-go/models/odataerrors"
)

func TransformError(err error) error {
	oDataErr, ok := err.(*odataerrors.ODataError)
	if ok && oDataErr != nil {
		if err := oDataErr.GetError(); err != nil {
			return errors.New(fmt.Sprintf("error while performing request. Code: %s, Message: %s", *err.GetCode(), *err.GetMessage()))
		}
	}
	return err
}
