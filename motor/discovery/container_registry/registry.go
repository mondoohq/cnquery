package container_registry

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/google/go-containerregistry/pkg/authn"

	"errors"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/motor/asset"
	"go.mondoo.com/cnquery/motor/motorid/containerid"
	"go.mondoo.com/cnquery/motor/platform"
	"go.mondoo.com/cnquery/motor/providers"
	"go.mondoo.com/cnquery/motor/providers/container/auth"
	"go.mondoo.com/cnquery/motor/providers/container/image"
	"go.mondoo.com/cnquery/motor/vault"
)

func NewContainerRegistryResolver() *DockerRegistryImages {
	return &DockerRegistryImages{}
}

type DockerRegistryImages struct {
	Insecure            bool
	DisableKeychainAuth bool
}

func (a *DockerRegistryImages) remoteOptions() []remote.Option {
	options := []remote.Option{}

	// does not work with bearer auth, therefore it need to be disabled when other remote auth options are used
	// TODO: we should implement this a bit differently
	if a.DisableKeychainAuth == false {
		options = append(options, remote.WithAuthFromKeychain(authn.DefaultKeychain))
	}

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
		return nil, errors.Join(err, errors.New("resolve registry"))
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
// index.docker.io/mondoo
// index.docker.io/mondoo/client
// harbor.lunalectric.com/library
// harbor.lunalectric.com/library/ubuntu
func (a *DockerRegistryImages) ListRepository(repoName string) ([]*asset.Asset, error) {
	assets := []*asset.Asset{}

	repo, err := name.NewRepository(repoName)
	if err != nil {
		return nil, err
	}

	// fetch tags
	tags, err := remote.List(repo, a.remoteOptions()...)
	if err != nil {
		return nil, handleUnauthorizedError(err, repo.Name())
	}

	foundAssets := map[string]*asset.Asset{}
	for i := range tags {
		repoWithTag := repo.Name() + ":" + tags[i]

		ref, err := name.ParseReference(repoWithTag)
		if err != nil {
			return nil, fmt.Errorf("parsing reference %q: %v", repoWithTag, err)
		}

		a, err := a.toAsset(ref, nil)
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

func (a *DockerRegistryImages) GetImage(ref name.Reference, creds []*vault.Credential, opts ...remote.Option) (*asset.Asset, error) {
	return a.toAsset(ref, creds, opts...)
}

func (a *DockerRegistryImages) toAsset(ref name.Reference, creds []*vault.Credential, opts ...remote.Option) (*asset.Asset, error) {
	desc, err := image.GetImageDescriptor(ref, auth.AuthOption(creds)...)
	if err != nil {
		return nil, handleUnauthorizedError(err, ref.Name())
	}
	imgDigest := desc.Digest.String()
	repoName := ref.Context().Name()
	imgTag := ref.Context().Tag(ref.Identifier()).TagStr()
	name := repoName + "@" + containerid.ShortContainerImageID(imgDigest)
	imageUrl := repoName + "@" + imgDigest
	asset := &asset.Asset{
		PlatformIds: []string{containerid.MondooContainerImageID(imgDigest)},
		Name:        name,
		Platform: &platform.Platform{
			Kind:    providers.Kind_KIND_CONTAINER_IMAGE,
			Runtime: providers.RUNTIME_DOCKER_REGISTRY,
		},
		Connections: []*providers.Config{
			{
				Backend:     providers.ProviderType_CONTAINER_REGISTRY,
				Host:        imageUrl,
				Credentials: creds,
			},
		},
		State:  asset.State_STATE_ONLINE,
		Labels: make(map[string]string),
	}

	// store digest and tag
	asset.Labels["docker.io/digest"] = imgDigest
	asset.Labels["docker.io/tags"] = imgTag
	log.Debug().Strs("platform-ids", asset.PlatformIds).Msg("asset platform ids")
	return asset, nil
}
