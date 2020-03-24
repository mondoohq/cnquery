package docker

import (
	"fmt"
	"strings"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"go.mondoo.io/mondoo/motor/runtime"
	"go.mondoo.io/mondoo/nexus/assets"
)

func NewDockerRegistryImages() *DockerRegistryImages {
	return &DockerRegistryImages{}
}

type DockerRegistryImages struct{}

func (a *DockerRegistryImages) Repositories(reg name.Registry) ([]string, error) {
	n := 100
	last := ""
	var res []string
	for {
		page, err := remote.CatalogPage(reg, last, n, remote.WithAuthFromKeychain(authn.DefaultKeychain))
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

func (a *DockerRegistryImages) Digest(r string) (string, error) {
	ref, err := name.ParseReference(r)
	if err != nil {
		return "", fmt.Errorf("parsing reference %q: %v", r, err)
	}

	desc, err := remote.Get(ref, remote.WithAuthFromKeychain(authn.DefaultKeychain))
	if err != nil {
		return "", err
	}
	return desc.Digest.String(), nil
}

func (a *DockerRegistryImages) Tags(r string) ([]string, error) {
	repo, err := name.NewRepository(r)
	if err != nil {
		return nil, fmt.Errorf("parsing repo %q: %v", r, err)
	}

	return remote.List(repo, remote.WithAuthFromKeychain(authn.DefaultKeychain))
}

func (a *DockerRegistryImages) Repository(reponame string) (map[string][]string, error) {
	fmt.Printf("fetch repo data %s\n", reponame)
	repo, err := name.NewRepository(reponame)
	if err != nil {
		return nil, err
	}

	tags, err := a.Tags(repo.Name())
	if err != nil {
		return nil, err
	}

	digestsImgs := map[string][]string{}

	for i := range tags {
		repoWithTag := repo.Name() + ":" + tags[i]
		fmt.Println(repoWithTag)
		digest, err := a.Digest(repoWithTag)
		fmt.Println(digest)
		if err != nil {
			return nil, err
		}
		_, ok := digestsImgs[digest]
		if !ok {
			digestsImgs[digest] = []string{repoWithTag}
		} else {
			digestsImgs[digest] = append(digestsImgs[digest], repoWithTag)
		}
	}
	return digestsImgs, nil
}

// user needs to provide a full path for docker hub like:
// index.docker.io/mondoolabs/mondoo
func (a *DockerRegistryImages) List() ([]*assets.Asset, error) {
	reg, err := name.NewRegistry("localhost:5000")
	if err != nil {
		return nil, err
	}

	fmt.Println(reg.RegistryStr())

	repos, err := a.Repositories(reg)
	if err != nil {
		return nil, err
	}

	digestsImgs := map[string][]string{}
	for i := range repos {
		fmt.Println(reg.RegistryStr() + "/" + repos[i])
		digests, err := a.Repository(reg.RegistryStr() + "/" + repos[i])
		if err != nil {
			return nil, err
		}
		for j := range digests {
			digestsImgs[j] = digests[j]
		}
	}

	imgs := []*assets.Asset{}
	for digest := range digestsImgs {

		asset := &assets.Asset{
			ReferenceIDs: []string{MondooContainerImageID(digest)},
			// Name:         strings.Join(dImg.RepoTags, ","),
			Platform: &assets.Platform{
				Kind:    assets.Kind_KIND_CONTAINER_IMAGE,
				Runtime: runtime.RUNTIME_DOCKER_REGISTRY,
			},
			Connections: []*assets.Connection{
				&assets.Connection{
					Backend: assets.ConnectionBackend_CONNECTION_DOCKER_IMAGE,
					Host:    reg.RegistryStr(),
				},
			},
			State:  assets.State_STATE_ONLINE,
			Labels: make(map[string]string),
		}

		// store digest
		asset.Labels["docker.io/digest"] = digest

		// store repo tags
		imageTags := []string{}
		for tag := range digestsImgs[digest] {
			imageTags = append(imageTags, digestsImgs[digest][tag])
		}
		asset.Labels["docker.io/tags"] = strings.Join(imageTags, ",")

		// store repo digest
		// NOTE: based on the current api, this case cannot happen
		// repoDigests := []string{repoURL + "@" + digest}
		// asset.Labels["docker.io/repo-digests"] = strings.Join(repoDigests, ",")

		imgs = append(imgs, asset)

	}

	return imgs, nil
}
