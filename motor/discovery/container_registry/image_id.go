package container_registry

import "strings"

func MondooContainerImageID(id string) string {
	id = strings.Replace(id, "sha256:", "", -1)
	return "//platformid.api.mondoo.app/runtime/docker/images/" + id
}

func ShortContainerID(id string) string {
	if len(id) > 12 {
		return id[0:12]
	}
	return id
}

func ShortContainerImageID(id string) string {
	id = strings.Replace(id, "sha256:", "", -1)
	if len(id) > 12 {
		return id[0:12]
	}
	return id
}

func MondooContainerID(id string) string {
	return "//platformid.api.mondoo.app/runtime/docker/containers/" + id
}
