package admission

import (
	"bytes"
	"encoding/base64"
	"fmt"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/providers/k8s/connection/shared"
	"go.mondoo.com/cnquery/providers/k8s/connection/shared/resources"
	admission "k8s.io/api/admission/v1"
	"k8s.io/apimachinery/pkg/version"
)

type Connection struct {
	shared.ManifestParser
	runtime string
	id      uint32
	asset   *inventory.Asset

	selectedResourceID string
	objectKind         string
}

// func newManifestProvider(selectedResourceID string, objectKind string, opts ...Option) (KubernetesProvider, error) {
func NewConnection(id uint32, asset *inventory.Asset, data string) (shared.Connection, error) {
	c := &Connection{
		// objectKind: objectKind,
		asset: asset,
	}

	admission, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		log.Error().Err(err).Msg("failed to decode admission review")
		return nil, err
	}

	c.ManifestParser, err = shared.NewManifestParser(admission, "", "")
	if err != nil {
		return nil, err
	}

	res, err := c.AdmissionReviews()

	for _, r := range res {
		// For each admission we want to also parse the object as an individual asset so we
		// can show the admission review and the resource together in the CI/CD view.
		objs, err := resources.ResourcesFromManifest(bytes.NewReader(r.Request.Object.Raw))
		if err != nil {
			log.Error().Err(err).Msg("failed to parse object from admission review")
		}
		c.Objects = append(c.Objects, objs...)
	}
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
		Name:    "k8s-admission",
		Family:  []string{"k8s"},
		Kind:    "code",
		Runtime: "k8s-admission",
		Title:   "Kubernetes Admission",
	}
}

func (c *Connection) AdmissionReviews() ([]admission.AdmissionReview, error) {
	res, err := c.Resources("admissionreview.v1.admission", "", "")
	if err != nil {
		return nil, err
	}

	if len(res.Resources) < 1 {
		return nil, fmt.Errorf("no admission review found")
	}

	reviews := make([]admission.AdmissionReview, 0, len(res.Resources))
	for _, r := range res.Resources {
		reviews = append(reviews, *r.(*admission.AdmissionReview))
	}
	return reviews, nil
}
