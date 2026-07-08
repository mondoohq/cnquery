// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	armresources "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources/v4"
	subscriptions "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armsubscriptions/v2"
	"github.com/rs/zerolog/log"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/azure/connection"
	"go.mondoo.com/mql/v13/utils/stringx"
)

const (
	SubscriptionLabel  = "azure.mondoo.com/subscription"
	ResourceGroupLabel = "azure.mondoo.com/resourcegroup"

	LocationLabel = "mondoo.com/location"
	InstanceLabel = "mondoo.com/instance"

	DiscoveryAuto          = "auto"
	DiscoveryAll           = "all"
	DiscoverySubscriptions = "subscriptions"
	DiscoveryInstances     = "instances"
	// TODO: this probably needs some more work on the linking to its OS counterpart side
	DiscoveryInstancesApi            = "instances-api"
	DiscoverySqlServers              = "sql-servers"
	DiscoveryPostgresServers         = "postgres-servers"
	DiscoveryPostgresFlexibleServers = "postgres-flexible-servers"
	DiscoveryMySqlServers            = "mysql-servers"
	DiscoveryMySqlFlexibleServers    = "mysql-flexible-servers"
	DiscoveryAksClusters             = "aks-clusters"
	DiscoveryAppServiceApps          = "app-service-webapps"
	DiscoveryCacheRedis              = "cache-redis-instances"
	DiscoveryBatchAccounts           = "batch-accounts"
	DiscoveryStorageAccounts         = "storage-accounts"
	DiscoveryStorageContainers       = "storage-containers"
	DiscoveryKeyVaults               = "keyvaults-vaults"
	DiscoveryManagedHsms             = "keyvaults-managed-hsms"
	DiscoveryIotHubs                 = "iot-hubs"
	DiscoverySecurityGroups          = "security-groups"
	DiscoveryCosmosDb                = "cosmosdb"
	DiscoveryVirtualNetworks         = "virtual-networks"
	DiscoveryContainerRegistries     = "container-registries"
	DiscoveryRecoveryServicesVaults  = "recovery-services-vaults"
	DiscoverySynapseWorkspaces       = "synapse-workspaces"
	DiscoveryDataFactories           = "data-factories"
	DiscoveryFunctionApps            = "function-apps"
	DiscoveryApplicationGateways     = "application-gateways"
	DiscoveryFirewalls               = "firewalls"
	DiscoveryContainerApps           = "container-apps"
)

// Auto includes all API resources except storage containers (which require
// additional permissions and can be very numerous). Defined in terms of
// AllAPIResources so the two lists don't drift apart.
var Auto = append(
	[]string{DiscoverySubscriptions},
	slices.DeleteFunc(slices.Clone(AllAPIResources), func(s string) bool {
		return s == DiscoveryStorageContainers
	})...,
)

// All includes every discovery target: Auto plus OS-level instance discovery
// and storage containers.
var All = append(
	slices.Clone(Auto),
	// DiscoveryInstances, note: we disable this for now since we dont support policies for this. we support the API version (DiscoveryInstancesApi)
	DiscoveryStorageContainers,
)

var AllAPIResources = []string{
	DiscoveryInstancesApi,
	DiscoverySqlServers,
	DiscoveryPostgresServers,
	DiscoveryPostgresFlexibleServers,
	DiscoveryMySqlServers,
	DiscoveryMySqlFlexibleServers,
	DiscoveryAksClusters,
	DiscoveryAppServiceApps,
	DiscoveryCacheRedis,
	DiscoveryBatchAccounts,
	DiscoveryStorageAccounts,
	DiscoveryStorageContainers,
	DiscoveryKeyVaults,
	DiscoveryManagedHsms,
	DiscoveryIotHubs,
	DiscoverySecurityGroups,
	DiscoveryCosmosDb,
	DiscoveryVirtualNetworks,
	DiscoveryContainerRegistries,
	DiscoveryRecoveryServicesVaults,
	DiscoverySynapseWorkspaces,
	DiscoveryDataFactories,
	DiscoveryFunctionApps,
	DiscoveryApplicationGateways,
	DiscoveryFirewalls,
	DiscoveryContainerApps,
}

// genericDiscoverySpec maps an ARM resource type to the discovery metadata
// needed to build an inventory asset. Resources listed here are discovered via
// a single armresources.Client.NewListPager call per subscription instead of
// individual service-specific API calls.
//
// When multiple specs share the same `armType` (e.g. function apps and web
// apps both live under "Microsoft.Web/sites"), `matchKind` distinguishes
// between them based on the resource's `kind` value returned by ARM.
type genericDiscoverySpec struct {
	armType                string                 // ARM resource type, e.g. "Microsoft.Sql/servers"
	discoveryTarget        string                 // discovery constant, e.g. DiscoverySqlServers
	service                string                 // service label for azureObject
	objectType             string                 // objectType label for azureObject
	includeObjectTypeInUrl bool                   // passed to mqlObjectToAsset
	matchKind              func(kind string) bool // optional: only match resources whose kind matches
}

// isFunctionAppKind reports whether an ARM "Microsoft.Web/sites" resource is
// a Function App. Function app kinds are "functionapp", "functionapp,linux",
// "functionapp,linux,container", "functionapp,workflowapp", etc.
func isFunctionAppKind(kind string) bool {
	return strings.Contains(strings.ToLower(kind), "functionapp")
}

// isWebAppKind reports whether an ARM "Microsoft.Web/sites" resource is a
// regular web/API app (i.e., not a function app). Web app kinds include
// "app", "app,linux", "api", etc. Anything containing "functionapp" is
// routed to function-app discovery instead.
func isWebAppKind(kind string) bool {
	return !isFunctionAppKind(kind)
}

var genericDiscoverySpecs = []genericDiscoverySpec{
	{armType: "Microsoft.Sql/servers", discoveryTarget: DiscoverySqlServers, service: "sql", objectType: "server"},
	{armType: "Microsoft.DBforMySQL/servers", discoveryTarget: DiscoveryMySqlServers, service: "mysql", objectType: "server"},
	{armType: "Microsoft.DBforMySQL/flexibleServers", discoveryTarget: DiscoveryMySqlFlexibleServers, service: "mysql", objectType: "flexible-server"},
	{armType: "Microsoft.DBforPostgreSQL/servers", discoveryTarget: DiscoveryPostgresServers, service: "postgresql", objectType: "server"},
	{armType: "Microsoft.DBforPostgreSQL/flexibleServers", discoveryTarget: DiscoveryPostgresFlexibleServers, service: "postgresql", objectType: "flexible-server"},
	{armType: "Microsoft.ContainerService/managedClusters", discoveryTarget: DiscoveryAksClusters, service: "aks", objectType: "cluster"},
	// Microsoft.Web/sites is shared by web apps and function apps; disambiguate by kind.
	{armType: "Microsoft.Web/sites", discoveryTarget: DiscoveryAppServiceApps, service: "app-service", objectType: "app", matchKind: isWebAppKind},
	{armType: "Microsoft.Web/sites", discoveryTarget: DiscoveryFunctionApps, service: "functions", objectType: "app", matchKind: isFunctionAppKind},
	{armType: "Microsoft.Cache/Redis", discoveryTarget: DiscoveryCacheRedis, service: "cache", objectType: "redis"},
	{armType: "Microsoft.Batch/batchAccounts", discoveryTarget: DiscoveryBatchAccounts, service: "batch", objectType: "account"},
	{armType: "Microsoft.Storage/storageAccounts", discoveryTarget: DiscoveryStorageAccounts, service: "storage", objectType: "account", includeObjectTypeInUrl: true},
	{armType: "Microsoft.Network/networkSecurityGroups", discoveryTarget: DiscoverySecurityGroups, service: "network", objectType: "security-group", includeObjectTypeInUrl: true},
	{armType: "Microsoft.Network/applicationGateways", discoveryTarget: DiscoveryApplicationGateways, service: "network", objectType: "application-gateway", includeObjectTypeInUrl: true},
	{armType: "Microsoft.Network/azureFirewalls", discoveryTarget: DiscoveryFirewalls, service: "network", objectType: "firewall", includeObjectTypeInUrl: true},
	{armType: "Microsoft.KeyVault/vaults", discoveryTarget: DiscoveryKeyVaults, service: "keyvault", objectType: "vault"},
	{armType: "Microsoft.KeyVault/managedHSMs", discoveryTarget: DiscoveryManagedHsms, service: "keyvault", objectType: "managed-hsm"},
	{armType: "Microsoft.Devices/IotHubs", discoveryTarget: DiscoveryIotHubs, service: "iot", objectType: "iothub"},
	{armType: "Microsoft.DocumentDB/databaseAccounts", discoveryTarget: DiscoveryCosmosDb, service: "cosmosdb", objectType: "account"},
	{armType: "Microsoft.Network/virtualNetworks", discoveryTarget: DiscoveryVirtualNetworks, service: "network", objectType: "virtual-network", includeObjectTypeInUrl: true},
	{armType: "Microsoft.ContainerRegistry/registries", discoveryTarget: DiscoveryContainerRegistries, service: "containerregistry", objectType: "registry"},
	{armType: "Microsoft.RecoveryServices/vaults", discoveryTarget: DiscoveryRecoveryServicesVaults, service: "recoveryservices", objectType: "vault"},
	{armType: "Microsoft.Synapse/workspaces", discoveryTarget: DiscoverySynapseWorkspaces, service: "synapse", objectType: "workspace"},
	{armType: "Microsoft.DataFactory/factories", discoveryTarget: DiscoveryDataFactories, service: "datafactory", objectType: "factory"},
	{armType: "Microsoft.App/containerApps", discoveryTarget: DiscoveryContainerApps, service: "containerapps", objectType: "app"},
}

type azureObject struct {
	subscription string
	tenant       *string
	id           string
	location     string
	service      string
	objectType   string
}

type azureObjectPlatformInfo struct {
	title    string
	platform string
}

type mqlObject struct {
	name        string
	labels      map[string]string
	azureObject azureObject
}

type subWithConfig struct {
	sub  subscriptions.Subscription
	conf *inventory.Config
}

func MondooAzureInstanceID(instanceID string) string {
	return "//platformid.api.mondoo.app/runtime/azure" + strings.ToLower(instanceID)
}

func getDiscoveryTargets(config *inventory.Config) []string {
	targets := config.Discover.Targets
	if len(targets) == 0 {
		return Auto
	}
	if stringx.ContainsAnyOf(targets, DiscoveryAll) {
		// return all discovery targets
		return All
	}
	if stringx.ContainsAnyOf(targets, DiscoveryAuto) {
		// remove the auto keyword (DeleteFunc handles every occurrence; mutating
		// the slice inside a range loop would skip elements after a deletion)
		targets = slices.DeleteFunc(targets, func(s string) bool { return s == DiscoveryAuto })
		// add in the required discovery targets
		return append(targets, Auto...)
	}
	// random assortment of targets
	return targets
}

func Discover(runtime *plugin.Runtime, rootConf *inventory.Config) (*inventory.Inventory, error) {
	conn, ok := runtime.Connection.(*connection.AzureConnection)
	if !ok {
		return nil, errors.New("invalid connection provided, it is not an Azure connection")
	}
	assets := []*inventory.Asset{}
	subsToInclude := rootConf.Options["subscriptions"]
	subsToExclude := rootConf.Options["subscriptions-exclude"]
	filter := connection.SubscriptionsFilter{}
	if len(subsToInclude) > 0 {
		filter.Include = strings.Split(subsToInclude, ",")
	}
	if len(subsToExclude) > 0 {
		filter.Exclude = strings.Split(subsToExclude, ",")
	}
	// note: we always need the subscriptions, either to return them as assets or discover resources inside the subs
	subs, err := discoverSubscriptions(conn, filter)
	if err != nil {
		return nil, err
	}

	subsWithConfigs := make([]subWithConfig, len(subs))
	for i := range subs {
		sub := subs[i]
		subsWithConfigs[i] = subWithConfig{sub: sub, conf: getSubConfig(conn.Conf, sub)}
	}

	targets := getDiscoveryTargets(rootConf)
	log.Debug().
		Int("subscriptions", len(subsWithConfigs)).
		Strs("targets", targets).
		Msg("azure.discovery> starting discovery")

	if stringx.ContainsAnyOf(targets, DiscoverySubscriptions) {
		// we've already discovered those, simply add them as assets
		for _, s := range subsWithConfigs {
			assets = append(assets, subToAsset(s))
		}
	}

	// FIXME: do not discover instances as OSes right now, only discover as API representations.
	if stringx.ContainsAnyOf(targets, DiscoveryInstances) {
		vms, err := discoverInstances(runtime, subsWithConfigs)
		if err != nil {
			return nil, err
		}
		assets = append(assets, vms...)
	}
	if stringx.ContainsAnyOf(targets, DiscoveryInstancesApi) {
		vms, err := discoverInstancesApi(runtime, subsWithConfigs)
		if err != nil {
			return nil, err
		}
		assets = append(assets, vms...)
	}
	// FIXME: bring back the storage containers as as part of FF scanning once we can do parallel scanning
	if stringx.ContainsAnyOf(targets, DiscoveryStorageContainers) {
		containers, err := discoverStorageAccountsContainers(runtime, subsWithConfigs)
		if err != nil {
			return nil, err
		}
		assets = append(assets, containers...)
	}

	// Discover all other resource types via a single ARM generic list call per
	// subscription, replacing 13 individual service-specific API calls.
	genericAssets, err := discoverGeneric(conn, subsWithConfigs, targets)
	if err != nil {
		return nil, err
	}
	assets = append(assets, genericAssets...)

	log.Debug().Int("assets", len(assets)).Msg("azure.discovery> discovery complete")
	return &inventory.Inventory{
		Spec: &inventory.InventorySpec{
			Assets: assets,
		},
	}, nil
}

func discoverInstancesApi(runtime *plugin.Runtime, subsWithConfigs []subWithConfig) ([]*inventory.Asset, error) {
	assets := []*inventory.Asset{}
	for _, subWithConfig := range subsWithConfigs {
		svc, err := NewResource(runtime, "azure.subscription.computeService", map[string]*llx.RawData{
			"subscriptionId": llx.StringDataPtr(subWithConfig.sub.SubscriptionID),
		})
		if err != nil {
			return nil, err
		}
		computeSvc := svc.(*mqlAzureSubscriptionComputeService)
		vms := computeSvc.GetVms()
		if vms.Error != nil {
			return nil, vms.Error
		}
		for _, v := range vms.Data {
			vm := v.(*mqlAzureSubscriptionComputeServiceVm)
			props := vm.GetProperties()
			if props.Error != nil {
				return nil, props.Error
			}
			asset := mqlObjectToAsset(mqlObject{
				name:   vm.Name.Data,
				labels: interfaceMapToStr(vm.Tags.Data),
				azureObject: azureObject{
					id:           vm.Id.Data,
					subscription: *subWithConfig.sub.SubscriptionID,
					tenant:       subWithConfig.sub.TenantID,
					location:     vm.Location.Data,
					service:      "compute",
					objectType:   "vm-api",
				},
			}, subWithConfig.conf, false)
			labels, err := getInstancesLabels(vm)
			if err != nil {
				return nil, err
			}
			enrichWithLabels(asset, labels)
			assets = append(assets, asset)
		}
	}
	return assets, nil
}

func discoverInstances(runtime *plugin.Runtime, subsWithConfigs []subWithConfig) ([]*inventory.Asset, error) {
	assets := []*inventory.Asset{}
	for _, subWithConfig := range subsWithConfigs {
		svc, err := NewResource(runtime, "azure.subscription.computeService", map[string]*llx.RawData{
			"subscriptionId": llx.StringDataPtr(subWithConfig.sub.SubscriptionID),
		})
		if err != nil {
			return nil, err
		}
		computeSvc := svc.(*mqlAzureSubscriptionComputeService)
		vms := computeSvc.GetVms()
		if vms.Error != nil {
			return nil, vms.Error
		}
		for _, v := range vms.Data {
			vm := v.(*mqlAzureSubscriptionComputeServiceVm)
			props := vm.GetProperties()
			if props.Error != nil {
				return nil, props.Error
			}

			ipAddresses := vm.GetPublicIpAddresses()
			if ipAddresses.Error != nil {
				return nil, ipAddresses.Error
			}
			asset := mqlObjectToAsset(mqlObject{
				name:   vm.Name.Data,
				labels: interfaceMapToStr(vm.Tags.Data),
				azureObject: azureObject{
					id:           vm.Id.Data,
					subscription: *subWithConfig.sub.SubscriptionID,
					tenant:       subWithConfig.sub.TenantID,
					location:     vm.Location.Data,
					service:      "compute",
					objectType:   "vm",
				},
			}, subWithConfig.conf, false)
			for _, ip := range ipAddresses.Data {
				ipAddress := ip.(*mqlAzureSubscriptionNetworkServiceIpAddress)
				// TODO: we need to make this work via another provider maybe?
				// this is the OS representation of the VM itself
				asset.Connections = append(asset.Connections, &inventory.Config{
					Type:     "ssh",
					Host:     ipAddress.IpAddress.Data,
					Insecure: true,
				})
			}
			labels, err := getInstancesLabels(vm)
			if err != nil {
				return nil, err
			}
			enrichWithLabels(asset, labels)
			asset.PlatformIds = []string{MondooAzureInstanceID(vm.Id.Data)}
			asset.Platform.Runtime = "azure"
			asset.Platform.Kind = inventory.AssetKindCloudVM
			assets = append(assets, asset)
		}
	}
	return assets, nil
}

// discoverGeneric uses a single ARM resource list call per subscription to
// discover all resource types that only need name/id/location/tags. This
// replaces 13 individual service-specific API calls.
func discoverGeneric(conn *connection.AzureConnection, subsWithConfigs []subWithConfig, targets []string) ([]*inventory.Asset, error) {
	// Filter to only specs whose discovery target is active.
	var activeSpecs []genericDiscoverySpec
	for _, spec := range genericDiscoverySpecs {
		if stringx.ContainsAnyOf(targets, spec.discoveryTarget) {
			activeSpecs = append(activeSpecs, spec)
		}
	}
	if len(activeSpecs) == 0 {
		return nil, nil
	}

	// Build OR filter on the de-duplicated set of ARM types (multiple specs may
	// share an armType when disambiguated by `kind`).
	seenTypes := make(map[string]struct{}, len(activeSpecs))
	clauses := make([]string, 0, len(activeSpecs))
	for _, s := range activeSpecs {
		key := strings.ToLower(s.armType)
		if _, ok := seenTypes[key]; ok {
			continue
		}
		seenTypes[key] = struct{}{}
		clauses = append(clauses, fmt.Sprintf("resourceType eq '%s'", s.armType))
	}
	filter := strings.Join(clauses, " or ")

	// Group specs by lowercase ARM type to allow kind-based dispatch.
	specsByType := make(map[string][]genericDiscoverySpec, len(activeSpecs))
	for _, s := range activeSpecs {
		key := strings.ToLower(s.armType)
		specsByType[key] = append(specsByType[key], s)
	}

	var assets []*inventory.Asset
	for _, swc := range subsWithConfigs {
		subId := *swc.sub.SubscriptionID
		log.Debug().Str("subscription", subId).Str("filter", filter).Msg("azure.discovery> listing resources in subscription")
		client, err := armresources.NewClient(subId, conn.Token(), &arm.ClientOptions{
			ClientOptions: conn.ClientOptions(),
		})
		if err != nil {
			return nil, err
		}

		pager := client.NewListPager(&armresources.ClientListOptions{
			Filter: &filter,
		})
		for pager.More() {
			page, err := pager.NextPage(context.Background())
			if err != nil {
				return nil, err
			}
			for _, resource := range page.Value {
				resType := strings.ToLower(derefStr(resource.Type))
				kind := derefStr(resource.Kind)
				spec, ok := matchSpec(specsByType[resType], kind)
				if !ok {
					continue
				}
				asset := mqlObjectToAsset(mqlObject{
					name:   derefStr(resource.Name),
					labels: convert.PtrMapStrToStr(resource.Tags),
					azureObject: azureObject{
						id:           derefStr(resource.ID),
						subscription: subId,
						tenant:       swc.sub.TenantID,
						location:     derefStr(resource.Location),
						service:      spec.service,
						objectType:   spec.objectType,
					},
				}, swc.conf, spec.includeObjectTypeInUrl)
				if asset != nil {
					assets = append(assets, asset)
				}
			}
		}
	}
	return assets, nil
}

// matchSpec picks the first spec from the candidates list whose matchKind
// accepts the given resource kind. Specs without a matchKind match anything.
func matchSpec(candidates []genericDiscoverySpec, kind string) (genericDiscoverySpec, bool) {
	for _, s := range candidates {
		if s.matchKind == nil || s.matchKind(kind) {
			return s, true
		}
	}
	return genericDiscoverySpec{}, false
}

func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func discoverStorageAccountsContainers(runtime *plugin.Runtime, subsWithConfig []subWithConfig) ([]*inventory.Asset, error) {
	assets := []*inventory.Asset{}
	for _, subWithConfig := range subsWithConfig {
		svc, err := NewResource(runtime, "azure.subscription.storageService", map[string]*llx.RawData{
			"subscriptionId": llx.StringDataPtr(subWithConfig.sub.SubscriptionID),
		})
		if err != nil {
			return nil, err
		}
		storageSvc := svc.(*mqlAzureSubscriptionStorageService)
		accounts := storageSvc.GetAccounts()
		if accounts.Error != nil {
			return nil, accounts.Error
		}
		for _, account := range accounts.Data {
			a := account.(*mqlAzureSubscriptionStorageServiceAccount)
			containers := a.GetContainers()
			if containers.Error != nil {
				return nil, containers.Error
			}
			for _, container := range containers.Data {
				c := container.(*mqlAzureSubscriptionStorageServiceAccountContainer)
				asset := mqlObjectToAsset(mqlObject{
					name:   c.Name.Data,
					labels: map[string]string{},
					azureObject: azureObject{
						id:           c.Id.Data,
						subscription: *subWithConfig.sub.SubscriptionID,
						tenant:       subWithConfig.sub.TenantID,
						location:     a.Location.Data,
						service:      "storage",
						objectType:   "container",
					},
				}, subWithConfig.conf, true)
				assets = append(assets, asset)
			}
		}
	}
	return assets, nil
}

func AzureObjectPlatformId(id string) string {
	// the azure resources have an unique id (even throughout multiple subscriptions), e.g.
	// /subscriptions/f1a2873a-6b27-4097-aa7c-3df51f103e96/resourceGroups/MS365-CIS/providers/Microsoft.Compute/virtualMachines/ms365-windows
	// that should be enough for an unique platform id
	return "//platformid.api.mondoo.app/runtime/azure/v1" + id
}

func enrichWithLabels(a *inventory.Asset, labels map[string]string) {
	if a.Labels == nil {
		a.Labels = map[string]string{}
	}
	for k, v := range labels {
		a.Labels[k] = v
	}
}

func getInstancesLabels(vm *mqlAzureSubscriptionComputeServiceVm) (map[string]string, error) {
	labels := map[string]string{}
	props := vm.GetProperties()
	if props.Error != nil {
		return nil, props.Error
	}

	propsDict, ok := props.Data.(map[string]any)
	if !ok {
		propsDict = map[string]any{}
	}
	if osProfile, ok := propsDict["osProfile"]; ok {
		if osProfileDict, ok := osProfile.(map[string]any); ok {
			if computerName, ok := osProfileDict["computerName"]; ok {
				if name, ok := computerName.(string); ok {
					labels["azure.mondoo.com/computername"] = name
				}
			}
		}
	}
	if storageProfile, ok := propsDict["storageProfile"]; ok {
		if storageProfile, ok := storageProfile.(map[string]any); ok {
			if osDisk, ok := storageProfile["osDisk"]; ok {
				if osDisk, ok := osDisk.(map[string]any); ok {
					if osType, ok := osDisk["osType"]; ok {
						if t, ok := osType.(string); ok {
							labels["azure.mondoo.com/ostype"] = t
						}
					}
				}
			}
		}
	}
	if vmId, ok := propsDict["vmId"]; ok {
		if id, ok := vmId.(string); ok {
			labels["mondoo.com/instance"] = id
		}
	}

	res, err := ParseResourceID(vm.Id.Data)
	if err != nil {
		return nil, err
	}
	labels["azure.mondoo.com/resourcegroup"] = res.ResourceGroup

	return labels, nil
}

func discoverSubscriptions(conn *connection.AzureConnection, filter connection.SubscriptionsFilter) ([]subscriptions.Subscription, error) {
	subsClient := connection.NewSubscriptionsClient(conn.Token(), conn.ClientOptions())
	subs, err := subsClient.GetSubscriptions(filter)
	if err != nil {
		return nil, err
	}
	if len(subs) == 0 {
		return nil, errors.New("cannot find an azure subscription with the provided credentials or the provided filters")
	}

	return subs, nil
}

func subToAsset(subWithConfig subWithConfig) *inventory.Asset {
	sub := subWithConfig.sub
	conf := subWithConfig.conf
	copyConf := conf.Clone(inventory.WithoutDiscovery())
	platformId := "//platformid.api.mondoo.app/runtime/azure/subscriptions/" + *sub.SubscriptionID
	tenantId := "unknown"
	if sub.TenantID != nil {
		tenantId = *sub.TenantID
	}
	platform := &inventory.Platform{
		TechnologyUrlSegments: []string{"azure", tenantId, *sub.SubscriptionID, "account"},
	}
	PlatformByName("azure").Apply(platform)
	return &inventory.Asset{
		Id:          platformId,
		Platform:    platform,
		Name:        fmt.Sprintf("Azure subscription %s", *sub.DisplayName),
		Connections: []*inventory.Config{copyConf},
		PlatformIds: []string{platformId},
	}
}

// creates a config with filled in subscription and tenant id, this config can be used by the subscription asset
// or any assets that are discovered within that subscription
func getSubConfig(rootConf *inventory.Config, sub subscriptions.Subscription) *inventory.Config {
	cfg := rootConf.Clone(inventory.WithoutDiscovery())
	if cfg.Options == nil {
		cfg.Options = map[string]string{}
	}
	cfg.Options[connection.OptionSubscriptionID] = *sub.SubscriptionID
	if sub.TenantID != nil {
		cfg.Options[connection.OptionTenantID] = *sub.TenantID
	}
	return cfg
}

func getTitleFamily(azureObject azureObject) (azureObjectPlatformInfo, error) {
	switch azureObject.service {
	case "compute":
		if azureObject.objectType == "vm" {
			return azureObjectPlatformInfo{title: "Azure Compute VM", platform: "azure-compute-vm"}, nil
		}
		if azureObject.objectType == "vm-api" {
			return azureObjectPlatformInfo{title: "Azure Compute VM", platform: "azure-compute-vm-api"}, nil
		}
	case "sql":
		if azureObject.objectType == "server" {
			return azureObjectPlatformInfo{title: "Azure SQL Database Server", platform: "azure-sql-server"}, nil
		}
	case "postgresql":
		if azureObject.objectType == "server" {
			return azureObjectPlatformInfo{title: "Azure PostgreSQL Server", platform: "azure-postgresql-server"}, nil
		}
		if azureObject.objectType == "flexible-server" {
			return azureObjectPlatformInfo{title: "Azure PostgreSQL Flexible Server", platform: "azure-postgresql-flexible-server"}, nil
		}
	case "mysql":
		if azureObject.objectType == "server" {
			return azureObjectPlatformInfo{title: "Azure MySQL Server", platform: "azure-mysql-server"}, nil
		}
		if azureObject.objectType == "flexible-server" {
			return azureObjectPlatformInfo{title: "Azure MySQL Flexible Server", platform: "azure-mysql-flexible-server"}, nil
		}
	case "aks":
		if azureObject.objectType == "cluster" {
			return azureObjectPlatformInfo{title: "Azure AKS Cluster", platform: "azure-aks-cluster"}, nil
		}
	case "app-service":
		if azureObject.objectType == "app" {
			return azureObjectPlatformInfo{title: "Azure App Service App", platform: "azure-app-service-webapp"}, nil
		}
	case "cache":
		if azureObject.objectType == "redis" {
			return azureObjectPlatformInfo{title: "Azure Cache for Redis Instance", platform: "azure-cache-redis-instance"}, nil
		}
	case "batch":
		if azureObject.objectType == "account" {
			return azureObjectPlatformInfo{title: "Azure Batch Account", platform: "azure-batch-account"}, nil
		}
	case "storage":
		if azureObject.objectType == "account" {
			return azureObjectPlatformInfo{title: "Azure Storage Account", platform: "azure-storage-account"}, nil
		}
		if azureObject.objectType == "container" {
			return azureObjectPlatformInfo{title: "Azure Storage Account Container", platform: "azure-storage-container"}, nil
		}
	case "network":
		if azureObject.objectType == "security-group" {
			return azureObjectPlatformInfo{title: "Azure Network Security Group", platform: "azure-network-security-group"}, nil
		}
		if azureObject.objectType == "virtual-network" {
			return azureObjectPlatformInfo{title: "Azure Virtual Network", platform: "azure-virtual-network"}, nil
		}
		if azureObject.objectType == "application-gateway" {
			return azureObjectPlatformInfo{title: "Azure Application Gateway", platform: "azure-application-gateway"}, nil
		}
		if azureObject.objectType == "firewall" {
			return azureObjectPlatformInfo{title: "Azure Firewall", platform: "azure-firewall"}, nil
		}
	case "functions":
		if azureObject.objectType == "app" {
			return azureObjectPlatformInfo{title: "Azure Function App", platform: "azure-function-app"}, nil
		}
	case "containerapps":
		if azureObject.objectType == "app" {
			return azureObjectPlatformInfo{title: "Azure Container App", platform: "azure-container-app"}, nil
		}
	case "keyvault":
		if azureObject.objectType == "vault" {
			return azureObjectPlatformInfo{title: "Azure Key Vault", platform: "azure-keyvault-vault"}, nil
		}
		if azureObject.objectType == "managed-hsm" {
			return azureObjectPlatformInfo{title: "Azure Key Vault Managed HSM", platform: "azure-keyvault-managedhsm"}, nil
		}
	case "iot":
		if azureObject.objectType == "iothub" {
			return azureObjectPlatformInfo{title: "Azure IoT Hub", platform: "azure-iot-iothub"}, nil
		}
	case "cosmosdb":
		if azureObject.objectType == "account" {
			return azureObjectPlatformInfo{title: "Azure Cosmos DB Account", platform: "azure-cosmosdb"}, nil
		}
	case "containerregistry":
		if azureObject.objectType == "registry" {
			return azureObjectPlatformInfo{title: "Azure Container Registry", platform: "azure-container-registry"}, nil
		}
	case "recoveryservices":
		if azureObject.objectType == "vault" {
			return azureObjectPlatformInfo{title: "Azure Recovery Services Vault", platform: "azure-recovery-services-vault"}, nil
		}
	case "synapse":
		if azureObject.objectType == "workspace" {
			return azureObjectPlatformInfo{title: "Azure Synapse Analytics Workspace", platform: "azure-synapse-workspace"}, nil
		}
	case "datafactory":
		if azureObject.objectType == "factory" {
			return azureObjectPlatformInfo{title: "Azure Data Factory", platform: "azure-datafactory"}, nil
		}
	}
	return azureObjectPlatformInfo{}, fmt.Errorf("missing runtime info for azure object service %s type %s", azureObject.service, azureObject.objectType)
}

func mqlObjectToAsset(mqlObject mqlObject, parentConf *inventory.Config, includeObjectTypeInUrl bool) *inventory.Asset {
	if mqlObject.name == "" {
		mqlObject.name = mqlObject.azureObject.id
	}
	info, err := getTitleFamily(mqlObject.azureObject)
	if err != nil {
		return nil
	}
	platformid := AzureObjectPlatformId(mqlObject.azureObject.id)
	cfg := parentConf.Clone(inventory.WithoutDiscovery())
	cfg.PlatformId = platformid

	tenantId := "unknown"
	if mqlObject.azureObject.tenant != nil {
		tenantId = *mqlObject.azureObject.tenant
	}

	assetUrl := []string{
		"azure", tenantId, mqlObject.azureObject.subscription,
		mqlObject.azureObject.service,
	}
	if includeObjectTypeInUrl {
		assetUrl = append(assetUrl, mqlObject.azureObject.objectType)
	}
	platform := &inventory.Platform{
		TechnologyUrlSegments: assetUrl,
	}
	PlatformByName(info.platform).Apply(platform)
	return &inventory.Asset{
		PlatformIds: []string{platformid, mqlObject.azureObject.id},
		Name:        mqlObject.name,
		Platform:    platform,
		State:       inventory.State_STATE_ONLINE,
		Labels:      addInformationalLabels(mqlObject.labels, mqlObject),
		Connections: []*inventory.Config{cfg},
	}
}

func addInformationalLabels(l map[string]string, o mqlObject) map[string]string {
	if l == nil {
		l = make(map[string]string)
	}
	l[LocationLabel] = o.azureObject.location
	l[SubscriptionLabel] = o.azureObject.subscription
	resourceID, err := ParseResourceID(o.azureObject.id)
	if err == nil {
		l[ResourceGroupLabel] = resourceID.ResourceGroup
	}
	return l
}

func interfaceMapToStr(m map[string]any) map[string]string {
	res := make(map[string]string)
	for k, v := range m {
		if str, ok := v.(string); ok {
			res[k] = str
		}
	}
	return res
}
