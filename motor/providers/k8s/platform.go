package k8s

import "strings"

func NewPlatformID(uid string) string {
	return "//platformid.api.mondoo.app/runtime/k8s/uid/" + uid
}

func NewPlatformWorkloadId(clusterIdentifier, workloadType, namespace, name string) string {
	platformIdentifier := clusterIdentifier
	// when mondoo is called with "--namespace xyz" the cluster identifier already contains the namespace
	// when called with --all-namespaces, it is missing, but we need it to identify workloads
	if !strings.Contains(clusterIdentifier, "namespace") {
		platformIdentifier += "/namespace/" + namespace
	}
	// add plural "s"
	platformIdentifier += "/" + workloadType + "s" + "/name/" + name
	return platformIdentifier
}
