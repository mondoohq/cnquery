package discovery

import (
	"context"
	"regexp"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/motor/asset"
	"go.mondoo.io/mondoo/motor/discovery/aws"
	"go.mondoo.io/mondoo/motor/discovery/azure"
	"go.mondoo.io/mondoo/motor/discovery/container_registry"
	"go.mondoo.io/mondoo/motor/discovery/docker_engine"
	"go.mondoo.io/mondoo/motor/discovery/gcp"
	"go.mondoo.io/mondoo/motor/discovery/instance"
	"go.mondoo.io/mondoo/motor/discovery/ipmi"
	"go.mondoo.io/mondoo/motor/discovery/k8s"
	"go.mondoo.io/mondoo/motor/discovery/mock"
	"go.mondoo.io/mondoo/motor/discovery/ms365"
	"go.mondoo.io/mondoo/motor/discovery/vagrant"
	"go.mondoo.io/mondoo/motor/discovery/vsphere"
	"go.mondoo.io/mondoo/motor/transports"
	"go.mondoo.io/mondoo/motor/vault"
)

var scheme = regexp.MustCompile(`^(.*?):\/\/(.*)$`)

type Resolver interface {
	Name() string
	ParseConnectionURL(url string, opts ...transports.TransportConfigOption) (*transports.TransportConfig, error)
	Resolve(t *transports.TransportConfig) ([]*asset.Asset, error)
}

var resolver map[string]Resolver

func init() {
	resolver = make(map[string]Resolver)
	resolver["local"] = &instance.Resolver{}
	resolver["winrm"] = &instance.Resolver{}
	resolver["ssh"] = &instance.Resolver{}
	resolver["docker"] = &docker_engine.Resolver{}
	resolver["docker+image"] = &docker_engine.Resolver{}
	resolver["mock"] = &instance.Resolver{}
	resolver["tar"] = &instance.Resolver{}
	resolver["k8s"] = &k8s.Resolver{}
	resolver["gcr"] = &gcp.GcrResolver{}
	resolver["gcp"] = &gcp.GcpResolver{}
	resolver["cr"] = &container_registry.Resolver{}
	resolver["az"] = &azure.Resolver{}
	resolver["azure"] = &azure.Resolver{}
	resolver["aws"] = &aws.Resolver{}
	resolver["ec2"] = &aws.Resolver{}
	resolver["vagrant"] = &vagrant.Resolver{}
	resolver["mock"] = &mock.Resolver{}
	resolver["vsphere"] = &vsphere.Resolver{}
	resolver["vsphere+vm"] = &vsphere.VMGuestResolver{}
	resolver["aristaeos"] = &instance.Resolver{}
	resolver["ms365"] = &ms365.Resolver{}
	resolver["ipmi"] = &ipmi.Resolver{}
}

type secret struct {
	backend  string
	host     string
	user     string
	password string
}

func getSecret(v vault.Vault, platformID string) (secret, error) {
	log.Info().Str("platform-id", platformID).Msg("get secret")
	ctx := context.Background()
	creds, err := v.Get(ctx, &vault.CredentialID{
		Key: vault.Mrn2secretKey(platformID),
	})
	if err != nil {
		log.Info().Msg("could not find the id")
		return secret{}, err
	}
	log.Info().Msgf("%v", creds)
	return secret{
		backend:  creds.Fields["backend"],
		host:     creds.Fields["host"],
		user:     creds.Fields["user"],
		password: creds.Fields["password"],
	}, nil
}

func ParseConnectionURL(url string, opts ...transports.TransportConfigOption) (*transports.TransportConfig, error) {
	m := scheme.FindStringSubmatch(url)
	if len(m) < 3 {
		return nil, errors.New("unsupported connection string: " + url)
	}
	resolverId := m[1]
	r, ok := resolver[resolverId]
	if !ok {
		return nil, errors.New("unsupported backend: " + resolverId)
	}
	log.Debug().Str("resolver", r.Name()).Msg("parse url")
	return r.ParseConnectionURL(url, opts...)
}

func enrichAssetInformation(v vault.Vault, a *asset.Asset) *asset.Asset {
	if v == nil {
		return a
	}

	// iterate over each platform id
	for pid := range a.ReferenceIDs {
		platformId := a.ReferenceIDs[pid]
		secret, err := getSecret(v, platformId)
		if err == nil {
			// TODO: we should only overwrite the data that is available
			// ensure there are no duplicates
			b, err := transports.MapSchemeBackend(secret.backend)
			if err != nil {
				log.Warn().Err(err).Msg("backend missing for " + platformId)
				continue
			}
			a.Connections = append(a.Connections, &transports.TransportConfig{
				Platformid: platformId,
				Backend:    b,
				Host:       secret.host, // that should come from the
				User:       secret.user,
				Password:   secret.password,
				Port:       "",
				Insecure:   true,
			})
		}
	}

	return a

}

func ResolveAsset(root *asset.Asset, v vault.Vault) ([]*asset.Asset, error) {
	resolved := []*asset.Asset{}

	// enrich asset with information from vault
	root = enrichAssetInformation(v, root)

	for i := range root.Connections {
		t := root.Connections[i]

		resolverId := t.Backend.Scheme()
		r, ok := resolver[resolverId]
		if !ok {
			return nil, errors.New("unsupported backend: " + resolverId)
		}

		resp, err := r.Resolve(t)
		if err != nil {
			return nil, err
		}

		for ai := range resp {
			asset := resp[ai]
			// enrich asset with information from vault
			asset = enrichAssetInformation(v, asset)

			resolved = append(resolved, asset)
		}
	}
	return resolved, nil
}

func ResolveAssets(rootAssets []*asset.Asset, v vault.Vault) ([]*asset.Asset, error) {
	resolved := []*asset.Asset{}

	for i := range rootAssets {
		asset := rootAssets[i]

		resolverAssets, err := ResolveAsset(asset, v)
		if err != nil {
			return nil, err
		}

		resolved = append(resolved, resolverAssets...)
	}

	return resolved, nil
}
