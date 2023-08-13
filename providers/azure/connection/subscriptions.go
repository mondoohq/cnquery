// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	subscriptions "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armsubscriptions"
)

type SubscriptionsClient struct {
	token azcore.TokenCredential
}
type SubscriptionsFilter struct {
	Exclude []string
	Include []string
}

func NewSubscriptionsClient(token azcore.TokenCredential) *SubscriptionsClient {
	return &SubscriptionsClient{
		token: token,
	}
}

func (client *SubscriptionsClient) GetSubscription(subscriptionId string) (subscriptions.Subscription, error) {
	subscriptionsC, err := subscriptions.NewClient(client.token, &arm.ClientOptions{})
	if err != nil {
		return subscriptions.Subscription{}, err
	}
	ctx := context.Background()
	resp, err := subscriptionsC.Get(ctx, subscriptionId, &subscriptions.ClientGetOptions{})
	if err != nil {
		return subscriptions.Subscription{}, err
	}
	return resp.Subscription, nil
}

func (client *SubscriptionsClient) GetSubscriptions(filter SubscriptionsFilter) ([]subscriptions.Subscription, error) {
	subscriptionsC, err := subscriptions.NewClient(client.token, &arm.ClientOptions{})

	ctx := context.Background()
	subs := []subscriptions.Subscription{}
	res := subscriptionsC.NewListPager(&subscriptions.ClientListOptions{})
	if err != nil {
		return nil, err
	}
	for res.More() {
		page, err := res.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, s := range page.Value {
			if !skipSub(s, filter) {
				subs = append(subs, *s)
			}
		}
	}
	return subs, nil
}

func skipSub(sub *subscriptions.Subscription, filter SubscriptionsFilter) bool {
	// anything explicitly specified in the list of includes means accept only from that list
	if len(filter.Include) > 0 {
		for _, s := range filter.Include {
			if s == *sub.SubscriptionID {
				return false
			}
		}
		// didn't find it, so it must be skipped
		return true
	}

	// if nothing explicitly meant to be included, then check whether
	// it should be excluded
	if len(filter.Exclude) > 0 {
		for _, s := range filter.Exclude {
			if s == *sub.SubscriptionID {
				return true
			}
		}

		return false
	}
	return false
}
