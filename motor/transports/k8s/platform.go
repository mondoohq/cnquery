package k8s

import (
	"errors"
	"os"

	"go.mondoo.io/mondoo/motor/transports/fsutil"
	"k8s.io/apimachinery/pkg/api/meta"
)

func (t *Transport) Identifier() (string, error) {
	uid := ""

	if t.manifestFile != "" {
		f, err := os.Open(t.manifestFile)
		if err != nil {
			return "", err
		}
		defer f.Close()
		hash, err := fsutil.Sha256(f)
		if err != nil {
			return "", err
		}
		uid = hash
	} else {
		// we use "kube-system" namespace uid as identifier for the cluster
		result, err := t.Resources("namespaces", "kube-system")
		if err != nil {
			return "", err
		}

		if len(result.RootResources) != 1 {
			return "", errors.New("could not identify the k8s cluster")
		}

		resource := result.RootResources[0]

		obj, err := meta.Accessor(resource)
		if err != nil {
			return "", err
		}

		uid = string(obj.GetUID())
	}

	return "//platformid.api.mondoo.app/runtime/k8s/uid/" + uid, nil
}
