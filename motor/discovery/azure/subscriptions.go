package azure

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/resources/mgmt/subscriptions"
)

type Subscriptions struct {
	AzureClient *AzureClient
}

func NewSubscriptions(client *AzureClient) *Subscriptions {
	return &Subscriptions{
		AzureClient: client,
	}
}

func (client *Subscriptions) GetSubscription(subscriptionId string) (subscriptions.Subscription, error) {
	subscriptionsC := subscriptions.NewClient()
	subscriptionsC.Authorizer = client.AzureClient.Authorizer

	ctx := context.Background()
	return subscriptionsC.Get(ctx, subscriptionId)
}

func (client *Subscriptions) GetSubscriptions() ([]subscriptions.Subscription, error) {
	subscriptionsC := subscriptions.NewClient()
	subscriptionsC.Authorizer = client.AzureClient.Authorizer

	ctx := context.Background()
	subs := []subscriptions.Subscription{}
	res, err := subscriptionsC.List(ctx)
	if err != nil {
		return nil, err
	}

	for {
		vals := res.Values()
		if vals == nil {
			break
		}
		subs = append(subs, vals...)
		err := res.NextWithContext(ctx)
		if err != nil {
			break
		}
	}
	return subs, nil
}
