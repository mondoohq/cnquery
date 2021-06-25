package container_registry

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/motor/asset"
	"go.mondoo.io/mondoo/motor/discovery/docker_engine"
	"go.mondoo.io/mondoo/motor/platform"

	"go.mondoo.io/mondoo/motor/transports"
)

func NewContainerRegistry() *DockerRegistryImages {
	return &DockerRegistryImages{}
}

type DockerRegistryImages struct {
	Insecure bool
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

	// fetch tags
	tags, err := remote.List(repo, a.remoteOptions()...)

	for i := range tags {
		repoWithTag := repo.Name() + ":" + tags[i]

		ref, err := name.ParseReference(repoWithTag)
		if err != nil {
			return nil, fmt.Errorf("parsing reference %q: %v", repoWithTag, err)
		}

		a, err := a.toAsset(ref)
		if err != nil {
			return nil, err
		}
		assets = append(assets, a)
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

func (a *DockerRegistryImages) GetImage(ref name.Reference) (*asset.Asset, error) {
	return a.toAsset(ref)
}

func (a *DockerRegistryImages) toAsset(ref name.Reference) (*asset.Asset, error) {
	desc, err := remote.Get(ref, a.remoteOptions()...)
	if err != nil {
		return nil, err
	}
	imgDigest := desc.Digest.String()
	repoName := ref.Name()
	imageUrl := repoName + "@" + imgDigest
	asset := &asset.Asset{
		PlatformIds: []string{docker_engine.MondooContainerImageID(imgDigest)},
		Name:        docker_engine.ShortContainerImageID(imgDigest),
		Platform: &platform.Platform{
			Kind:    transports.Kind_KIND_CONTAINER_IMAGE,
			Runtime: transports.RUNTIME_DOCKER_REGISTRY,
		},
		Connections: []*transports.TransportConfig{
			{
				Backend: transports.TransportBackend_CONNECTION_CONTAINER_REGISTRY,
				Host:    imageUrl,
			},
		},
		State:  asset.State_STATE_ONLINE,
		Labels: make(map[string]string),
	}

	// store digest
	asset.Labels["docker.io/digest"] = imgDigest

	// store repo tags
	// asset.Labels["docker.io/tags"] = strings.Join(tags, ",")

	log.Debug().Strs("platform-ids", asset.PlatformIds).Msg("asset platform ids")

	// store repo digest
	// NOTE: based on the current api, this case cannot happen
	// repoDigests := []string{repoURL + "@" + digest}
	// asset.Labels["docker.io/repo-digests"] = strings.Join(repoDigests, ",")
	return asset, nil
}

func ShortContainerImageID(id string) string {
	id = strings.Replace(id, "sha256:", "", -1)
	if len(id) > 12 {
		return id[0:12]
	}
	return id
}
