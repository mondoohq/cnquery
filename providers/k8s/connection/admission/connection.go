// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

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
	namespace string

	selectedResourceID string
	objectKind         string
}

// func newManifestProvider(selectedResourceID string, objectKind string, opts ...Option) (KubernetesProvider, error) {
func NewConnection(id uint32, asset *inventory.Asset, data string) (shared.Connection, error) {
	c := &Connection{
		// objectKind: objectKind,
		asset: asset,
		namespace:          asset.Connections[0].Options[shared.OPTION_NAMESPACE],
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

func (p *Connection) ServerVersion() *version.Info {
	return nil
}

func (p *Connection) SupportedResourceTypes() (*resources.ApiResourceIndex, error) {
	return p.ManifestParser.SupportedResourceTypes()
}

func (p *Connection) ID() uint32 {
	return p.id
}

func (c *Connection) AssetId() (string, error) {
	reviews, err := c.AdmissionReviews()
	if err != nil {
		return "", err
	}

	return shared.NewPlatformId(string(reviews[0].Request.UID)), nil
}

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
