// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package manifest

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers/k8s/connection/shared"
	"go.mondoo.com/cnquery/v11/providers/k8s/connection/shared/resources"
	admissionv1 "k8s.io/api/admission/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/version"
)

var _ plugin.Closer = (*Connection)(nil)

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

func WithCloser(closer func()) Option {
	return func(p *Connection) {
		p.closer = closer
	}
}

type Connection struct {
	shared.ManifestParser
	plugin.Connection
	asset     *inventory.Asset
	namespace string

	manifestFile    string
	manifestContent []byte
	closer          func()
}

func NewGitConnection(id uint32, asset *inventory.Asset, opts ...Option) (shared.Connection, error) {
	path, closer, err := plugin.NewGitClone(asset)
	if err != nil {
		return nil, err
	}

	// After we have cloned the repo, we just work with the path. This makes sure consequent
	// connect calls will not trigger repo clone again.
	conf := asset.Connections[0]
	delete(conf.Options, shared.OPTION_GIT_HTTP)
	conf.Options[shared.OPTION_MANIFEST] = path

	opts = append(opts, WithCloser(closer), WithManifestFile(path))
	return NewConnection(id, asset, opts...)
}

func NewConnection(id uint32, asset *inventory.Asset, opts ...Option) (shared.Connection, error) {
	c := &Connection{
		Connection: plugin.NewConnection(id, asset),
		asset:      asset,
		namespace:  asset.Connections[0].Options[shared.OPTION_NAMESPACE],
	}

	for _, option := range opts {
		option(c)
	}

	manifest := []byte{}
	var err error

	clusterName := ""
	if len(c.manifestContent) > 0 {
		manifest = c.manifestContent
		clusterName = "K8s Manifest"
	} else if c.manifestFile != "" {
		manifest, err = shared.LoadManifestFile(c.manifestFile)
		if err != nil {
			return nil, err
		}
		// manifest parent directory name
		clusterName = shared.ProjectNameFromPath(c.manifestFile)
		clusterName = "K8s Manifest " + clusterName
	}
	// discovered assets pass by here
	// They already have a name, so do not override it here.
	if asset.Name == "" {
		asset.Name = clusterName
	}

	c.ManifestParser, err = shared.NewManifestParser(manifest, c.namespace, "")
	if err != nil {
		return nil, err
	}

	return c, nil
}

func (c *Connection) Close() {
	if c.closer != nil {
		c.closer()
	}
}

func (c *Connection) ServerVersion() *version.Info {
	return nil
}

func (c *Connection) SupportedResourceTypes() (*resources.ApiResourceIndex, error) {
	return c.ManifestParser.SupportedResourceTypes()
}

func (c *Connection) Name() string {
	return c.asset.Name
}

func (c *Connection) Runtime() string {
	return "k8s-manifest"
}

func (c *Connection) Platform() *inventory.Platform {
	return &inventory.Platform{
		Name:                  "k8s-manifest",
		Family:                []string{"k8s"},
		Kind:                  "code",
		Runtime:               c.Runtime(),
		Title:                 "Kubernetes Manifest",
		TechnologyUrlSegments: []string{"iac", "k8s-manifest"},
	}
}

func (c *Connection) Asset() *inventory.Asset {
	return c.asset
}

func (c *Connection) AssetId() (string, error) {
	// If we are doing an admission control scan, we have 1 resource in the manifest and it has a UID.
	// Instead of using the file path to generate the ID, use the resource UID. We do this because for
	// CI/CD scans, the manifest is stored in a random file. This means we can potentially be scanning
	// the same resource multiple times but it will result in different assets because of the random
	// file name.

	if len(c.Objects) == 1 && c.asset.Platform.Runtime == "k8s-admission" {
		o, err := meta.Accessor(c.Objects[0])
		if err == nil {
			if o.GetUID() != "" {
				return shared.NewPlatformId(string(o.GetUID())), nil
			}
		}
	}

	h := sha256.New()

	// special handling for embedded content (e.g. piped in via stdin)
	if len(c.manifestContent) > 0 {
		h.Write([]byte("stdin"))
		return hex.EncodeToString(h.Sum(nil)), nil
	}

	_, err := os.Stat(c.manifestFile)
	if err != nil {
		return "", errors.Wrap(err, "could not determine platform identifier for "+c.manifestFile)
	}

	absPath, err := filepath.Abs(c.manifestFile)
	if err != nil {
		return "", errors.Wrap(err, "could not determine platform identifier for "+c.manifestFile)
	}

	h.Write([]byte(absPath))
	return shared.NewPlatformId(hex.EncodeToString(h.Sum(nil))), nil
}

func (c *Connection) InventoryConfig() *inventory.Config {
	return c.asset.Connections[0]
}

func (p *Connection) AdmissionReviews() ([]admissionv1.AdmissionReview, error) {
	return []admissionv1.AdmissionReview{}, nil
}
