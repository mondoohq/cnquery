package azure

import (
	"github.com/cockroachdb/errors"
	"go.mondoo.com/cnquery/motor/asset"
	"go.mondoo.com/cnquery/motor/discovery/common"
	"go.mondoo.com/cnquery/motor/providers"
	azure "go.mondoo.com/cnquery/motor/providers/microsoft/azure"
)

type instance struct {
	Id                string
	Name              string
	Tags              map[string]string
	Location          string
	Properties        map[string]interface{}
	PublicIpAddresses []struct {
		IpAddress string
	}
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
			return azureObjectPlatformInfo{title: "Azure SQL Server", platform: "azure-sql-server"}, nil
		}
	case "postgresql":
		if azureObject.objectType == "server" {
			return azureObjectPlatformInfo{title: "Azure PostgreSQL Server", platform: "azure-postgresql-server"}, nil
		}
	case "mysql":
		if azureObject.objectType == "server" {
			return azureObjectPlatformInfo{title: "Azure MySQL Server", platform: "azure-mysql-server"}, nil
		}
	case "mariadb":
		if azureObject.objectType == "server" {
			return azureObjectPlatformInfo{title: "Azure MariaDB Server", platform: "azure-mariadb-server"}, nil
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
	case "keyvault":
		if azureObject.objectType == "vault" {
			return azureObjectPlatformInfo{title: "Azure Key Vault", platform: "azure-keyvault-vault"}, nil
		}
	}
	return azureObjectPlatformInfo{}, errors.Newf("missing runtime info for azure object service %s type %s", azureObject.service, azureObject.objectType)
}

func fetchInstances(m *MqlDiscovery) ([]instance, error) {
	return GetList[instance](m, "return azure.subscription.compute.vms {id name tags location properties publicIpAddresses{ipAddress}}")
}

func convertInstance(vm instance, subscription string, tc *providers.Config, objectType string) (*asset.Asset, error) {
	osProfile, ok := vm.Properties["osProfile"]
	if ok {
		if osProfileDict, ok := osProfile.(map[string]interface{}); ok {
			vm.Tags["azure.mondoo.com/computername"] = osProfileDict["computerName"].(string)
		}
	}
	storageProfile, ok := vm.Properties["storageProfile"]
	if ok {
		if storageProfile, ok := storageProfile.(map[string]interface{}); ok {
			osDisk, ok := storageProfile["osDisk"]
			if ok {
				if osDisk, ok := osDisk.(map[string]interface{}); ok {
					if osType, ok := osDisk["osType"]; ok {
						vm.Tags["azure.mondoo.com/ostype"] = osType.(string)
					}
				}
			}
		}
	}
	vmId, ok := vm.Properties["vmId"]
	if ok {
		if casted, ok := vmId.(string); ok {
			vm.Tags["mondoo.com/instance"] = casted
		}
	}

	res, err := azure.ParseResourceID(vm.Id)
	if err != nil {
		return nil, err
	}
	vm.Tags["azure.mondoo.com/resourcegroup"] = res.ResourceGroup

	return MqlObjectToAsset(
		mqlObject{
			name:   vm.Name,
			labels: vm.Tags,
			azureObject: azureObject{
				id:           vm.Id,
				region:       vm.Location,
				subscription: subscription,
				service:      "compute",
				objectType:   objectType,
			},
		}, tc), nil
}

func computeInstances(m *MqlDiscovery, subscription string, tc *providers.Config, sfn common.QuerySecretFn) ([]*asset.Asset, error) {
	assets := []*asset.Asset{}
	instances, err := fetchInstances(m)
	if err != nil {
		return nil, err
	}
	for _, vm := range instances {
		asset, err := convertInstance(vm, subscription, tc, "vm")
		if err != nil {
			return nil, err
		}
		for _, ip := range vm.PublicIpAddresses {
			asset.Connections = append(asset.Connections, &providers.Config{
				Backend:  providers.ProviderType_SSH,
				Host:     ip.IpAddress,
				Insecure: tc.Insecure,
			})
		}
		// override platform info here, we want to discover this as a proper VM (and not an API representation)
		asset.PlatformIds = []string{MondooAzureInstanceID(vm.Id)}
		asset.Platform.Runtime = providers.RUNTIME_AZ_COMPUTE
		asset.Platform.Kind = providers.Kind_KIND_VIRTUAL_MACHINE
		common.EnrichAssetWithSecrets(asset, sfn)
		assets = append(assets, asset)
	}
	return assets, nil
}

func computeInstancesApi(m *MqlDiscovery, subscription string, tc *providers.Config) ([]*asset.Asset, error) {
	assets := []*asset.Asset{}
	instances, err := fetchInstances(m)
	if err != nil {
		return nil, err
	}
	for _, vm := range instances {
		asset, err := convertInstance(vm, subscription, tc, "vm-api")
		if err != nil {
			return nil, err
		}
		assets = append(assets, asset)
	}
	return assets, nil
}

func computeSqlServers(m *MqlDiscovery, subscription string, tc *providers.Config) ([]*asset.Asset, error) {
	assets := []*asset.Asset{}
	type sqlServer struct {
		Id       string
		Name     string
		Tags     map[string]string
		Location string
	}
	sqlServers, err := GetList[sqlServer](m, "return azure.subscription.sql.servers {id name location tags}")
	if err != nil {
		return nil, err
	}
	for _, sqlServer := range sqlServers {
		assets = append(assets, MqlObjectToAsset(
			mqlObject{
				name:   sqlServer.Name,
				labels: sqlServer.Tags,
				azureObject: azureObject{
					id:           sqlServer.Id,
					service:      "sql",
					objectType:   "server",
					region:       sqlServer.Location,
					subscription: subscription,
				},
			}, tc))
	}
	return assets, nil
}

func computePostgresqlServers(m *MqlDiscovery, subscription string, tc *providers.Config) ([]*asset.Asset, error) {
	assets := []*asset.Asset{}
	type postgreSqlServer struct {
		Id       string
		Name     string
		Tags     map[string]string
		Location string
	}
	postgreSqlServers, err := GetList[postgreSqlServer](m, "return azure.subscription.postgreSql.servers {id name location tags}")
	if err != nil {
		return nil, err
	}
	for _, server := range postgreSqlServers {
		assets = append(assets, MqlObjectToAsset(
			mqlObject{
				name:   server.Name,
				labels: server.Tags,
				azureObject: azureObject{
					id:           server.Id,
					service:      "postgresql",
					objectType:   "server",
					region:       server.Location,
					subscription: subscription,
				},
			}, tc))
	}
	return assets, nil
}

func computeMySqlServers(m *MqlDiscovery, subscription string, tc *providers.Config) ([]*asset.Asset, error) {
	assets := []*asset.Asset{}
	type mySqlServer struct {
		Id       string
		Name     string
		Tags     map[string]string
		Location string
	}
	mySqlServers, err := GetList[mySqlServer](m, "return azure.subscription.mysql.servers {id name location tags}")
	if err != nil {
		return nil, err
	}
	for _, server := range mySqlServers {
		assets = append(assets, MqlObjectToAsset(
			mqlObject{
				name:   server.Name,
				labels: server.Tags,
				azureObject: azureObject{
					id:           server.Id,
					service:      "mysql",
					objectType:   "server",
					region:       server.Location,
					subscription: subscription,
				},
			}, tc))
	}
	return assets, nil
}

func computeMariaDbServers(m *MqlDiscovery, subscription string, tc *providers.Config) ([]*asset.Asset, error) {
	assets := []*asset.Asset{}
	type mariaDbServer struct {
		Id       string
		Name     string
		Tags     map[string]string
		Location string
	}
	mariadbServers, err := GetList[mariaDbServer](m, "return azure.subscription.mariadb.servers {id name location tags}")
	if err != nil {
		return nil, err
	}
	for _, server := range mariadbServers {
		assets = append(assets, MqlObjectToAsset(
			mqlObject{
				name:   server.Name,
				labels: server.Tags,
				azureObject: azureObject{
					id:           server.Id,
					service:      "mariadb",
					objectType:   "server",
					region:       server.Location,
					subscription: subscription,
				},
			}, tc))
	}
	return assets, nil
}

func computeStorageAccounts(m *MqlDiscovery, subscription string, tc *providers.Config) ([]*asset.Asset, error) {
	assets := []*asset.Asset{}
	type storageAccount struct {
		Id       string
		Name     string
		Tags     map[string]string
		Location string
	}
	storageAccs, err := GetList[storageAccount](m, "return azure.subscription.storage.accounts {id name location tags}")
	if err != nil {
		return nil, err
	}
	for _, acc := range storageAccs {
		assets = append(assets, MqlObjectToAsset(
			mqlObject{
				name:   acc.Name,
				labels: acc.Tags,
				azureObject: azureObject{
					id:           acc.Id,
					service:      "storage",
					objectType:   "account",
					region:       acc.Location,
					subscription: subscription,
				},
			}, tc))
	}
	return assets, nil
}

func computeStorageAccountContainers(m *MqlDiscovery, subscription string, tc *providers.Config) ([]*asset.Asset, error) {
	assets := []*asset.Asset{}
	type storageAccountContainer struct {
		Id   string
		Name string
	}

	type storageAccount struct {
		Location   string
		Containers []storageAccountContainer
	}

	accounts, err := GetList[storageAccount](m, "return azure.subscription.storage.accounts { location containers { id name } }")
	if err != nil {
		return nil, err
	}
	for _, a := range accounts {
		for _, container := range a.Containers {
			assets = append(assets, MqlObjectToAsset(
				mqlObject{
					name:   container.Name,
					labels: map[string]string{},
					azureObject: azureObject{
						id:         container.Id,
						service:    "storage",
						objectType: "container",
						// use the same region as the account to which the container belongs
						region:       a.Location,
						subscription: subscription,
					},
				}, tc))
		}
	}
	return assets, nil
}

func computeNetworkSecurityGroups(m *MqlDiscovery, subscription string, tc *providers.Config) ([]*asset.Asset, error) {
	assets := []*asset.Asset{}
	type networkSecurityGroup struct {
		Id       string
		Name     string
		Tags     map[string]string
		Location string
	}
	securityGroups, err := GetList[networkSecurityGroup](m, "return azure.subscription.network.securityGroups { id name location tags }")
	if err != nil {
		return nil, err
	}
	for _, secGroup := range securityGroups {
		assets = append(assets, MqlObjectToAsset(
			mqlObject{
				name:   secGroup.Name,
				labels: secGroup.Tags,
				azureObject: azureObject{
					id:           secGroup.Id,
					service:      "network",
					objectType:   "security-group",
					region:       secGroup.Location,
					subscription: subscription,
				},
			}, tc))
	}
	return assets, nil
}

func computeKeyVaultsVaults(m *MqlDiscovery, subscription string, tc *providers.Config) ([]*asset.Asset, error) {
	assets := []*asset.Asset{}
	type vault struct {
		Id        string
		VaultName string
		Tags      map[string]string
		Location  string
	}
	vaults, err := GetList[vault](m, "return azure.subscription.keyVault.vaults { id vaultName location tags }")
	if err != nil {
		return nil, err
	}
	for _, vault := range vaults {
		assets = append(assets, MqlObjectToAsset(
			mqlObject{
				name:   vault.VaultName,
				labels: vault.Tags,
				azureObject: azureObject{
					id:           vault.Id,
					service:      "keyvault",
					objectType:   "vault",
					region:       vault.Location,
					subscription: subscription,
				},
			}, tc))
	}
	return assets, nil
}
