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

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
	"go.mondoo.io/mondoo/motor/discovery/common"
	"go.mondoo.io/mondoo/motor/platform"
	"go.mondoo.io/mondoo/motor/transports"
	"go.mondoo.io/mondoo/motor/transports/fsutil"
	"go.mondoo.io/mondoo/motor/transports/k8s/resources"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sRuntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/version"
)

type Option func(*manifestTransport)

func WithNamespace(namespace string) Option {
	return func(t *manifestTransport) {
		t.namespace = namespace
	}
}

func WithManifestFile(filename string) Option {
	return func(t *manifestTransport) {
		t.manifestFile = filename
	}
}

func WithRuntimeObjects(objects []k8sRuntime.Object) Option {
	return func(t *manifestTransport) {
		t.objects = objects
	}
}

func newManifestTransport(opts ...Option) Transport {
	t := &manifestTransport{}

	for _, option := range opts {
		option(t)
	}

	return t
}

type manifestTransport struct {
	manifestFile string
	namespace    string
	objects      []runtime.Object
}

func (t *manifestTransport) RunCommand(command string) (*transports.Command, error) {
	return nil, errors.New("k8s does not implement RunCommand")
}

func (t *manifestTransport) FileInfo(path string) (transports.FileInfoDetails, error) {
	return transports.FileInfoDetails{}, errors.New("k8s does not implement FileInfo")
}

func (t *manifestTransport) FS() afero.Fs {
	return &fsutil.NoFs{}
}

func (t *manifestTransport) Close() {}

func (t *manifestTransport) Capabilities() transports.Capabilities {
	return transports.Capabilities{}
}

func (t *manifestTransport) PlatformInfo() *platform.Platform {
	return &platform.Platform{
		Name:    "kubernetes",
		Title:   "Kubernetes Manifest",
		Kind:    transports.Kind_KIND_CODE,
		Runtime: t.Runtime(),
	}
}

func (t *manifestTransport) Kind() transports.Kind {
	return transports.Kind_KIND_API
}

func (t *manifestTransport) Runtime() string {
	return transports.RUNTIME_KUBERNETES_MANIFEST
}

func (t *manifestTransport) PlatformIdDetectors() []transports.PlatformIdDetector {
	return []transports.PlatformIdDetector{
		transports.TransportPlatformIdentifierDetector,
	}
}

func (t *manifestTransport) ServerVersion() *version.Info {
	return nil
}

func (t *manifestTransport) SupportedResourceTypes() (*resources.ApiResourceIndex, error) {
	return resources.NewApiResourceIndex(), nil
}

func (t *manifestTransport) ID() (string, error) {
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
}

func (t *manifestTransport) PlatformIdentifier() (string, error) {
	uid, err := t.ID()
	if err != nil {
		return "", err
	}

	return NewPlatformID(uid), nil
}

func (t *manifestTransport) Identifier() (string, error) {
	return t.PlatformIdentifier()
}

func (t *manifestTransport) Name() (string, error) {
	// manifest parent directory name
	clusterName := common.ProjectNameFromPath(t.manifestFile)
	clusterName = "K8S Manifest " + clusterName
	return clusterName, nil
}

// Namespaces iterates over all file-based manifests and extracts all namespaces used
func (t *manifestTransport) Namespaces() ([]v1.Namespace, error) {
	// iterate over all resources and extract all the namespaces
	resourceObjects, _, err := t.resourceIndex()
	if err != nil {
		return nil, err
	}

	log.Debug().Msgf("found %d resource objects", len(resourceObjects))

	namespaceMap := map[string]struct{}{}
	for i := range resourceObjects {
		res := resourceObjects[i]
		o, err := meta.Accessor(res)
		if err != nil {
			return nil, err
		}

		namespaceMap[o.GetNamespace()] = struct{}{}
	}

	var nss []v1.Namespace

	// NOTE: this only does the minimal required for our current implementation
	// going forward we may need a bit more information
	for k := range namespaceMap {
		nss = append(nss, v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: k,
			},
		})
	}

	return nss, nil
}

func (t *manifestTransport) Pod(namespace string, name string) (*v1.Pod, error) {
	result, err := t.Resources("pods.v1.", "")
	if err != nil {
		return nil, err
	}

	foundPod := &v1.Pod{}
	for i := range result.Resources {
		r := result.Resources[i]

		pod, ok := r.(*v1.Pod)
		if !ok {
			log.Warn().Msg("could not convert k8s resource to pod")
			continue
		}
		if pod.Name == name && pod.Namespace == namespace {
			foundPod = pod
			break
		}
	}

	if foundPod.Name == "" {
		return nil, errors.New("pod not found")
	}
	return foundPod, nil
}

func (t *manifestTransport) Pods(namespace v1.Namespace) ([]v1.Pod, error) {
	// iterate over all resources and extract the pods

	result, err := t.Resources("pods.v1.", "")
	if err != nil {
		return nil, err
	}

	var pods []v1.Pod
	for i := range result.Resources {
		r := result.Resources[i]

		pod, ok := r.(*v1.Pod)
		if !ok {
			log.Warn().Msg("could not convert k8s resource to pod")
			continue
		}
		pods = append(pods, *pod)
	}

	return pods, nil
}

func (t *manifestTransport) resourceIndex() ([]k8sRuntime.Object, *resources.ApiResourceIndex, error) {
	resourceObjects, err := t.load()
	if err != nil {
		return nil, nil, errors.Wrap(err, "could not query resource objects")
	}
	log.Debug().Msgf("found %d resource objects", len(resourceObjects))

	// find root nodes
	resList, err := resources.CachedServerResources()
	if err != nil {
		return nil, nil, err
	}

	resTypes, err := resources.ResourceIndex(resList)
	if err != nil {
		return nil, nil, err
	}
	return resourceObjects, resTypes, nil
}

func (t *manifestTransport) Resources(kind string, name string) (*ResourceResult, error) {
	ns := t.namespace
	allNs := false
	if ns == "" {
		allNs = true
	}

	resourceObjects, resTypes, err := t.resourceIndex()
	if err != nil {
		return nil, err
	}

	resType, err := resTypes.Lookup(kind)
	if err != nil {
		return nil, err
	}

	resources, err := resources.FilterResource(resType, resourceObjects, name)

	return &ResourceResult{
		Name:         name,
		Kind:         kind,
		ResourceType: resType,
		Resources:    resources,
		Namespace:    ns,
		AllNs:        allNs,
	}, err
}

func (t *manifestTransport) load() ([]k8sRuntime.Object, error) {
	res := []k8sRuntime.Object{}
	if t.manifestFile != "" {
		log.Debug().Str("file", t.manifestFile).Msg("load resources from manifest files")
		input, err := t.loadManifestFile(t.manifestFile)
		if err != nil {
			return nil, errors.Wrap(err, "could not load manifest")
		}
		objects, err := resources.ResourcesFromManifest(bytes.NewReader(input))
		if err != nil {
			return nil, err
		}
		res = append(res, objects...)
	}

	if len(t.objects) > 0 {
		res = append(res, t.objects...)
	}

	return res, nil
}

func (t *manifestTransport) loadManifestFile(manifestFile string) ([]byte, error) {
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
