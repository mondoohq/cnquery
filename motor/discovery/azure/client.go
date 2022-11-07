package azure

import (
	"github.com/Azure/go-autorest/autorest"
)

type AzureClient struct {
	Authorizer autorest.Authorizer
}

func NewAzureClient(authorizer autorest.Authorizer) *AzureClient {
	return &AzureClient{
		Authorizer: authorizer,
	}
}
