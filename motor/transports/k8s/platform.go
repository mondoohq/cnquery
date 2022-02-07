package k8s

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"

	"github.com/cockroachdb/errors"
	"github.com/gosimple/slug"
	"github.com/rs/zerolog/log"
	"k8s.io/apimachinery/pkg/api/meta"
)

func (t *Transport) Identifier() (string, error) {
	uid := ""

	if t.manifestFile != "" {
		_, err := os.Stat(t.manifestFile)
		if err != nil {
			return "", errors.Wrap(err, "could not determine platform identifier for "+t.manifestFile)
		}

		absPath, err := filepath.Abs(t.manifestFile)
		if err != nil {
			return "", errors.Wrap(err, "could not determine platform identifier for "+t.manifestFile)
		}

		h := sha256.New()
		h.Write([]byte(absPath))
		return hex.EncodeToString(h.Sum(nil)), nil
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
		id := "//platformid.api.mondoo.app/runtime/k8s/uid/" + uid

		if t.opts[OPTION_NAMESPACE] != "" {
			id += "/namespace/" + slug.Make(t.opts[OPTION_NAMESPACE])
		}

		return id, nil
	}
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
