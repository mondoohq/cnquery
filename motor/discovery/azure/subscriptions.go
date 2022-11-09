package azure

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	subscriptions "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armsubscriptions"
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
	subscriptionsC, err := subscriptions.NewClient(client.AzureClient.Token, &arm.ClientOptions{})
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

func (client *Subscriptions) GetSubscriptions() ([]subscriptions.Subscription, error) {
	subscriptionsC, err := subscriptions.NewClient(client.AzureClient.Token, &arm.ClientOptions{})

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
			subs = append(subs, *s)
		}
	}
	return subs, nil
}
