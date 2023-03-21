package k8s

import (
	"fmt"
	"strings"
)

func NewPlatformID(uid string) string {
	return "//platformid.api.mondoo.app/runtime/k8s/uid/" + uid
}

func NewPlatformWorkloadId(clusterIdentifier, workloadType, namespace, name, uid string) string {
	if workloadType == "namespace" {
		return NewNamespacePlatformId(clusterIdentifier, name, uid)
	}

	platformIdentifier := clusterIdentifier
	// when mondoo is called with "--namespace xyz" the cluster identifier already contains the namespace
	// when called without the namespace, it is missing, but we need it to identify workloads
	if !strings.Contains(clusterIdentifier, "namespace") && namespace != "" {
		platformIdentifier += "/namespace/" + namespace
	}
	// add plural "s"
	platformIdentifier += "/" + workloadType + "s" + "/name/" + name
	return platformIdentifier
}

func NewNamespacePlatformId(clusterIdentifier, name, uid string) string {
	if clusterIdentifier == "" {
		return fmt.Sprintf("//platformid.api.mondoo.app/runtime/k8s/namespace/%s", name)
	}

	return fmt.Sprintf("%s/namespace/%s/uid/%s", clusterIdentifier, name, uid)
}
