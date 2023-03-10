package azure

const (
	// Discovery flags
	DiscoverySubscriptions = "subscriptions"
	DiscoveryInstances     = "instances"
	// TODO: this probably needs some more work on the linking to its OS counterpart side
	DiscoveryInstancesApi      = "instances-api"
	DiscoverySqlServers        = "sql-servers"
	DiscoveryPostgresServers   = "postgres-servers"
	DiscoveryMySqlServers      = "mysql-servers"
	DiscoveryMariaDbServers    = "mariadb-servers"
	DiscoveryStorageAccounts   = "storage-accounts"
	DiscoveryStorageContainers = "storage-containers"
	DiscoveryKeyVaults         = "keyvaults-vaults"
	DiscoverySecurityGroups    = "security-groups"

	// Labels
	SubscriptionLabel = "azure.mondoo.com/subscription"
	RegionLabel       = "mondoo.com/region"
	InstanceLabel     = "mondoo.com/instance"
)
