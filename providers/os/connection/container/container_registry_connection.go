package container

import (
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v10/providers/os/connection/container/auth"
	"go.mondoo.com/cnquery/v10/providers/os/connection/container/image"
	"go.mondoo.com/cnquery/v10/providers/os/connection/tar"
	"go.mondoo.com/cnquery/v10/providers/os/id/containerid"
)

// NewContainerRegistryImage loads a container image from a remote registry
func NewContainerRegistryImage(id uint32, conf *inventory.Config, asset *inventory.Asset) (*tar.TarConnection, error) {
	ref, err := name.ParseReference(conf.Host, name.WeakValidation)
	if err == nil {
		log.Debug().Str("ref", ref.Name()).Msg("found valid container registry reference")

		registryOpts := []image.Option{image.WithInsecure(conf.Insecure)}
		remoteOpts := auth.AuthOption(conf.Credentials)
		registryOpts = append(registryOpts, remoteOpts...)

		img, err := image.LoadImageFromRegistry(ref, registryOpts...)
		if err != nil {
			return nil, err
		}
		if asset.Connections[0].Options == nil {
			asset.Connections[0].Options = map[string]string{}
		}

		conn, err := tar.NewTarConnectionForContainer(id, conf, asset, img)
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
		name := repoName + "@" + containerid.ShortContainerImageID(imgDigest)
		if asset.Name == "" {
			asset.Name = name
		}
		if len(asset.PlatformIds) == 0 {
			asset.PlatformIds = []string{identifier}
		} else {
			asset.PlatformIds = append(asset.PlatformIds, identifier)
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
		asset.Labels = labels

		return conn, err
	}
	log.Debug().Str("image", conf.Host).Msg("Could not detect a valid repository url")
	return nil, err
}
