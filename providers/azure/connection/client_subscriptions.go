// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	subscriptions "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armsubscriptions/v2"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
)

type subscriptionsClient struct {
	token         azcore.TokenCredential
	clientOptions policy.ClientOptions
}

func NewSubscriptionsClient(token azcore.TokenCredential, clientOptions policy.ClientOptions) *subscriptionsClient {
	return &subscriptionsClient{
		token:         token,
		clientOptions: clientOptions,
	}
}

func (client *subscriptionsClient) GetSubscriptions(filter SubscriptionsFilter) ([]subscriptions.Subscription, error) {
	subscriptionsC, err := subscriptions.NewClient(client.token, &arm.ClientOptions{
		ClientOptions: client.clientOptions,
	})
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	subs := []subscriptions.Subscription{}
	res := subscriptionsC.NewListPager(&subscriptions.ClientListOptions{})
	for res.More() {
		page, err := res.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, s := range page.Value {
			if !filter.IsFilteredOut(*s.SubscriptionID) {
				subs = append(subs, *s)
			}
		}
	}
	return subs, nil
}

// GetSubscriptionTags reads a single subscription's tags via the Subscriptions
// Get API — the same endpoint resources/subscription.go uses for the
// azure.subscription.tags field, so it needs no additional permission.
func (client *subscriptionsClient) GetSubscriptionTags(subscriptionID string) (map[string]string, error) {
	subscriptionsC, err := subscriptions.NewClient(client.token, &arm.ClientOptions{
		ClientOptions: client.clientOptions,
	})
	if err != nil {
		return nil, err
	}
	resp, err := subscriptionsC.Get(context.Background(), subscriptionID, &subscriptions.ClientGetOptions{})
	if err != nil {
		return nil, err
	}
	return convert.PtrMapStrToStr(resp.Tags), nil
}
