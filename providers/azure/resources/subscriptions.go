// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	subscriptions "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armsubscriptions"
)

type subscriptionsClient struct {
	token azcore.TokenCredential
}

type subscriptionsFilter struct {
	exclude []string
	include []string
}

func NewSubscriptionsClient(token azcore.TokenCredential) *subscriptionsClient {
	return &subscriptionsClient{
		token: token,
	}
}

func (client *subscriptionsClient) GetSubscriptions(filter subscriptionsFilter) ([]subscriptions.Subscription, error) {
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

func skipSub(sub *subscriptions.Subscription, filter subscriptionsFilter) bool {
	// anything explicitly specified in the list of includes means accept only from that list
	if len(filter.include) > 0 {
		for _, s := range filter.include {
			if s == *sub.SubscriptionID {
				return false
			}
		}
		// didn't find it, so it must be skipped
		return true
	}

	// if nothing explicitly meant to be included, then check whether
	// it should be excluded
	if len(filter.exclude) > 0 {
		for _, s := range filter.exclude {
			if s == *sub.SubscriptionID {
				return true
			}
		}

		return false
	}
	return false
}
