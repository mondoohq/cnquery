package manifest

import (
	"go.mondoo.com/cnquery/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/providers/k8s/connection/shared"
	"go.mondoo.com/cnquery/providers/k8s/connection/shared/resources"
	admissionv1 "k8s.io/api/admission/v1"
	"k8s.io/apimachinery/pkg/version"
)

type Option func(*Connection)

func WithNamespace(namespace string) Option {
	return func(p *Connection) {
		p.namespace = namespace
	}
}

func WithManifestFile(filename string) Option {
	return func(p *Connection) {
		p.manifestFile = filename
	}
}

func WithManifestContent(data []byte) Option {
	return func(p *Connection) {
		p.manifestContent = data
	}
}

const (
	Api shared.ConnectionType = "api"
)

type Connection struct {
	shared.ManifestParser
	runtime   string
	id        uint32
	asset     *inventory.Asset
	namespace string

	manifestFile       string
	manifestContent    []byte
	selectedResourceID string
	objectKind         string
}

// func newManifestProvider(selectedResourceID string, objectKind string, opts ...Option) (KubernetesProvider, error) {
func NewConnection(id uint32, asset *inventory.Asset, opts ...Option) (shared.Connection, error) {
	c := &Connection{
		// objectKind: objectKind,
		asset: asset,
	}

	for _, option := range opts {
		option(c)
	}

	manifest := []byte{}
	var err error

	if len(c.manifestContent) > 0 {
		manifest = c.manifestContent
		// c.assetName = "K8s Manifest"
	} else if c.manifestFile != "" {
		manifest, err = shared.LoadManifestFile(c.manifestFile)
		if err != nil {
			return nil, err
		}
		// manifest parent directory name
		clusterName := shared.ProjectNameFromPath(c.manifestFile)
		clusterName = "K8s Manifest " + clusterName
		// c.assetName = clusterName
	}

	c.ManifestParser, err = shared.NewManifestParser(manifest, c.namespace, "")
	if err != nil {
		return nil, err
	}

	// c.selectedResourceID = selectedResourceID
	return c, nil
}

// func (p *manifestProvider) PlatformInfo() *platform.Platform {
// 	platformData := getPlatformInfo(p.objectKind, p.Runtime())
// 	if platformData != nil {
// 		return platformData
// 	}

// 	return &platform.Platform{
// 		Name:    "k8s-manifest",
// 		Title:   "Kubernetes Manifest",
// 		Kind:    p.Kind(),
// 		Family:  []string{"k8s"},
// 		Runtime: p.Runtime(),
// 	}
// }

func (p *Connection) ServerVersion() *version.Info {
	return nil
}

func (p *Connection) SupportedResourceTypes() (*resources.ApiResourceIndex, error) {
	return p.ManifestParser.SupportedResourceTypes()
}

func (p *Connection) ID() uint32 {
	return p.id
}

// func (p *manifestProvider) Identifier() (string, error) {
// 	if p.selectedResourceID != "" {
// 		return p.selectedResourceID, nil
// 	}

// 	uid, err := p.ID()
// 	if err != nil {
// 		return "", err
// 	}

// 	return NewPlatformID(uid), nil
// }

func (p *Connection) Name() string {
	return p.asset.Name
}

func (c *Connection) Platform() *inventory.Platform {
	return &inventory.Platform{
		Name:    "k8s-manifest",
		Family:  []string{"k8s"},
		Kind:    "code",
		Runtime: "k8s-manifest",
		Title:   "Kubernetes Manifest",
	}
}

func (p *Connection) AdmissionReviews() ([]admissionv1.AdmissionReview, error) {
	return []admissionv1.AdmissionReview{}, nil
}
