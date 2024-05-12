// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package sbom

import (
	"bytes"
	"errors"
	"github.com/google/uuid"
	"go.mondoo.com/cnquery/v11/providers/os/fsutil"
	"os"

	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers/os/connection/shared"
	"go.mondoo.com/cnquery/v11/sbom"
	"go.mondoo.com/ranger-rpc/codes"
	"go.mondoo.com/ranger-rpc/status"
)

var (
	_ shared.Connection = (*Connection)(nil)
)

func NewConnection(id uint32, conf *inventory.Config, asset *inventory.Asset) (*Connection, error) {
	if conf == nil {
		return nil, status.Error(codes.InvalidArgument, "missing config")
	}

	// convert sbom to recording
	data, err := os.ReadFile(conf.Path)
	if err != nil {
		return nil, err

	}

	decoder := []sbom.Decoder{}

	decoder = append(decoder,
		sbom.NewCycloneDX(sbom.FormatCycloneDxJSON),
		sbom.NewCycloneDX(sbom.FormatCycloneDxXML),
		sbom.NewSPDX(sbom.FormatSpdxTagValue),
		sbom.NewSPDX(sbom.FormatSpdxJSON),
	)

	var sbomReport *sbom.Sbom
	found := false
	for i := range decoder {
		sbomReport, err = decoder[i].Parse(bytes.NewReader(data))
		if err == nil {
			found = true
			break
		}
	}
	if !found {
		return nil, errors.New("unsupported sbom format")
	}

	// Platform need to be set, otherwise it will panic
	asset.Platform = &inventory.Platform{}
	if sbomReport.Asset.Platform != nil {
		asset.Name = sbomReport.Asset.Name
		asset.PlatformIds = sbomReport.Asset.PlatformIds
		asset.Platform = &inventory.Platform{
			Title:   sbomReport.Asset.Platform.Title,
			Name:    sbomReport.Asset.Platform.Name,
			Version: sbomReport.Asset.Platform.Version,
			Build:   sbomReport.Asset.Platform.Build,
			Arch:    sbomReport.Asset.Platform.Arch,
			Family:  sbomReport.Asset.Platform.Family,
		}
	}

	return &Connection{
		Connection: plugin.NewConnection(id, asset),
		asset:      asset,
		sbom:       sbomReport,
		fs:         fsutil.NoFs{},
	}, nil
}

type Connection struct {
	plugin.Connection
	Conf  *inventory.Config
	asset *inventory.Asset
	fs    afero.Fs
	sbom  *sbom.Sbom
}

func (c *Connection) RunCommand(command string) (*shared.Command, error) {
	return nil, plugin.ErrRunCommandNotImplemented
}

func (c *Connection) FileSystem() afero.Fs {
	return c.fs
}

func (c *Connection) FileInfo(path string) (shared.FileInfoDetails, error) {
	return shared.FileInfoDetails{}, errors.New("not implemented")
}

func (c *Connection) Capabilities() shared.Capabilities {
	return shared.Capabilities(0)
}

func (c *Connection) Identifier() (string, error) {
	// TODO: revisit this approach when we have a better way to uniquely identify the asset
	// behind the sbom file
	return "//platformid.api.mondoo.app/runtime/sbom/uuid/" + uuid.New().String(), nil
}

func (c *Connection) Name() string {
	return string(shared.Type_SBOM)
}

func (c *Connection) Type() shared.ConnectionType {
	return shared.Type_SBOM
}

func (c *Connection) Asset() *inventory.Asset {
	return c.asset
}

func (p *Connection) UpdateAsset(asset *inventory.Asset) {
	p.asset = asset
}

func (p *Connection) GetRecording() *recordingCallbacks {
	r, err := newRecording(p.asset, p.sbom)
	if err != nil {
		log.Error().Err(err).Msg("failed to create recording")
		return nil
	}
	return newRecordingCallbacks(r)
}
