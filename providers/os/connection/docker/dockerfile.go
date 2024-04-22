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
	"go.mondoo.com/cnquery/v11/providers/os/connection/fs"
	"go.mondoo.com/cnquery/v11/providers/os/connection/shared"
	"go.mondoo.com/cnquery/v11/utils/multierr"
	"go.mondoo.com/cnquery/v11/utils/urlx"
)

var _ shared.Connection = &DockerfileConnection{}

type DockerfileConnection struct {
	*fs.FileSystemConnection
	Filename string
}

func NewDockerfile(id uint32, conf *inventory.Config, asset *inventory.Asset) (*DockerfileConnection, error) {
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

	// if we have a regular file, we need to point the fs.Connection to
	// look at the folder instead and store the filename separately
	var filename string
	if !stat.IsDir() {
		filename = filepath.Base(absSrc)
		absSrc = filepath.Dir(absSrc)
		conf.Path = absSrc
	}

	fsconn, err := fs.NewConnection(id, conf, asset)
	if err != nil {
		return nil, err
	}

	asset.Platform = &inventory.Platform{
		Name:    "dockerfile",
		Title:   "Dockerfile",
		Family:  []string{"docker"},
		Kind:    "code",
		Runtime: "docker",
	}

	url, ok := asset.Connections[0].Options["ssh-url"]
	if ok {
		domain, org, repo, err := urlx.ParseGitSshUrl(url)
		if err != nil {
			return nil, err
		}
		platformID := "//platformid.api.mondoo.app/runtime/dockerfile/domain/" + domain + "/org/" + org + "/repo/" + repo
		asset.Connections[0].PlatformId = platformID
		asset.PlatformIds = []string{platformID}
		asset.Name = "Dockerfile analysis " + org + "/" + repo

	} else {
		h := sha256.New()
		h.Write([]byte(absSrc))
		hash := hex.EncodeToString(h.Sum(nil))
		platformID := "//platformid.api.mondoo.app/runtime/dockerfile/hash/" + hash

		asset.Connections[0].PlatformId = platformID
		asset.PlatformIds = []string{platformID}
		asset.Name = "Dockerfile analysis " + filename
	}

	return &DockerfileConnection{
		FileSystemConnection: fsconn,
		Filename:             filename,
	}, nil
}
