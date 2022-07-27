package k8s

import "strings"

func NewPlatformID(uid string) string {
	return "//platformid.api.mondoo.app/runtime/k8s/uid/" + uid
}

func NewPlatformPodId(clusterIdentifier string, namespace string, name string, uid string) string {
	if strings.Contains(clusterIdentifier, "namespace") {
		return clusterIdentifier + "/pods/name/" + name + "/uid/" + uid
	}
	return clusterIdentifier + "/namespace/" + namespace + "/pods/name/" + name + "/uid/" + uid
}
