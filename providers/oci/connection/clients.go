// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/oracle/oci-go-sdk/v65/aidocument"
	"github.com/oracle/oci-go-sdk/v65/ailanguage"
	"github.com/oracle/oci-go-sdk/v65/aispeech"
	"github.com/oracle/oci-go-sdk/v65/aivision"
	"github.com/oracle/oci-go-sdk/v65/apigateway"
	"github.com/oracle/oci-go-sdk/v65/audit"
	"github.com/oracle/oci-go-sdk/v65/bastion"
	"github.com/oracle/oci-go-sdk/v65/certificatesmanagement"
	"github.com/oracle/oci-go-sdk/v65/cloudguard"
	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/containerengine"
	"github.com/oracle/oci-go-sdk/v65/containerinstances"
	"github.com/oracle/oci-go-sdk/v65/core"
	"github.com/oracle/oci-go-sdk/v65/database"
	"github.com/oracle/oci-go-sdk/v65/datasafe"
	"github.com/oracle/oci-go-sdk/v65/datascience"
	"github.com/oracle/oci-go-sdk/v65/events"
	"github.com/oracle/oci-go-sdk/v65/filestorage"
	"github.com/oracle/oci-go-sdk/v65/functions"
	"github.com/oracle/oci-go-sdk/v65/generativeai"
	"github.com/oracle/oci-go-sdk/v65/generativeaiagent"
	"github.com/oracle/oci-go-sdk/v65/identity"
	"github.com/oracle/oci-go-sdk/v65/keymanagement"
	"github.com/oracle/oci-go-sdk/v65/loadbalancer"
	"github.com/oracle/oci-go-sdk/v65/logging"
	"github.com/oracle/oci-go-sdk/v65/monitoring"
	"github.com/oracle/oci-go-sdk/v65/networkfirewall"
	"github.com/oracle/oci-go-sdk/v65/objectstorage"
	"github.com/oracle/oci-go-sdk/v65/ons"
	"github.com/oracle/oci-go-sdk/v65/redis"
	"github.com/oracle/oci-go-sdk/v65/vault"
	"github.com/oracle/oci-go-sdk/v65/vulnerabilityscanning"
	"github.com/oracle/oci-go-sdk/v65/waf"
)

func (c *OciConnection) IdentityClient() (identity.IdentityClient, error) {
	return identity.NewIdentityClientWithConfigurationProvider(c.config)
}

func (c *OciConnection) TenantID() string {
	return c.tenancyOcid
}

func (c *OciConnection) Tenant(ctx context.Context) (*identity.Tenancy, error) {
	oClient, err := c.IdentityClient()
	if err != nil {
		return nil, err
	}

	resp, err := oClient.GetTenancy(ctx, identity.GetTenancyRequest{
		TenancyId: &c.tenancyOcid,
	})
	if err != nil {
		return nil, err
	}
	return &resp.Tenancy, nil
}

func (c *OciConnection) GetCompartments(ctx context.Context) ([]identity.Compartment, error) {
	oClient, err := c.IdentityClient()
	if err != nil {
		return nil, err
	}

	compartments := make([]identity.Compartment, 0)

	req := identity.GetCompartmentRequest{
		CompartmentId: &c.tenancyOcid,
	}

	resp, err := oClient.GetCompartment(ctx, req)
	if err != nil {
		return nil, err
	}
	compartments = append(compartments, resp.Compartment)

	var page *string
	for {
		request := identity.ListCompartmentsRequest{
			CompartmentId:          common.String(c.tenancyOcid),
			CompartmentIdInSubtree: common.Bool(true),
			LifecycleState:         identity.CompartmentLifecycleStateActive,
			Page:                   page,
		}

		response, err := oClient.ListCompartments(ctx, request)
		if err != nil {
			return nil, errors.Join(errors.New("failed to list compartments in tenancy: "+c.tenancyOcid), err)
		}

		for i := range response.Items {
			compartments = append(compartments, response.Items[i])
		}

		page = response.OpcNextPage
		if response.OpcNextPage == nil {
			break
		}
	}

	return compartments, nil
}

func (c *OciConnection) GetRegions(ctx context.Context) ([]identity.RegionSubscription, error) {
	oClient, err := c.IdentityClient()
	if err != nil {
		return nil, err
	}

	request := identity.ListRegionSubscriptionsRequest{
		TenancyId: common.String(c.tenancyOcid),
	}

	response, err := oClient.ListRegionSubscriptions(ctx, request)
	if err != nil {
		return nil, err
	}

	regions := make([]identity.RegionSubscription, 0)
	for _, region := range response.Items {
		if region.Status != identity.RegionSubscriptionStatusReady {
			continue
		}
		regions = append(regions, region)
	}

	return regions, nil
}

func (c *OciConnection) ComputeClient(region string) (*core.ComputeClient, error) {
	client, err := core.NewComputeClientWithConfigurationProvider(c.config)
	if err != nil {
		return nil, err
	}
	client.SetRegion(region)
	return &client, nil
}

func (c *OciConnection) IdentityClientWithRegion(region string) (*identity.IdentityClient, error) {
	client, err := identity.NewIdentityClientWithConfigurationProvider(c.config)
	if err != nil {
		return nil, err
	}
	client.SetRegion(region)
	return &client, nil
}

func (c *OciConnection) NetworkClient(region string) (*core.VirtualNetworkClient, error) {
	client, err := core.NewVirtualNetworkClientWithConfigurationProvider(c.config)
	if err != nil {
		return nil, err
	}
	client.SetRegion(region)
	return &client, nil
}

func (c *OciConnection) AuditClient(region string) (*audit.AuditClient, error) {
	client, err := audit.NewAuditClientWithConfigurationProvider(c.config)
	if err != nil {
		return nil, err
	}
	client.SetRegion(region)
	return &client, nil
}

func (c *OciConnection) ObjectStorageClient(region string) (*objectstorage.ObjectStorageClient, error) {
	client, err := objectstorage.NewObjectStorageClientWithConfigurationProvider(c.config)
	if err != nil {
		return nil, err
	}
	client.SetRegion(region)
	return &client, nil
}

func (c *OciConnection) BlockstorageClient(region string) (*core.BlockstorageClient, error) {
	client, err := core.NewBlockstorageClientWithConfigurationProvider(c.config)
	if err != nil {
		return nil, err
	}
	client.SetRegion(region)
	return &client, nil
}

func (c *OciConnection) FileStorageClient(region string) (*filestorage.FileStorageClient, error) {
	client, err := filestorage.NewFileStorageClientWithConfigurationProvider(c.config)
	if err != nil {
		return nil, err
	}
	client.SetRegion(region)
	return &client, nil
}

func (c *OciConnection) LoggingClient(region string) (*logging.LoggingManagementClient, error) {
	client, err := logging.NewLoggingManagementClientWithConfigurationProvider(c.config)
	if err != nil {
		return nil, err
	}
	client.SetRegion(region)
	return &client, nil
}

func (c *OciConnection) KmsVaultClient(region string) (*keymanagement.KmsVaultClient, error) {
	client, err := keymanagement.NewKmsVaultClientWithConfigurationProvider(c.config)
	if err != nil {
		return nil, err
	}
	client.SetRegion(region)
	return &client, nil
}

func (c *OciConnection) KmsManagementClient(endpoint string) (*keymanagement.KmsManagementClient, error) {
	client, err := keymanagement.NewKmsManagementClientWithConfigurationProvider(c.config, endpoint)
	if err != nil {
		return nil, err
	}
	return &client, nil
}

func (c *OciConnection) EventsClient(region string) (*events.EventsClient, error) {
	client, err := events.NewEventsClientWithConfigurationProvider(c.config)
	if err != nil {
		return nil, err
	}
	client.SetRegion(region)
	return &client, nil
}

func (c *OciConnection) NotificationControlPlaneClient(region string) (*ons.NotificationControlPlaneClient, error) {
	client, err := ons.NewNotificationControlPlaneClientWithConfigurationProvider(c.config)
	if err != nil {
		return nil, err
	}
	client.SetRegion(region)
	return &client, nil
}

func (c *OciConnection) CloudGuardClient(region string) (*cloudguard.CloudGuardClient, error) {
	client, err := cloudguard.NewCloudGuardClientWithConfigurationProvider(c.config)
	if err != nil {
		return nil, err
	}
	client.SetRegion(region)
	return &client, nil
}

func (c *OciConnection) NotificationDataPlaneClient(region string) (*ons.NotificationDataPlaneClient, error) {
	client, err := ons.NewNotificationDataPlaneClientWithConfigurationProvider(c.config)
	if err != nil {
		return nil, err
	}
	client.SetRegion(region)
	return &client, nil
}

func (c *OciConnection) BastionClient(region string) (*bastion.BastionClient, error) {
	client, err := bastion.NewBastionClientWithConfigurationProvider(c.config)
	if err != nil {
		return nil, err
	}
	client.SetRegion(region)
	return &client, nil
}

func (c *OciConnection) MonitoringClient(region string) (*monitoring.MonitoringClient, error) {
	client, err := monitoring.NewMonitoringClientWithConfigurationProvider(c.config)
	if err != nil {
		return nil, err
	}
	client.SetRegion(region)
	return &client, nil
}

func (c *OciConnection) VaultsClient(region string) (*vault.VaultsClient, error) {
	client, err := vault.NewVaultsClientWithConfigurationProvider(c.config)
	if err != nil {
		return nil, err
	}
	client.SetRegion(region)
	return &client, nil
}

func (c *OciConnection) LoadBalancerClient(region string) (*loadbalancer.LoadBalancerClient, error) {
	client, err := loadbalancer.NewLoadBalancerClientWithConfigurationProvider(c.config)
	if err != nil {
		return nil, err
	}
	client.SetRegion(region)
	return &client, nil
}

func (c *OciConnection) NetworkFirewallClient(region string) (*networkfirewall.NetworkFirewallClient, error) {
	client, err := networkfirewall.NewNetworkFirewallClientWithConfigurationProvider(c.config)
	if err != nil {
		return nil, err
	}
	client.SetRegion(region)
	return &client, nil
}

func (c *OciConnection) ContainerEngineClient(region string) (*containerengine.ContainerEngineClient, error) {
	client, err := containerengine.NewContainerEngineClientWithConfigurationProvider(c.config)
	if err != nil {
		return nil, err
	}
	client.SetRegion(region)
	return &client, nil
}

func (c *OciConnection) WafClient(region string) (*waf.WafClient, error) {
	client, err := waf.NewWafClientWithConfigurationProvider(c.config)
	if err != nil {
		return nil, err
	}
	client.SetRegion(region)
	return &client, nil
}

func (c *OciConnection) DatabaseClient(region string) (*database.DatabaseClient, error) {
	client, err := database.NewDatabaseClientWithConfigurationProvider(c.config)
	if err != nil {
		return nil, err
	}
	client.SetRegion(region)
	return &client, nil
}

func (c *OciConnection) DataSafeClient(region string) (*datasafe.DataSafeClient, error) {
	client, err := datasafe.NewDataSafeClientWithConfigurationProvider(c.config)
	if err != nil {
		return nil, err
	}
	client.SetRegion(region)
	return &client, nil
}

func (c *OciConnection) FunctionsManagementClient(region string) (*functions.FunctionsManagementClient, error) {
	client, err := functions.NewFunctionsManagementClientWithConfigurationProvider(c.config)
	if err != nil {
		return nil, err
	}
	client.SetRegion(region)
	return &client, nil
}

func (c *OciConnection) ContainerInstanceClient(region string) (*containerinstances.ContainerInstanceClient, error) {
	client, err := containerinstances.NewContainerInstanceClientWithConfigurationProvider(c.config)
	if err != nil {
		return nil, err
	}
	client.SetRegion(region)
	return &client, nil
}

func (c *OciConnection) ApiGatewayClient(region string) (*apigateway.ApiGatewayClient, error) {
	client, err := apigateway.NewApiGatewayClientWithConfigurationProvider(c.config)
	if err != nil {
		return nil, err
	}
	client.SetRegion(region)
	return &client, nil
}

func (c *OciConnection) ApiGatewayGatewayClient(region string) (*apigateway.GatewayClient, error) {
	client, err := apigateway.NewGatewayClientWithConfigurationProvider(c.config)
	if err != nil {
		return nil, err
	}
	client.SetRegion(region)
	return &client, nil
}

func (c *OciConnection) ApiGatewayDeploymentClient(region string) (*apigateway.DeploymentClient, error) {
	client, err := apigateway.NewDeploymentClientWithConfigurationProvider(c.config)
	if err != nil {
		return nil, err
	}
	client.SetRegion(region)
	return &client, nil
}

func (c *OciConnection) CertificatesManagementClient(region string) (*certificatesmanagement.CertificatesManagementClient, error) {
	client, err := certificatesmanagement.NewCertificatesManagementClientWithConfigurationProvider(c.config)
	if err != nil {
		return nil, err
	}
	client.SetRegion(region)
	return &client, nil
}

func (c *OciConnection) RedisClusterClient(region string) (*redis.RedisClusterClient, error) {
	client, err := redis.NewRedisClusterClientWithConfigurationProvider(c.config)
	if err != nil {
		return nil, err
	}
	client.SetRegion(region)
	return &client, nil
}

func (c *OciConnection) VulnerabilityScanningClient(region string) (*vulnerabilityscanning.VulnerabilityScanningClient, error) {
	client, err := vulnerabilityscanning.NewVulnerabilityScanningClientWithConfigurationProvider(c.config)
	if err != nil {
		return nil, err
	}
	client.SetRegion(region)
	return &client, nil
}

func (c *OciConnection) GenerativeAiAgentClient(region string) (*generativeaiagent.GenerativeAiAgentClient, error) {
	client, err := generativeaiagent.NewGenerativeAiAgentClientWithConfigurationProvider(c.config)
	if err != nil {
		return nil, err
	}
	client.SetRegion(region)
	failFastOnUnreachableRegion(&client.BaseClient)
	return &client, nil
}

func (c *OciConnection) DataScienceClient(region string) (*datascience.DataScienceClient, error) {
	client, err := datascience.NewDataScienceClientWithConfigurationProvider(c.config)
	if err != nil {
		return nil, err
	}
	client.SetRegion(region)
	failFastOnUnreachableRegion(&client.BaseClient)
	return &client, nil
}

// failFastOnUnreachableRegion caps the per-request timeout and disables the
// SDK's default retry policy on a region-limited AI client. Several AI services
// publish a wildcard DNS record in regions where they are not deployed, so calls
// there resolve but the connection times out; without this the SDK would retry
// the timeout ~8 times with backoff and hang for minutes. With it, unavailable
// regions fail fast and are skipped (see ociRegionServiceUnavailable).
func failFastOnUnreachableRegion(bc *common.BaseClient) {
	bc.HTTPClient = &http.Client{Timeout: 15 * time.Second}
	noRetry := common.NoRetryPolicy()
	bc.SetCustomClientConfiguration(common.CustomClientConfiguration{RetryPolicy: &noRetry})
}

func (c *OciConnection) GenerativeAiClient(region string) (*generativeai.GenerativeAiClient, error) {
	client, err := generativeai.NewGenerativeAiClientWithConfigurationProvider(c.config)
	if err != nil {
		return nil, err
	}
	client.SetRegion(region)
	failFastOnUnreachableRegion(&client.BaseClient)
	return &client, nil
}

func (c *OciConnection) AILanguageClient(region string) (*ailanguage.AIServiceLanguageClient, error) {
	client, err := ailanguage.NewAIServiceLanguageClientWithConfigurationProvider(c.config)
	if err != nil {
		return nil, err
	}
	client.SetRegion(region)
	failFastOnUnreachableRegion(&client.BaseClient)
	return &client, nil
}

func (c *OciConnection) AIVisionClient(region string) (*aivision.AIServiceVisionClient, error) {
	client, err := aivision.NewAIServiceVisionClientWithConfigurationProvider(c.config)
	if err != nil {
		return nil, err
	}
	client.SetRegion(region)
	failFastOnUnreachableRegion(&client.BaseClient)
	return &client, nil
}

func (c *OciConnection) AISpeechClient(region string) (*aispeech.AIServiceSpeechClient, error) {
	client, err := aispeech.NewAIServiceSpeechClientWithConfigurationProvider(c.config)
	if err != nil {
		return nil, err
	}
	client.SetRegion(region)
	failFastOnUnreachableRegion(&client.BaseClient)
	return &client, nil
}

func (c *OciConnection) AIDocumentClient(region string) (*aidocument.AIServiceDocumentClient, error) {
	client, err := aidocument.NewAIServiceDocumentClientWithConfigurationProvider(c.config)
	if err != nil {
		return nil, err
	}
	client.SetRegion(region)
	failFastOnUnreachableRegion(&client.BaseClient)
	return &client, nil
}
