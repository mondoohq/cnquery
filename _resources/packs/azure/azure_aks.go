package azure

import (
	"context"
	"fmt"
	"time"

	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/resources/packs/core"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	clusters "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerservice/armcontainerservice/v2"
)

func (a *mqlAzureSubscriptionAksService) init(args *resources.Args) (*resources.Args, AzureSubscriptionAksService, error) {
	if len(*args) > 0 {
		return args, nil, nil
	}

	at, err := azureTransport(a.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, nil, err
	}

	(*args)["subscriptionId"] = at.SubscriptionID()

	return args, nil, nil
}

func (a *mqlAzureSubscriptionAksService) id() (string, error) {
	subId, err := a.SubscriptionId()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("/subscriptions/%s/aksService", subId), nil
}

func (a *mqlAzureSubscriptionAksServiceCluster) id() (string, error) {
	return a.Id()
}

func (a *mqlAzureSubscriptionAksService) GetClusters() ([]interface{}, error) {
	at, err := azureTransport(a.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	token, err := at.GetTokenCredential()
	if err != nil {
		return nil, err
	}
	client, err := clusters.NewManagedClustersClient(at.SubscriptionID(), token, &arm.ClientOptions{})
	if err != nil {
		return nil, err
	}
	pager := client.NewListPager(&clusters.ManagedClustersClientListOptions{})
	res := []interface{}{}
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, entry := range page.Value {
			storageProfile, err := core.JsonToDict(entry.Properties.StorageProfile)
			if err != nil {
				return nil, err
			}
			workloadAutoScalerProfile, err := core.JsonToDict(entry.Properties.WorkloadAutoScalerProfile)
			if err != nil {
				return nil, err
			}
			securityProfile, err := core.JsonToDict(entry.Properties.SecurityProfile)
			if err != nil {
				return nil, err
			}
			podIdentityProfile, err := core.JsonToDict(entry.Properties.PodIdentityProfile)
			if err != nil {
				return nil, err
			}
			networkProfile, err := core.JsonToDict(entry.Properties.NetworkProfile)
			if err != nil {
				return nil, err
			}
			httpProxyConfig, err := core.JsonToDict(entry.Properties.HTTPProxyConfig)
			if err != nil {
				return nil, err
			}
			addonProfiles := []interface{}{}
			for k, a := range entry.Properties.AddonProfiles {
				dict, err := core.JsonToDict(a)
				if err != nil {
					return nil, err
				}
				m := map[string]interface{}{}
				m[k] = dict
				addonProfiles = append(addonProfiles, m)
			}
			if err != nil {
				return nil, err
			}
			agentPoolProfiles, err := core.JsonToDictSlice(entry.Properties.AgentPoolProfiles)
			if err != nil {
				return nil, err
			}

			var createdAt *time.Time
			if entry.SystemData != nil {
				createdAt = entry.SystemData.CreatedAt
			}

			mqlAksCluster, err := a.MotorRuntime.CreateResource("azure.subscription.aksService.cluster",
				"id", core.ToString(entry.ID),
				"name", core.ToString(entry.Name),
				"location", core.ToString(entry.Location),
				"kubernetesVersion", core.ToString(entry.Properties.KubernetesVersion),
				"provisioningState", core.ToString(entry.Properties.ProvisioningState),
				"createdAt", createdAt,
				"nodeResourceGroup", core.ToString(entry.Properties.NodeResourceGroup),
				"powerState", core.ToString((*string)(entry.Properties.PowerState.Code)),
				"tags", azureTagsToInterface(entry.Tags),
				"rbacEnabled", core.ToBool(entry.Properties.EnableRBAC),
				"dnsPrefix", core.ToString(entry.Properties.DNSPrefix),
				"fqdn", core.ToString(entry.Properties.Fqdn),
				"agentPoolProfiles", agentPoolProfiles,
				"addonProfiles", addonProfiles,
				"httpProxyConfig", httpProxyConfig,
				"networkProfile", networkProfile,
				"podIdentityProfile", podIdentityProfile,
				"securityProfile", securityProfile,
				"storageProfile", storageProfile,
				"workloadAutoScalerProfile", workloadAutoScalerProfile)
			if err != nil {
				return nil, err
			}
			res = append(res, mqlAksCluster)
		}
	}
	return res, nil
}
