package azure

import (
	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
)

type AzureClient struct {
	Token azcore.TokenCredential
}

func NewAzureClient(token azcore.TokenCredential) *AzureClient {
	return &AzureClient{
		Token: token,
	}
}
