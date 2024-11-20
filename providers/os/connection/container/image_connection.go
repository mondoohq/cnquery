// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package container

import (
	"errors"
	"os"
	"slices"

	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v11/cli/tmp"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers/os/connection/container/auth"
	"go.mondoo.com/cnquery/v11/providers/os/connection/container/image"
	"go.mondoo.com/cnquery/v11/providers/os/connection/tar"
	"go.mondoo.com/cnquery/v11/providers/os/id/containerid"
	"go.mondoo.com/cnquery/v11/providers/os/resources/discovery/container_registry"
)

// NewImageConnection uses a container image reference as input and creates a tar connection
func NewImageConnection(id uint32, conf *inventory.Config, asset *inventory.Asset, img v1.Image) (*tar.Connection, error) {
	// FIXME: DEPRECATED, remove in v12.0 vv
	// The DelayDiscovery flag should always be set from v12
	if conf.Options == nil || conf.Options[plugin.DISABLE_DELAYED_DISCOVERY_OPTION] == "" {
		conf.DelayDiscovery = true // Delay discovery, to make sure we don't directly download the image
	}
	// ^^
	f, err := tmp.File()
	if err != nil {
		return nil, err
	}

	conf.Options[tar.OPTION_FILE] = f.Name()

	return tar.NewConnection(id, conf, asset,
		tar.WithFetchFn(func() (string, error) {
			log.Debug().Msg("tar> starting image extract to temporary file")
			err = tar.StreamToTmpFile(mutate.Extract(img), f)
			if err != nil {
				_ = os.Remove(f.Name())
				return "", err
			}
			log.Debug().Msg("tar> extracted image to temporary file")
			return f.Name(), nil
		}),
		tar.WithCloseFn(func() {
			log.Debug().Str("tar", f.Name()).Msg("tar> remove temporary tar file on connection close")
			_ = os.Remove(f.Name())
		}),
	)
}

// NewRegistryImage loads a container image from a remote registry
func NewRegistryImage(id uint32, conf *inventory.Config, asset *inventory.Asset) (*tar.Connection, error) {
	ref, err := name.ParseReference(conf.Host, name.WeakValidation)
	if err != nil {
		return nil, errors.New("invalid container registry reference: " + conf.Host)
	}
	log.Debug().Str("ref", ref.Name()).Msg("found valid container registry reference")

	registryOpts, err := container_registry.RemoteOptionsFromConfigOptions(conf)
	if err != nil {
		return nil, err
	}
	registryOpts = append(registryOpts, auth.AuthOption(ref.Name(), conf.Credentials))
	img, err := image.LoadImageFromRegistry(ref, registryOpts...)
	if err != nil {
		return nil, err
	}
	if conf.Options == nil {
		conf.Options = map[string]string{}
	}

	conn, err := NewImageConnection(id, conf, asset, img)
	if err != nil {
		return nil, err
	}

	var identifier string
	hash, err := img.Digest()
	if err == nil {
		identifier = containerid.MondooContainerImageID(hash.String())
	}

	conn.PlatformIdentifier = identifier
	conn.Metadata.Name = containerid.ShortContainerImageID(hash.String())

	repoName := ref.Context().Name()
	imgDigest := hash.String()
	containerAssetName := repoName + "@" + containerid.ShortContainerImageID(imgDigest)
	if asset.Name == "" {
		asset.Name = containerAssetName
	}
	if len(asset.PlatformIds) == 0 {
		asset.PlatformIds = []string{identifier}
	} else {
		if !slices.Contains(asset.PlatformIds, identifier) {
			asset.PlatformIds = append(asset.PlatformIds, identifier)
		}
	}

	// set the platform architecture using the image configuration
	imgConfig, err := img.ConfigFile()
	if err == nil {
		conn.PlatformArchitecture = imgConfig.Architecture
	}

	labels := map[string]string{}
	labels["docker.io/digests"] = ref.String()

	manifest, err := img.Manifest()
	if err == nil {
		labels["mondoo.com/image-id"] = manifest.Config.Digest.String()
	}

	conn.Metadata.Labels = labels
	if asset.Labels == nil {
		asset.Labels = map[string]string{}
	}

	for k, v := range labels {
		asset.Labels[k] = v
	}

	return conn, err
}

func NewFromTar(id uint32, conf *inventory.Config, asset *inventory.Asset) (*tar.Connection, error) {
	if conf == nil || len(conf.Options[tar.OPTION_FILE]) == 0 {
		return nil, errors.New("tar provider requires a valid tar file")
	}

	if conf.Options == nil {
		conf.Options = map[string]string{}
	}

	filename := conf.Options[tar.OPTION_FILE]
	var identifier string

	// try to determine if the tar is a container image
	img, iErr := tarball.ImageFromPath(filename, nil)
	if iErr != nil {
		return nil, iErr
	}

	hash, err := img.Digest()
	if err != nil {
		return nil, err
	}
	identifier = containerid.MondooContainerImageID(hash.String())

	// we need to extract the image from the tar file and create a new tar connection
	imageFilename := ""

	f, err := tmp.File()
	if err != nil {
		return nil, err
	}
	imageFilename = f.Name()
	conf.Options[tar.OPTION_FILE] = imageFilename

	c, err := tar.NewConnection(id, conf, asset,
		tar.WithFetchFn(func() (string, error) {
			err = tar.StreamToTmpFile(mutate.Extract(img), f)
			if err != nil {
				_ = os.Remove(imageFilename)
				return imageFilename, err
			}
			return imageFilename, nil
		}),
		tar.WithCloseFn(func() {
			// remove temporary file on stream close
			log.Debug().Str("tar", imageFilename).Msg("tar> remove temporary flattened image file on connection close")
			_ = os.Remove(imageFilename)
		}),
	)
	if err != nil {
		return nil, err
	}

	c.PlatformIdentifier = identifier
	return c, nil
}
