package k8s

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"io/fs"
	"os"
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
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	"k8s.io/apimachinery/pkg/version"
)

type Option func(*manifestProvider)

func WithNamespace(namespace string) Option {
	return func(p *manifestProvider) {
		p.namespace = namespace
	}
}

func WithManifestFile(filename string) Option {
	return func(p *manifestProvider) {
		p.manifestFile = filename
	}
}

func WithManifestContent(data []byte) Option {
	return func(p *manifestProvider) {
		p.manifestContent = data
	}
}

func newManifestProvider(selectedResourceID string, objectKind string, opts ...Option) (KubernetesProvider, error) {
	p := &manifestProvider{
		objectKind: objectKind,
	}

	for _, option := range opts {
		option(p)
	}

	manifest := []byte{}
	var err error

	if len(p.manifestContent) > 0 {
		manifest = p.manifestContent
		p.assetName = "K8s Manifest"
	} else if p.manifestFile != "" {
		manifest, err = loadManifestFile(p.manifestFile)
		if err != nil {
			return nil, err
		}
		// manifest parent directory name
		clusterName := common.ProjectNameFromPath(p.manifestFile)
		clusterName = "K8s Manifest " + clusterName
		p.assetName = clusterName
	}

	p.manifestParser, err = newManifestParser(manifest, p.namespace, selectedResourceID)
	if err != nil {
		return nil, err
	}

	p.selectedResourceID = selectedResourceID
	return p, nil
}

type manifestProvider struct {
	manifestParser
	assetName          string
	manifestFile       string
	manifestContent    []byte
	namespace          string
	selectedResourceID string
	objectKind         string
}

func (p *manifestProvider) RunCommand(command string) (*os_provider.Command, error) {
	return nil, errors.New("k8s does not implement RunCommand")
}

func (p *manifestProvider) FileInfo(path string) (os_provider.FileInfoDetails, error) {
	return os_provider.FileInfoDetails{}, errors.New("k8s does not implement FileInfo")
}

func (p *manifestProvider) FS() afero.Fs {
	return &fsutil.NoFs{}
}

func (p *manifestProvider) Close() {}

func (p *manifestProvider) Capabilities() providers.Capabilities {
	return providers.Capabilities{}
}

func (p *manifestProvider) PlatformInfo() *platform.Platform {
	platformData := getPlatformInfo(p.objectKind, p.Runtime())
	if platformData != nil {
		return platformData
	}

	return &platform.Platform{
		Name:    "k8s-manifest",
		Title:   "Kubernetes Manifest",
		Kind:    p.Kind(),
		Family:  []string{"k8s"},
		Runtime: p.Runtime(),
	}
}

func (p *manifestProvider) Kind() providers.Kind {
	return providers.Kind_KIND_CODE
}

func (p *manifestProvider) Runtime() string {
	return providers.RUNTIME_KUBERNETES_MANIFEST
}

func (p *manifestProvider) PlatformIdDetectors() []providers.PlatformIdDetector {
	return []providers.PlatformIdDetector{
		providers.TransportPlatformIdentifierDetector,
	}
}

func (p *manifestProvider) ServerVersion() *version.Info {
	return nil
}

func (p *manifestProvider) SupportedResourceTypes() (*resources.ApiResourceIndex, error) {
	return p.manifestParser.SupportedResourceTypes()
}

func (p *manifestProvider) ID() (string, error) {
	// If we are doing an admission control scan, we have 1 resource in the manifest and it has a UID.
	// Instead of using the file path to generate the ID, use the resource UID. We do this because for
	// CI/CD scans, the manifest is stored in a random file. This means we can potentially be scanning
	// the same resource multiple times but it will result in different assets because of the random
	// file name.

	if len(p.objects) == 1 {
		o, err := meta.Accessor(p.objects[0])
		if err == nil {
			if o.GetUID() != "" {
				return string(o.GetUID()), nil
			}
		}
	}

	h := sha256.New()

	// special handling for embedded content (e.g. piped in via stdin)
	if len(p.manifestContent) > 0 {
		h.Write([]byte("stdin"))
		return hex.EncodeToString(h.Sum(nil)), nil
	}

	_, err := os.Stat(p.manifestFile)
	if err != nil {
		return "", errors.Wrap(err, "could not determine platform identifier for "+p.manifestFile)
	}

	absPath, err := filepath.Abs(p.manifestFile)
	if err != nil {
		return "", errors.Wrap(err, "could not determine platform identifier for "+p.manifestFile)
	}

	h.Write([]byte(absPath))
	return hex.EncodeToString(h.Sum(nil)), nil
}

func (p *manifestProvider) Identifier() (string, error) {
	if p.selectedResourceID != "" {
		return p.selectedResourceID, nil
	}

	uid, err := p.ID()
	if err != nil {
		return "", err
	}

	return NewPlatformID(uid), nil
}

func (p *manifestProvider) Name() (string, error) {
	return p.assetName, nil
}

func (p *manifestProvider) AdmissionReviews() ([]admissionv1.AdmissionReview, error) {
	return []admissionv1.AdmissionReview{}, nil
}

func loadManifestFile(manifestFile string) ([]byte, error) {
	log.Debug().Str("filename", manifestFile).Msg("loading manifest file")
	var input io.Reader

	// return all resources from manifest
	filenames := []string{}

	fi, err := os.Stat(manifestFile)
	if err != nil {
		return nil, err
	}

	if fi.IsDir() {
		yamlDecoder := yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)
		filepath.WalkDir(manifestFile, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			// only load yaml files for now
			if !d.IsDir() {
				ext := filepath.Ext(path)
				if ext != ".yaml" && ext != ".yml" {
					log.Debug().Str("file", path).Msg("ignore file, no .yaml or .yml ending")
					return nil
				}
				// check whether this is valid k8s yaml
				content, err := os.ReadFile(path)
				if err != nil {
					log.Debug().Str("file", path).Err(err).Msg("ignore file, could not read file")
					return nil
				}
				// At this point, we do not care about specific schemes, just whether the file is a valid k8s yaml
				_, _, err = yamlDecoder.Decode(content, nil, nil)
				if err != nil {
					// the err contains the file content, which is not useful in the output
					errorString := ""
					if len(err.Error()) > 40 {
						errorString = err.Error()[:40] + "..."
					} else {
						errorString = err.Error()
					}
					log.Debug().Str("file", path).Str("error", errorString).Msg("ignore file, no valid kubernetes yaml")
					return nil
				}
				log.Debug().Str("file", path).Msg("add file to manifest loading")
				filenames = append(filenames, path)
			}

			return nil
		})
	} else {
		filenames = append(filenames, manifestFile)
	}

	input, err = resources.MergeManifestFiles(filenames)
	if err != nil {
		return nil, err
	}

	return io.ReadAll(input)
}
