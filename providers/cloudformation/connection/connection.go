// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"

	"github.com/aws-cloudformation/rain/cft"
	"github.com/aws-cloudformation/rain/cft/parse"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
)

var (
	_ plugin.Connection = (*CloudformationConnection)(nil)
	_ plugin.Closer     = (*CloudformationConnection)(nil)
)

type CloudformationConnection struct {
	plugin.Connection
	Conf  *inventory.Config
	asset *inventory.Asset
	// Add custom connection fields here
	path        string
	content     string
	cftTemplate cft.Template
	closer      func()
}

func NewCloudformationConnection(id uint32, asset *inventory.Asset, conf *inventory.Config) (*CloudformationConnection, error) {
	conn := &CloudformationConnection{
		Connection: plugin.NewConnection(id, asset),
		Conf:       conf,
		asset:      asset,
	}
	// initialize your connection here
	if len(asset.Connections) == 0 {
		return nil, errors.New("no connection options for asset")
	}

	// If a git clone is performed below, clean up the temporary directory on any
	// error path. Close() is a no-op when nothing was cloned, and the guard is
	// disarmed once the connection is returned and takes ownership of cleanup.
	cleanupClone := true
	defer func() {
		if cleanupClone {
			conn.Close()
		}
	}()

	cc := asset.Connections[0]
	path := cc.Options["path"]
	// When discovered from a git repository (e.g. by the GitHub provider) the
	// asset carries the repo URL plus a repo-relative path to the template.
	// Clone the repo and resolve the template within the checkout. We keep the
	// repo-relative path in the options so the detector can build a stable,
	// human-friendly asset name and platform ID from the repo rather than the
	// temporary clone directory.
	if _, ok := cc.Options["http-url"]; ok {
		clonePath, closer, err := plugin.NewGitClone(asset)
		if err != nil {
			return nil, err
		}
		conn.closer = closer
		path = filepath.Join(clonePath, path)
	}
	conn.path = path

	// Read the raw bytes up front and parse from them, so we can later extract
	// the source text a resource/output/parameter spans for file-context.
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	// Convert to a string once here; nodeContext extracts source ranges from it
	// for every resource/output/parameter, and a per-call []byte->string copy of
	// a large template would be wasteful.
	conn.content = string(data)

	cftTemplate, err := parse.Reader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	if cftTemplate == nil {
		return nil, errors.New("cftTemplate is nil")
	}
	conn.cftTemplate = *cftTemplate

	cleanupClone = false
	return conn, nil
}

// Close cleans up any temporary directory created by a git clone.
func (c *CloudformationConnection) Close() {
	if c.closer != nil {
		c.closer()
	}
}

func (c *CloudformationConnection) Name() string {
	return "cloudformation"
}

func (c *CloudformationConnection) Asset() *inventory.Asset {
	return c.asset
}

func (c *CloudformationConnection) CftTemplate() cft.Template {
	return c.cftTemplate
}

// Path returns the template file path this connection was opened with.
func (c *CloudformationConnection) Path() string {
	return c.path
}

// Content returns the raw text of the template file, used to extract the
// source text a resource/output/parameter spans for file-context. The string
// is converted once at connection time so repeated range extractions don't
// each re-copy the whole template.
func (c *CloudformationConnection) Content() string {
	return c.content
}
