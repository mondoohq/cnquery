package platform

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
