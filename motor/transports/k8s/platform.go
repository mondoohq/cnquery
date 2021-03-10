package k8s

import (
	"errors"

	"k8s.io/apimachinery/pkg/api/meta"
)

func (t *Transport) Identifier() (string, error) {
	// we use "kube-system" namespace uid as identifier for the cluster
	result, err := t.Resources("namespaces", "kube-system")
	if err != nil {
		return "", err
	}

	if len(result.RootsResources) != 1 {
		return "", errors.New("could not identify the cluster")
	}

	resource := result.RootsResources[0]

	obj, err := meta.Accessor(resource)
	if err != nil {
		return "", err
	}

	uid := obj.GetUID()
	return "//platformid.api.mondoo.app/runtime/k8s/uid/" + string(uid), nil
}
