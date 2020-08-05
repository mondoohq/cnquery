package azure

import (
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/azure/auth"
)

func (t *Transport) Authorizer() (autorest.Authorizer, error) {
	return auth.NewAuthorizerFromCLI()
}

func (t *Transport) GraphAuthorizer() (autorest.Authorizer, error) {
	return auth.NewAuthorizerFromCLIWithResource(azure.PublicCloud.ResourceIdentifiers.Graph)
}

func (t *Transport) ParseResourceID(id string) (*ResourceID, error) {
	return ParseResourceID(id)
}
