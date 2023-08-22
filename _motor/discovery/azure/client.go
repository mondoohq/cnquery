// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

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
