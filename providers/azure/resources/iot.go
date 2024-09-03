package resources

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/iothub/armiothub"
	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v11/providers/azure/connection"
)

func initAzureSubscriptionIotService(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 0 {
		return args, nil, nil
	}

	conn, ok := runtime.Connection.(*connection.AzureConnection)
	if !ok {
		return nil, nil, fmt.Errorf("invalid connection provided, it is not an Azure connection")
	}
	args["subscriptionId"] = llx.StringData(conn.SubId())

	return args, nil, nil
}

func (a *mqlAzureSubscriptionIotService) hubs() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()

	subscriptionID := a.SubscriptionId.Data

	clientFactory, err := armiothub.NewClientFactory(subscriptionID, token, nil)
	if err != nil {
		return nil, err
	}

	client := clientFactory.NewResourceClient()
	hubsPager := client.NewListBySubscriptionPager(nil)
	var hubs []interface{}

	for hubsPager.More() {
		page, err := hubsPager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, hub := range page.Value {
			hubData, err := convert.JsonToDict(hub)
			if err != nil {
				return nil, err
			}
			hubs = append(hubs, hubData)
		}
	}

	return hubs, nil
}
