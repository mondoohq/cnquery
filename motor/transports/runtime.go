package transports

const (
	RUNTIME_AWS              = "aws"     // api
	RUNTIME_AWS_EC2          = "aws ec2" // ec2 instances
	RUNTIME_AWS_SSM_MANAGED  = "aws ssm-managed"
	RUNTIME_AWS_ECR          = "aws ecr"
	RUNTIME_GCP_COMPUTE      = "gcp compute"
	RUNTIME_GCP_GCR          = "gcp gcr"
	RUNTIME_DOCKER_CONTAINER = "docker container"
	RUNTIME_DOCKER_IMAGE     = "docker image"
	RUNTIME_DOCKER_REGISTRY  = "docker registry"
	RUNTIME_KUBERNETES       = "k8s"
	RUNTIME_AZ               = "az" // api
	RUNTIME_AZ_COMPUTE       = "az compute"
	RUNTIME_VSPHERE          = "vsphere"      // api
	RUNTIME_VSPHERE_HOSTS    = "vsphere host" // esxi instances
	RUNTIME_VSPHERE_VM       = "vsphere vm"   // vms running on esxi
	RUNTIME_MICROSOFT_GRAPH  = "ms graph"     // api
)
