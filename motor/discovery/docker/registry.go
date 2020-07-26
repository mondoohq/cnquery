package docker

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/motor/asset"

	"go.mondoo.io/mondoo/motor/transports"
)

func NewDockerRegistryImages() *DockerRegistryImages {
	return &DockerRegistryImages{}
}

type DockerRegistryImages struct {
	Insecure bool
}

func (a *DockerRegistryImages) Repositories(reg name.Registry) ([]string, error) {
	n := 100
	last := ""
	var res []string
	for {
		page, err := remote.CatalogPage(reg, last, n, a.remoteOptions()...)
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

	desc, err := remote.Get(ref, a.remoteOptions()...)
	if err != nil {
		return "", err
	}
	return desc.Digest.String(), nil
}

func (a *DockerRegistryImages) remoteOptions() []remote.Option {
	options := []remote.Option{}
	options = append(options, remote.WithAuthFromKeychain(authn.DefaultKeychain))

	if a.Insecure {
		// NOTE: config to get remote running with an insecure registry, we need to override the TLSClientConfig
		tr := &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
				DualStack: true,
			}).DialContext,
			ForceAttemptHTTP2:     true,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		}
		options = append(options, remote.WithTransport(tr))
	}

	return options
}

func (a *DockerRegistryImages) Tags(repo name.Repository) ([]string, error) {
	return remote.List(repo, a.remoteOptions()...)
}

// Repository reads information about a specific repo and returns its entry digests with related tags
func (a *DockerRegistryImages) Repository(repo name.Repository) (map[string][]string, error) {
	tags, err := a.Tags(repo)
	if err != nil {
		return nil, err
	}

	digestsImgs := map[string][]string{}

	for i := range tags {
		repoWithTag := repo.Name() + ":" + tags[i]
		digest, err := a.Digest(repoWithTag)
		log.Debug().Str("repo", repo.Name()).Str("tag", tags[i]).Msg("discovered image with tag")
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

// ListRegistry tries to iterate over all repositores in one registry
// eg. 1234567.dkr.ecr.us-east-1.amazonaws.com
func (a *DockerRegistryImages) ListRegistry(registry string) ([]*asset.Asset, error) {
	reg, err := name.NewRegistry(registry)
	if err != nil {
		return nil, errors.Wrap(err, "resolve registry")
	}

	repos, err := a.Repositories(reg)
	if err != nil {
		return nil, err
	}

	assets := []*asset.Asset{}
	for i := range repos {
		repoName := reg.RegistryStr() + "/" + repos[i]
		log.Debug().Str("repository", repoName).Msg("discovered repository")

		repo, err := name.NewRepository(repoName)
		if err != nil {
			return nil, err
		}

		digests, err := a.Repository(repo)
		if err != nil {
			return nil, err
		}
		for imgDigest := range digests {
			tags := digests[imgDigest]
			assets = append(assets, a.toAsset(repoName, imgDigest, tags))
		}
	}

	return assets, nil
}

// ListRepository tries to fetch all details about a specific repository
// index.docker.io/mondoolabs
// index.docker.io/mondoolabs/mondoo
// harbor.yourdomain.com/library
// harbor.yourdomain.com/library/ubuntu
func (a *DockerRegistryImages) ListRepository(repoName string) ([]*asset.Asset, error) {
	assets := []*asset.Asset{}

	repo, err := name.NewRepository(repoName)
	if err != nil {
		return nil, err
	}

	digests, err := a.Repository(repo)
	if err != nil {
		return nil, err
	}
	for imgDigest := range digests {
		tags := digests[imgDigest]
		assets = append(assets, a.toAsset(repoName, imgDigest, tags))
	}
	return assets, nil
}

// ListImages either takes a registry or a repository and tries to fetch as many images as possible
func (a *DockerRegistryImages) ListImages(repoName string) ([]*asset.Asset, error) {
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

func (a *DockerRegistryImages) toAsset(repoName string, imgDigest string, tags []string) *asset.Asset {
	imageUrl := repoName + "@" + imgDigest
	asset := &asset.Asset{
		ReferenceIDs: []string{MondooContainerImageID(imgDigest)},
		Name:         ShortContainerImageID(imgDigest),
		Kind:         asset.Kind_KIND_CONTAINER_IMAGE,
		Runtime:      asset.RUNTIME_DOCKER_REGISTRY,
		Connections: []*transports.TransportConfig{
			&transports.TransportConfig{
				Backend: transports.TransportBackend_CONNECTION_DOCKER_IMAGE,
				Host:    imageUrl,
			},
		},
		State:  asset.State_STATE_ONLINE,
		Labels: make(map[string]string),
	}

	// store digest
	asset.Labels["docker.io/digest"] = imgDigest

	// store repo tags
	asset.Labels["docker.io/tags"] = strings.Join(tags, ",")

	// store repo digest
	// NOTE: based on the current api, this case cannot happen
	// repoDigests := []string{repoURL + "@" + digest}
	// asset.Labels["docker.io/repo-digests"] = strings.Join(repoDigests, ",")
	return asset
}

func ShortContainerImageID(id string) string {
	id = strings.Replace(id, "sha256:", "", -1)
	if len(id) > 12 {
		return id[0:12]
	}
	return id
}
