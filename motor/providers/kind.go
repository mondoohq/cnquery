package providers

func (x Kind) Name() string {
	switch x {
	case Kind_KIND_VIRTUAL_MACHINE_IMAGE:
		return "virtualmachine-image"
	case Kind_KIND_CONTAINER_IMAGE:
		return "container-image"
	case Kind_KIND_CODE:
		return "code"
	case Kind_KIND_PACKAGE:
		return "package"
	case Kind_KIND_VIRTUAL_MACHINE:
		return "virtualmachine"
	case Kind_KIND_CONTAINER:
		return "container"
	case Kind_KIND_PROCESS:
		return "process"
	case Kind_KIND_API:
		return "api"
	case Kind_KIND_BARE_METAL:
		return "baremetal"
	case Kind_KIND_NETWORK:
		return "network"
	case Kind_KIND_K8S_OBJECT:
		return "k8s-object"
	case Kind_KIND_GCP_OBJECT:
		return "gcp-object"
	case Kind_KIND_AWS_OBJECT:
		return "aws-object"
	case Kind_KIND_AZURE_OBJECT:
		return "azure-object"
	case Kind_KIND_UNKNOWN:
		fallthrough
	default:
		return "unknown"
	}
}

func GetKind(kind string) Kind {
	val := Kind_value[kind]
	return Kind(val)
}
