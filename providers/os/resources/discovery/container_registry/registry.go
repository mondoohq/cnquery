// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package container_registry

import (
	"fmt"
	"net/url"

	"github.com/cockroachdb/errors"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/vault"
	"go.mondoo.com/cnquery/v11/providers/os/connection/container/auth"
	"go.mondoo.com/cnquery/v11/providers/os/connection/container/image"
	"go.mondoo.com/cnquery/v11/providers/os/connection/shared"
	"go.mondoo.com/cnquery/v11/providers/os/id/containerid"
)

func NewContainerRegistryResolver(opts ...remote.Option) *DockerRegistryImages {
	return &DockerRegistryImages{
		opts: opts,
	}
}

type DockerRegistryImages struct {
	opts     []remote.Option
	Insecure bool
}

func (a *DockerRegistryImages) remoteOptions(name string) []remote.Option {
	// either use the provided options or the default options
	if len(a.opts) > 0 {
		return a.opts
	}
	log.Debug().Str("name", name).Msg("using default remote options")
	return auth.DefaultOpts(name, a.Insecure)
}

func (a *DockerRegistryImages) Repositories(reg name.Registry) ([]string, error) {
	n := 100
	last := ""
	var res []string
	for {
		opts := a.remoteOptions(reg.Name())
		page, err := remote.CatalogPage(reg, last, n, opts...)
		if err != nil {
			return nil, err
		}

		if len(page) > 0 {
			last = page[len(page)-1]
			res = append(res, page...)
		}

		if len(page) < n {
			break
		}
	}

	return res, nil
}

// ListRegistry tries to iterate over all repositores in one registry
// eg. 1234567.dkr.ecr.us-east-1.amazonaws.com
func (a *DockerRegistryImages) ListRegistry(registry string) ([]*inventory.Asset, error) {
	reg, err := name.NewRegistry(registry)
	if err != nil {
		return nil, errors.Wrap(err, "resolve registry")
	}

	repos, err := a.Repositories(reg)
	if err != nil {
		return nil, err
	}

	assets := []*inventory.Asset{}
	for i := range repos {
		repoName := reg.RegistryStr() + "/" + repos[i]
		log.Debug().Str("repository", repoName).Msg("discovered repository")

		// iterate over all repository digests
		repoImages, err := a.ListRepository(repoName)
		if err != nil {
			return nil, err
		}
		assets = append(assets, repoImages...)
	}

	return assets, nil
}

// ListRepository tries to fetch all details about a specific repository
// index.docker.io/mondoo
// index.docker.io/mondoo/client
// harbor.lunalectric.com/library
// harbor.lunalectric.com/library/ubuntu
func (a *DockerRegistryImages) ListRepository(repoName string) ([]*inventory.Asset, error) {
	assets := []*inventory.Asset{}

	repo, err := name.NewRepository(repoName)
	if err != nil {
		return nil, err
	}

	// fetch tags
	opts := a.remoteOptions(repo.Name())
	tags, err := remote.List(repo, opts...)
	if err != nil {
		return nil, handleUnauthorizedError(err, repo.Name())
	}

	foundAssets := map[string]*inventory.Asset{}
	for i := range tags {
		repoWithTag := repo.Name() + ":" + tags[i]

		ref, err := name.ParseReference(repoWithTag)
		if err != nil {
			return nil, fmt.Errorf("parsing reference %q: %v", repoWithTag, err)
		}

		a, err := a.toAsset(ref, nil, opts...)
		if err != nil {
			return nil, err
		}
		if foundAsset, ok := foundAssets[a.PlatformIds[0]]; ok {
			// only add tags to the first asset
			foundAsset.Labels["docker.io/tags"] = foundAsset.Labels["docker.io/tags"] + "," + a.Labels["docker.io/tags"]
			log.Debug().Str("tags", foundAsset.Labels["docker.io/tags"]).Str("image", foundAsset.Name).Msg("found additional tags for image")
			continue
		}
		foundAssets[a.PlatformIds[0]] = a
	}

	// flatten map
	for k := range foundAssets {
		assets = append(assets, foundAssets[k])
	}
	return assets, nil
}

// ListImages either takes a registry or a repository and tries to fetch as many images as possible
func (a *DockerRegistryImages) ListImages(repoName string) ([]*inventory.Asset, error) {
	url, err := url.Parse("//" + repoName)
	if err != nil {
		return nil, fmt.Errorf("registries must be valid RFC 3986 URI authorities: %s", repoName)
	}

	if url.Host == repoName {
		// fetch registry information
		return a.ListRegistry(repoName)
	} else {
		// fetch repo information
		return a.ListRepository(repoName)
	}
}

func (a *DockerRegistryImages) GetImage(ref name.Reference, creds []*vault.Credential, opts ...remote.Option) (*inventory.Asset, error) {
	return a.toAsset(ref, creds, opts...)
}

func (a *DockerRegistryImages) toAsset(ref name.Reference, creds []*vault.Credential, opts ...remote.Option) (*inventory.Asset, error) {
	desc, err := image.GetImageDescriptor(ref, opts...)
	if err != nil {
		return nil, handleUnauthorizedError(err, ref.Name())
	}
	imgDigest := desc.Digest.String()
	repoName := ref.Context().Name()
	imgTag := ref.Context().Tag(ref.Identifier()).TagStr()
	name := repoName + "@" + containerid.ShortContainerImageID(imgDigest)
	imageUrl := repoName + "@" + imgDigest
	asset := &inventory.Asset{
		PlatformIds: []string{containerid.MondooContainerImageID(imgDigest)},
		Name:        name,
		Connections: []*inventory.Config{
			{
				Type:        string(shared.Type_RegistryImage),
				Host:        imageUrl,
				Credentials: creds,
			},
		},
		State:  inventory.State_STATE_ONLINE,
		Labels: make(map[string]string),
	}

	// store digest and tag
	asset.Labels["docker.io/digest"] = imgDigest
	asset.Labels["docker.io/tags"] = imgTag
	log.Debug().Strs("platform-ids", asset.PlatformIds).Msg("asset platform ids")
	return asset, nil
}
