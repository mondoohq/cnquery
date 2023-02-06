package gcp

const (
	// Discovery flags
	DiscoveryOrganization       = "organization"
	DiscoveryInstances          = "instances"
	DiscoveryProjects           = "projects"
	DiscoveryComputeImages      = "compute-images"
	DiscoveryComputeNetworks    = "compute-networks"
	DiscoveryComputeSubnetworks = "compute-subnetworks"
	DiscoveryComputeFirewalls   = "compute-firewalls"
	DiscoveryGkeClusters        = "gke-clusters"
	DiscoveryStorageBuckets     = "storage-buckets"
	DiscoveryBigQueryDatasets   = "bigquery-datasets"

	// Labels
	ProjectLabel  = "gcp.mondoo.com/project"
	RegionLabel   = "mondoo.com/region"
	InstanceLabel = "mondoo.com/instance"
)
