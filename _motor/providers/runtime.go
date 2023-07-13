package providers

const (
	RUNTIME_AWS                  = "aws"              // api
	RUNTIME_AWS_EC2              = "aws-ec2-instance" // ec2 instances
	RUNTIME_AWS_SSM_MANAGED      = "aws-ssm-managed"
	RUNTIME_AWS_ECR              = "aws-ecr"
	RUNTIME_AWS_ECS              = "aws-ecs"
	RUNTIME_GCP                  = "gcp" // api
	RUNTIME_GCP_COMPUTE          = "gcp-vm"
	RUNTIME_GCP_GCR              = "gcp-gcr"
	RUNTIME_GOOGLE_WORKSPACE     = "google-workspace"
	RUNTIME_DOCKER_CONTAINER     = "docker-container"
	RUNTIME_DOCKER_IMAGE         = "docker-image"
	RUNTIME_DOCKER_REGISTRY      = "docker-registry"
	RUNTIME_KUBERNETES_CLUSTER   = "k8s-cluster"
	RUNTIME_KUBERNETES_MANIFEST  = "k8s-manifest"
	RUNTIME_KUBERNETES_ADMISSION = "k8s-admission"
	RUNTIME_AZ                   = "azure" // api
	RUNTIME_AZ_COMPUTE           = "azure-vm"
	RUNTIME_VSPHERE              = "vsphere"       // api
	RUNTIME_VSPHERE_HOSTS        = "vsphere-host"  // esxi instances
	RUNTIME_VSPHERE_VM           = "vsphere-vm"    // vms running on esxi
	RUNTIME_MICROSOFT_GRAPH      = "ms-graph"      // api
	RUNTIME_EQUINIX_METAL        = "equinix-metal" // api
	RUNTIME_GITHUB               = "github"        // api
	RUNTIME_AWS_EC2_EBS          = "aws-ec2-ebs"
	RUNTIME_GITLAB               = "gitlab" // api
	RUNTIME_TERRAFORM            = "terraform"
	RUNTIME_OKTA                 = "okta"
	RUNTIME_SLACK                = "slack"
	RUNTIME_OCI                  = "oci"
	RUNTIME_OPCUA                = "opc-ua"
)
