package k8s

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/motor/discovery/common"
	"go.mondoo.io/mondoo/motor/platform"
	"go.mondoo.io/mondoo/motor/transports"
	"go.mondoo.io/mondoo/motor/transports/k8s/resources"
	v1 "k8s.io/api/core/v1"
	k8sRuntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/version"
)

type Option func(*ManifestConnector)

func WithNamespace(namespace string) Option {
	return func(connector *ManifestConnector) {
		connector.namespace = namespace
	}
}

func WithManifestFile(filename string) Option {
	return func(connector *ManifestConnector) {
		connector.manifestFile = filename
	}
}

func WithRuntimeObjects(objects []k8sRuntime.Object) Option {
	return func(connector *ManifestConnector) {
		connector.objects = objects
	}
}

func NewManifestConnector(opts ...Option) *ManifestConnector {
	mc := &ManifestConnector{}

	for _, option := range opts {
		option(mc)
	}

	return mc
}

type ManifestConnector struct {
	manifestFile string
	namespace    string
	objects      []k8sRuntime.Object
}

func (mc *ManifestConnector) Identifier() (string, error) {
	_, err := os.Stat(mc.manifestFile)
	if err != nil {
		return "", errors.Wrap(err, "could not determine platform identifier for "+mc.manifestFile)
	}

	absPath, err := filepath.Abs(mc.manifestFile)
	if err != nil {
		return "", errors.Wrap(err, "could not determine platform identifier for "+mc.manifestFile)
	}

	h := sha256.New()
	h.Write([]byte(absPath))
	return hex.EncodeToString(h.Sum(nil)), nil
}

func (mc *ManifestConnector) Name() (string, error) {
	// manifest parent directory name
	clusterName := common.ProjectNameFromPath(mc.manifestFile)
	clusterName = "K8S Manifest " + clusterName
	return clusterName, nil
}

func (mc *ManifestConnector) loadManifestFile(manifestFile string) ([]byte, error) {
	var input io.Reader

	// if content is piped
	if manifestFile == "-" {
		input = os.Stdin
	} else {
		// return all resources from manifest
		filenames := []string{}

		fi, err := os.Stat(manifestFile)
		if err != nil {
			return nil, err
		}

		if fi.IsDir() {
			// NOTE: we are not using filepath.WalkDir since we do not net recursive walking
			files, err := ioutil.ReadDir(manifestFile)
			if err != nil {
				return nil, err
			}
			for i := range files {
				f := files[i]
				if f.IsDir() {
					continue
				}
				filename := path.Join(manifestFile, f.Name())

				// only load yaml files for now
				ext := filepath.Ext(filename)
				if ext == ".yaml" || ext == ".yml" {
					log.Debug().Str("file", filename).Msg("add file to manifest loading")
					filenames = append(filenames, filename)
				} else {
					log.Debug().Str("file", filename).Msg("ignore file")
				}

			}

		} else {
			filenames = append(filenames, manifestFile)
		}

		input, err = resources.MergeManifestFiles(filenames)
		if err != nil {
			return nil, err
		}
	}

	return ioutil.ReadAll(input)
}

func (mc *ManifestConnector) load() ([]k8sRuntime.Object, error) {
	res := []k8sRuntime.Object{}
	if mc.manifestFile != "" {
		log.Debug().Str("file", mc.manifestFile).Msg("load resources from manifest files")
		input, err := mc.loadManifestFile(mc.manifestFile)
		if err != nil {
			return nil, errors.Wrap(err, "could not load manifest")
		}
		objects, err := resources.ResourcesFromManifest(bytes.NewReader(input))
		if err != nil {
			return nil, err
		}
		res = append(res, objects...)
	}

	if len(mc.objects) > 0 {
		res = append(res, mc.objects...)
	}

	return res, nil
}

func (mc *ManifestConnector) Resources(kind string, name string) (*ResourceResult, error) {
	ns := mc.namespace
	allNs := false
	if ns == "" {
		allNs = true
	}

	resourceObjects, err := mc.load()
	if err != nil {
		return nil, errors.Wrap(err, "could not query resource objects")
	}
	log.Debug().Msgf("found %d resource objects", len(resourceObjects))

	// find root nodes
	resList, err := resources.CachedServerResources()
	if err != nil {
		return nil, err
	}

	resTypes, err := resources.ResourceIndex(resList)
	if err != nil {
		return nil, err
	}

	resType, rootResources, err := resources.FilterResource(resTypes, resourceObjects, kind, name)

	return &ResourceResult{
		Name:          name,
		Kind:          kind,
		ResourceType:  resType,
		AllResources:  resourceObjects,
		RootResources: rootResources,
		Namespace:     ns,
		AllNs:         allNs,
	}, err
}

func (ac *ManifestConnector) PlatformInfo() *platform.Platform {
	return &platform.Platform{
		Name:    "kubernetes",
		Title:   "Kubernetes Manifest",
		Kind:    transports.Kind_KIND_CODE,
		Runtime: transports.RUNTIME_KUBERNETES,
	}
}

func (ac *ManifestConnector) ServerVersion() *version.Info {
	return nil
}

func (ac *ManifestConnector) SupportedResourceTypes() (*resources.ApiResourceIndex, error) {
	return resources.NewApiResourceIndex(), nil
}

func (ac *ManifestConnector) Namespaces() (*v1.NamespaceList, error) {
	return nil, errors.New("not implemented")
}

func (ac *ManifestConnector) Pods(namespace v1.Namespace) (*v1.PodList, error) {
	return nil, errors.New("not implemented")
}
