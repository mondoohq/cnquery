// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package gcr

import (
	"context"
	"log"
	"strings"
	"sync"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/google"
	"go.mondoo.com/cnquery/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/providers/os/id/containerid"
	"google.golang.org/api/cloudresourcemanager/v1"
)

func NewGCRImages() *GcrImages {
	return &GcrImages{}
}

type GcrImages struct{}

func (a *GcrImages) Name() string {
	return "GCP Container Registry Discover"
}

// lists a repository like "gcr.io/mondoo-base-infra"
func (a *GcrImages) ListRepository(repository string, recursive bool) ([]*inventory.Asset, error) {
	repo, err := name.NewRepository(repository)
	if err != nil {
		log.Fatalln(err)
	}

	auth, err := google.Keychain.Resolve(repo.Registry)
	if err != nil {
		log.Fatalf("getting auth for %q: %v", repository, err)
	}

	imgs := []*inventory.Asset{}

	toAssetFunc := func(repo name.Repository, tags *google.Tags, err error) error {
		if err != nil {
			return err
		}

		for digest, manifest := range tags.Manifests {
			repoURL := repo.String()
			imageUrl := repoURL + "@" + digest

			asset := &inventory.Asset{
				PlatformIds: []string{MondooContainerImageID(digest)},
				Name:        containerid.ShortContainerImageID(digest),
				Platform: &inventory.Platform{
					Kind:    "container-image",
					Runtime: "gcp-gcr",
				},

				Connections: []*inventory.Config{
					{
						Backend: "cr",
						Type:    "container-registry",
						Host:    imageUrl,
					},
				},
				State:  inventory.State_STATE_ONLINE,
				Labels: make(map[string]string),
			}

			// store digest
			asset.Labels["docker.io/digest"] = digest

			// store repo tags
			imageTags := []string{}
			for j := range manifest.Tags {
				imageTags = append(imageTags, repoURL+":"+manifest.Tags[j])
			}
			asset.Labels["docker.io/tags"] = strings.Join(imageTags, ",")

			// store repo digest
			repoDigests := []string{imageUrl}
			asset.Labels["docker.io/repo-digests"] = strings.Join(repoDigests, ",")

			imgs = append(imgs, asset)
		}
		return nil
	}

	// walk nested repos
	if recursive {
		err := google.Walk(repo, toAssetFunc, google.WithAuth(auth))
		if err != nil {
			return nil, err
		}
		return imgs, nil
	}

	// NOTE: since we're not recursing, we ignore tags.Children
	tags, err := google.List(repo, google.WithAuth(auth))
	if err != nil {
		return nil, err
	}

	err = toAssetFunc(repo, tags, nil)
	if err != nil {
		return nil, err
	}
	return imgs, nil
}

// List uses your GCP credentials to iterate over all your projects to identify potential repos
func (a *GcrImages) List() ([]*inventory.Asset, error) {
	assets := []*inventory.Asset{}

	resSrv, err := cloudresourcemanager.NewService(context.Background())
	if err != nil {
		return nil, err
	}

	projectsResp, err := resSrv.Projects.List().Do()
	if err != nil {
		return nil, err
	}

	var wg sync.WaitGroup

	wg.Add(len(projectsResp.Projects))
	mux := &sync.Mutex{}
	for i := range projectsResp.Projects {

		project := projectsResp.Projects[i].Name
		go func() {
			repoAssets, err := a.ListRepository("gcr.io/"+project, true)
			if err == nil && repoAssets != nil {
				mux.Lock()
				assets = append(assets, repoAssets...)
				mux.Unlock()
			}
			wg.Done()
		}()
	}

	wg.Wait()
	return assets, nil
}

// combine with docker image MondooContainerImageID
func MondooContainerImageID(id string) string {
	id = strings.Replace(id, "sha256:", "", -1)
	return "//platformid.api.mondoo.app/runtime/docker/images/" + id
}
