package azure

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseResourceID(t *testing.T) {
	tests := []struct {
		id        string
		expected  *ResourceID
		expectErr bool
	}{
		{
			// no "resourceGroups"
			"/subscriptions/3bbaebfd-abfe-485c-8902-4391ad93a962//myResourceGroup/",
			nil,
			true,
		},
		{
			// no resource group ID
			"/subscriptions/3bbaebfd-abfe-485c-8902-4391ad93a962/resourceGroups//",
			nil,
			true,
		},
		{
			"random",
			nil,
			true,
		},
		{
			"/subscriptions/3bbaebfd-abfe-485c-8902-4391ad93a962",
			&ResourceID{
				SubscriptionID: "3bbaebfd-abfe-485c-8902-4391ad93a962",
				ResourceGroup:  "",
				Provider:       "",
				Path:           map[string]string{},
			},
			false,
		},
		{
			"subscriptions/3bbaebfd-abfe-485c-8902-4391ad93a962",
			nil,
			true,
		},
		{
			"/subscriptions/3bbaebfd-abfe-485c-8902-4391ad93a962/resourceGroups/resGroup1",
			&ResourceID{
				SubscriptionID: "3bbaebfd-abfe-485c-8902-4391ad93a962",
				ResourceGroup:  "resGroup1",
				Provider:       "",
				Path:           map[string]string{},
			},
			false,
		},
		{
			"/subscriptions/3bbaebfd-abfe-485c-8902-4391ad93a962/resourceGroups/resGroup1/providers/Microsoft.Network",
			&ResourceID{
				SubscriptionID: "3bbaebfd-abfe-485c-8902-4391ad93a962",
				ResourceGroup:  "resGroup1",
				Provider:       "Microsoft.Network",
				Path:           map[string]string{},
			},
			false,
		},
		{
			// Missing leading /
			"subscriptions/3bbaebfd-abfe-485c-8902-4391ad93a962/resourceGroups/resGroup1/providers/Microsoft.Network/virtualNetworks/net1/",
			nil,
			true,
		},
		{
			"/subscriptions/3bbaebfd-abfe-485c-8902-4391ad93a962/resourceGroups/resGroup1/providers/Microsoft.Network/virtualNetworks/net1",
			&ResourceID{
				SubscriptionID: "3bbaebfd-abfe-485c-8902-4391ad93a962",
				ResourceGroup:  "resGroup1",
				Provider:       "Microsoft.Network",
				Path: map[string]string{
					"virtualNetworks": "net1",
				},
			},
			false,
		},
		{
			"/subscriptions/3bbaebfd-abfe-485c-8902-4391ad93a962/resourceGroups/resGroup1/providers/Microsoft.Network/virtualNetworks/net1?api-version=2006-01-02-preview",
			&ResourceID{
				SubscriptionID: "3bbaebfd-abfe-485c-8902-4391ad93a962",
				ResourceGroup:  "resGroup1",
				Provider:       "Microsoft.Network",
				Path: map[string]string{
					"virtualNetworks": "net1",
				},
			},
			false,
		},
		{
			"/subscriptions/3bbaebfd-abfe-485c-8902-4391ad93a962/resourceGroups/resGroup1/providers/Microsoft.Network/virtualNetworks/net1/subnets/pubInstance?api-version=2006-01-02-preview",
			&ResourceID{
				SubscriptionID: "3bbaebfd-abfe-485c-8902-4391ad93a962",
				ResourceGroup:  "resGroup1",
				Provider:       "Microsoft.Network",
				Path: map[string]string{
					"virtualNetworks": "net1",
					"subnets":         "pubInstance",
				},
			},
			false,
		},
		{
			"/subscriptions/34ca515c-4629-458e-bf7c-738d77e0d0ea/resourceGroups/resGroup1/providers/Microsoft.ServiceBus/namespaces/testNamespace1/topics/testTopic1/subscriptions/testSubscription1",
			&ResourceID{
				SubscriptionID: "34ca515c-4629-458e-bf7c-738d77e0d0ea",
				ResourceGroup:  "resGroup1",
				Provider:       "Microsoft.ServiceBus",
				Path: map[string]string{
					"namespaces":    "testNamespace1",
					"topics":        "testTopic1",
					"subscriptions": "testSubscription1",
				},
			},
			false,
		},
		{
			"/subscriptions/3bbaebfd-abfe-485c-8902-4391ad93a962/resourceGroups/example-resources/providers/Microsoft.ApiManagement/service/service1/subscriptions/20b49349-a251-4f7e-a10f-22f1b4341b17",
			&ResourceID{
				SubscriptionID: "3bbaebfd-abfe-485c-8902-4391ad93a962",
				ResourceGroup:  "example-resources",
				Provider:       "Microsoft.ApiManagement",
				Path: map[string]string{
					"service":       "service1",
					"subscriptions": "20b49349-a251-4f7e-a10f-22f1b4341b17",
				},
			},
			false,
		},
		{
			// missing resource group
			"/subscriptions/3bbaebfd-abfe-485c-8902-4391ad93a962/providers/Microsoft.ApiManagement/service/service1/subscriptions/20b49349-a251-4f7e-a10f-22f1b4341b17",
			&ResourceID{
				SubscriptionID: "3bbaebfd-abfe-485c-8902-4391ad93a962",
				Provider:       "Microsoft.ApiManagement",
				Path: map[string]string{
					"service":       "service1",
					"subscriptions": "20b49349-a251-4f7e-a10f-22f1b4341b17",
				},
			},
			false,
		},
		{
			"/subscriptions/3bbaebfd-abfe-485c-8902-4391ad93a962/resourceGroups/demo/providers/Microsoft.DBforPostgreSQL/servers/pg11-test",
			&ResourceID{
				SubscriptionID: "3bbaebfd-abfe-485c-8902-4391ad93a962",
				ResourceGroup:  "demo",
				Provider:       "Microsoft.DBforPostgreSQL",
				Path: map[string]string{
					"servers": "pg11-test",
				},
			},
			false,
		},
	}

	for _, test := range tests {
		parsed, err := ParseResourceID(test.id)
		if test.expectErr {
			assert.Error(t, err, test.id)
		} else {
			assert.NoError(t, err, test.id)
		}
		assert.EqualValues(t, test.expected, parsed)
	}
}
