// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package docker

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"

	"go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v11/providers/os/connection/local"
	"go.mondoo.com/cnquery/v11/providers/os/connection/shared"
	"go.mondoo.com/cnquery/v11/utils/multierr"
	"go.mondoo.com/cnquery/v11/utils/urlx"
)

var _ shared.Connection = &DockerfileConnection{}

type DockerfileConnection struct {
	*local.LocalConnection
	// Filename must be the absolute path of the Dockerfile so
	// that we find the file downstream
	Filename string
}

func NewDockerfileConnection(_ uint32,
	conf *inventory.Config, asset *inventory.Asset,
	localConn *local.LocalConnection, localFamily []string) (*DockerfileConnection, error) {
	if conf == nil {
		return nil, errors.New("missing configuration to create dockerfile connection")
	}

	src := conf.Path
	if src == "" {
		return nil, errors.New("please specify a target path for the dockerfile connection")
	}

	absSrc, err := filepath.Abs(src)
	if err != nil {
		return nil, multierr.Wrap(err, "can't get absolute path for dockerfile")
	}

	stat, err := os.Stat(absSrc)
	if err != nil {
		return nil, err
	}

	var filename string
	if !stat.IsDir() {
		filename = filepath.Base(absSrc)
		conf.Path = absSrc
	}

	asset.Platform = &inventory.Platform{
		Name:    "dockerfile",
		Title:   "Dockerfile",
		Family:  []string{"docker"},
		Kind:    "code",
		Runtime: "docker",
	}
	// this helps with running commands against the local connection
	asset.Platform.Family = append(asset.Platform.Family, localFamily...)

	if url, ok := conf.Options["ssh-url"]; ok {
		domain, org, repo, err := urlx.ParseGitSshUrl(url)
		if err != nil {
			return nil, err
		}
		platformID := "//platformid.api.mondoo.app/runtime/dockerfile/domain/" + domain + "/org/" + org + "/repo/" + repo
		conf.PlatformId = platformID
		asset.PlatformIds = []string{platformID}
		asset.Name = "Dockerfile analysis " + org + "/" + repo

	} else {
		h := sha256.New()
		h.Write([]byte(absSrc))
		hash := hex.EncodeToString(h.Sum(nil))
		platformID := "//platformid.api.mondoo.app/runtime/dockerfile/hash/" + hash

		conf.PlatformId = platformID
		asset.PlatformIds = []string{platformID}
		asset.Name = "Dockerfile analysis " + filename
	}

	conn := &DockerfileConnection{
		LocalConnection: localConn,
		// here we must use the absolute path of the Dockerfile so
		// that we find the file downstream
		Filename: absSrc,
	}

	return conn, nil
}
