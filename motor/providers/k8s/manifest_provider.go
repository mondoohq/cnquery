package k8s

import (
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
	"go.mondoo.com/cnquery/motor/discovery/common"
	"go.mondoo.com/cnquery/motor/platform"
	"go.mondoo.com/cnquery/motor/providers"
	"go.mondoo.com/cnquery/motor/providers/k8s/resources"
	os_provider "go.mondoo.com/cnquery/motor/providers/os"
	"go.mondoo.com/cnquery/motor/providers/os/fsutil"
	admissionv1 "k8s.io/api/admission/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/version"
)

type Option func(*manifestProvider)

func WithNamespace(namespace string) Option {
	return func(t *manifestProvider) {
		t.namespace = namespace
	}
}

func WithManifestFile(filename string) Option {
	return func(t *manifestProvider) {
		t.manifestFile = filename
	}
}

func newManifestProvider(selectedResourceID string, objectKind string, opts ...Option) (KubernetesProvider, error) {
	t := &manifestProvider{
		objectKind: objectKind,
	}

	for _, option := range opts {
		option(t)
	}

	manifest, err := loadManifestFile(t.manifestFile)
	if err != nil {
		return nil, err
	}
	t.manifestParser, err = newManifestParser(manifest, t.namespace, selectedResourceID)
	if err != nil {
		return nil, err
	}

	t.selectedResourceID = selectedResourceID
	return t, nil
}

type manifestProvider struct {
	manifestParser
	manifestFile       string
	namespace          string
	selectedResourceID string
	objectKind         string
}

func (t *manifestProvider) RunCommand(command string) (*os_provider.Command, error) {
	return nil, errors.New("k8s does not implement RunCommand")
}

func (t *manifestProvider) FileInfo(path string) (os_provider.FileInfoDetails, error) {
	return os_provider.FileInfoDetails{}, errors.New("k8s does not implement FileInfo")
}

func (t *manifestProvider) FS() afero.Fs {
	return &fsutil.NoFs{}
}

func (t *manifestProvider) Close() {}

func (t *manifestProvider) Capabilities() providers.Capabilities {
	return providers.Capabilities{}
}

func (t *manifestProvider) PlatformInfo() *platform.Platform {
	platformData := getPlatformInfo(t.objectKind, t.Runtime())
	if platformData != nil {
		return platformData
	}

	return &platform.Platform{
		Name:    "kubernetes",
		Title:   "Kubernetes Manifest",
		Kind:    providers.Kind_KIND_CODE,
		Runtime: t.Runtime(),
	}
}

func (t *manifestProvider) Kind() providers.Kind {
	return providers.Kind_KIND_API
}

func (t *manifestProvider) Runtime() string {
	return providers.RUNTIME_KUBERNETES_MANIFEST
}

func (t *manifestProvider) PlatformIdDetectors() []providers.PlatformIdDetector {
	return []providers.PlatformIdDetector{
		providers.TransportPlatformIdentifierDetector,
	}
}

func (t *manifestProvider) ServerVersion() *version.Info {
	return nil
}

func (t *manifestProvider) SupportedResourceTypes() (*resources.ApiResourceIndex, error) {
	return t.manifestParser.SupportedResourceTypes()
}

func (t *manifestProvider) ID() (string, error) {
	// If we are doing an admission control scan, we have 1 resource in the manifest and it has a UID.
	// Instead of using the file path to generate the ID, use the resource UID. We do this because for
	// CI/CD scans, the manifest is stored in a random file. This means we can potentially be scanning
	// the same resource multiple times but it will result in different assets because of the random
	// file name.

	if len(t.objects) == 1 {
		o, err := meta.Accessor(t.objects[0])
		if err == nil {
			if o.GetUID() != "" {
				return string(o.GetUID()), nil
			}
		}
	}

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

func (t *manifestProvider) PlatformIdentifier() (string, error) {
	if t.selectedResourceID != "" {
		return t.selectedResourceID, nil
	}

	uid, err := t.ID()
	if err != nil {
		return "", err
	}

	return NewPlatformID(uid), nil
}

func (t *manifestProvider) Identifier() (string, error) {
	return t.PlatformIdentifier()
}

func (t *manifestProvider) Name() (string, error) {
	// manifest parent directory name
	clusterName := common.ProjectNameFromPath(t.manifestFile)
	clusterName = "K8S Manifest " + clusterName
	return clusterName, nil
}

func (t *manifestProvider) AdmissionReviews() ([]admissionv1.AdmissionReview, error) {
	return []admissionv1.AdmissionReview{}, nil
}

func loadManifestFile(manifestFile string) ([]byte, error) {
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
