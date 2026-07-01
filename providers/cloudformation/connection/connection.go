// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"bytes"
	"errors"
	"os"

	"github.com/aws-cloudformation/rain/cft"
	"github.com/aws-cloudformation/rain/cft/parse"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
)

var _ plugin.Connection = (*CloudformationConnection)(nil)

type CloudformationConnection struct {
	plugin.Connection
	Conf  *inventory.Config
	asset *inventory.Asset
	// Add custom connection fields here
	path        string
	content     string
	cftTemplate cft.Template
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
	cc := asset.Connections[0]
	path := cc.Options["path"]
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

	return conn, nil
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
