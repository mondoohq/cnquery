package k8s

import (
	"errors"
	"os"

	"github.com/rs/zerolog/log"

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

type ClusterInfo struct {
	Name string
}

func (t *Transport) ClusterInfo() (ClusterInfo, error) {
	res := ClusterInfo{}

	// right now we use the name of the first node to identify the cluster
	result, err := t.Resources("nodes.v1.", "")
	if err != nil {
		return res, err
	}

	if len(result.RootResources) > 0 {
		node := result.RootResources[0]
		obj, err := meta.Accessor(node)
		if err != nil {
			log.Error().Err(err).Msg("could not access object attributes")
			return res, err
		}
		res.Name = obj.GetName()
	}

	return res, nil
}
