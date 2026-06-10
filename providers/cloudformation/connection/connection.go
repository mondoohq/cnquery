// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
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
	cftTemplate cft.Template
	closer      func()
}

func NewCloudformationConnection(id uint32, asset *inventory.Asset, conf *inventory.Config) (*CloudformationConnection, error) {
	conn := &CloudformationConnection{
		Connection: plugin.NewConnection(id, asset),
		Conf:       conf,
		asset:      asset,
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

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	cftTemplate, err := parse.Reader(f)
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
