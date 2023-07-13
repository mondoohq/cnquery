package msgraphclient

import (
	"github.com/cockroachdb/errors"
	absauth "github.com/microsoft/kiota-abstractions-go/authentication"
	msgraphsdkgo "github.com/microsoftgraph/msgraph-sdk-go"
)

const DefaultMSGraphScope = "https://graph.microsoft.com/.default"

var DefaultMSGraphScopes = []string{DefaultMSGraphScope}

func NewGraphRequestAdapterWithFn(providerFn func() (absauth.AuthenticationProvider, error)) (*msgraphsdkgo.GraphRequestAdapter, error) {
	auth, err := providerFn()
	if err != nil {
		return nil, errors.Wrap(err, "authentication provider error")
	}

	return msgraphsdkgo.NewGraphRequestAdapter(auth)
}
