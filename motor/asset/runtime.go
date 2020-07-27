package asset

const (
	RUNTIME_AWS_EC2          = "aws ec2"
	RUNTIME_AWS_SSM_MANAGED  = "aws ssm-managed"
	RUNTIME_AWS_ECR          = "aws ecr"
	RUNTIME_GCP_COMPUTE      = "gcp compute"
	RUNTIME_GCP_GCR          = "gcp gcr"
	RUNTIME_DOCKER_CONTAINER = "docker container"
	RUNTIME_DOCKER_IMAGE     = "docker image"
	RUNTIME_DOCKER_REGISTRY  = "docker registry"
	RUNTIME_KUBERNETES       = "k8s"
	RUNTIME_AZ_COMPUTE       = "az compute"
	RUNTIME_VSPHERE          = "vsphere"      // api
	RUNTIME_VSPHERE_HOSTS    = "vsphere host" // esxi instances
	RUNTIME_VSPHERE_VM       = "vsphere vm"   // vms running on esxi
)

func (x Kind) Name() string {
	switch x {
	case Kind_KIND_VIRTUAL_MACHINE_IMAGE:
		return "virtual machine image"
	case Kind_KIND_CONTAINER_IMAGE:
		return "container image"
	case Kind_KIND_CODE:
		return "code"
	case Kind_KIND_PACKAGE:
		return "package"
	case Kind_KIND_VIRTUAL_MACHINE:
		return "virtual machine"
	case Kind_KIND_CONTAINER:
		return "container"
	case Kind_KIND_PROCESS:
		return "process"
	case Kind_KIND_API:
		return "api"
	case Kind_KIND_BARE_METAL:
		return "bare metal"
	case Kind_KIND_UNKNOWN:
		fallthrough
	default:
		return "unknown"
	}
}
