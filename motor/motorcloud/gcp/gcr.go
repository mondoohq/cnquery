package gcp

import (
	"log"
	"strings"
	"sync"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/google"
	"go.mondoo.io/mondoo/nexus/assets"
	"google.golang.org/api/cloudresourcemanager/v1"
	"google.golang.org/api/compute/v1"
)

func NewGCRImages() *GcrImages {
	return &GcrImages{}
}

type GcrImages struct{}

// lists a repository like "gcr.io/mondoo-base-infra"
func (a *GcrImages) ListRepository(root string, recursive bool) ([]*assets.Asset, error) {
	repo, err := name.NewRepository(root)
	if err != nil {
		log.Fatalln(err)
	}

	auth, err := google.Keychain.Resolve(repo.Registry)
	if err != nil {
		log.Fatalf("getting auth for %q: %v", root, err)
	}

	imgs := []*assets.Asset{}

	toAssetFunc := func(repo name.Repository, tags *google.Tags, err error) error {
		if err != nil {
			return err
		}

		for digest, manifest := range tags.Manifests {
			repoURL := repo.String()

			asset := &assets.Asset{
				ReferenceIDs: []string{MondooContainerImageID(digest)},
				// Name:         strings.Join(dImg.RepoTags, ","),
				Platform: &assets.Platform{
					Kind:    assets.Kind_KIND_CONTAINER_IMAGE,
					Runtime: "gcp gcr",
				},
				Connections: []*assets.Connection{
					&assets.Connection{
						Backend: assets.ConnectionBackend_CONNECTION_DOCKER_REGISTRY,
						Host:    repo.RegistryStr(),
					},
				},
				State:  assets.State_STATE_ONLINE,
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
			repoDigests := []string{repoURL + "@" + digest}
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

func (a *GcrImages) List() ([]*assets.Asset, error) {
	assets := []*assets.Asset{}
	// repoAssets, err := a.ListRepository("index.docker.io/mondoolabs/mondoo", false)
	// if err == nil && repoAssets != nil {
	// 	assets = append(assets, repoAssets...)
	// }
	// return assets, nil

	client, err := gcpClient(compute.CloudPlatformScope)
	resSrv, err := cloudresourcemanager.New(client)
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
