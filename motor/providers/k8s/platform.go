package k8s

import "strings"

func NewPlatformID(uid string) string {
	return "//platformid.api.mondoo.app/runtime/k8s/uid/" + uid
}

func NewPlatformWorkloadId(clusterIdentifier string, workloadType string, namespace string, name string, uid string) string {
	platformIdentifier := clusterIdentifier
	// when mondoo is called with "--namespace xyz" the cluster identifier already contains the namespace
	// when called with --all-namespaces, it is missing, but we need it to identify workloads
	if !strings.Contains(clusterIdentifier, "namespace") {
		platformIdentifier += "/namespace/" + namespace
	}
	platformIdentifier += "/" + workloadType + "/name/" + name + "/uid/" + uid
	return platformIdentifier
}
