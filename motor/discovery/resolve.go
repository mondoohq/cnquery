package discovery

import (
	"context"
	"regexp"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/apps/mondoo/cmd/options"
	"go.mondoo.io/mondoo/motor/asset"
	"go.mondoo.io/mondoo/motor/transports"
	"go.mondoo.io/mondoo/motor/vault"
)

var scheme = regexp.MustCompile(`^(.*?):\/\/(.*)$`)

type Resolver interface {
	Resolve(asset *options.VulnOptsAsset, opts *options.VulnOpts) ([]*asset.Asset, error)
}

var resolver map[string]Resolver

func init() {
	resolver = make(map[string]Resolver)
	resolver["local"] = &instanceResolver{}
	resolver["winrm"] = &instanceResolver{}
	resolver["ssh"] = &instanceResolver{}
	resolver["docker"] = &dockerResolver{}
	resolver["mock"] = &instanceResolver{}
	resolver["tar"] = &instanceResolver{}
	resolver["k8s"] = &k8sResolver{}
	resolver["gcr"] = &gcrResolver{}
	resolver["gcp"] = &gcpResolver{}
	resolver["cr"] = &containerRegistryResolver{}
	resolver["az"] = &azureResolver{}
	resolver["azure"] = &azureResolver{}
	resolver["aws"] = &awsResolver{}
	resolver["ec2"] = &awsResolver{}
	resolver["vagrant"] = &vagrantResolver{}
	resolver["mock"] = &mockResolver{}
	resolver["vsphere"] = &vsphereResolver{}
	resolver["vsphere+vm"] = &vmwareGuestResolver{}
	resolver["aristaeos"] = &instanceResolver{}
	resolver["ms365"] = &ms365Resolver{}
	resolver["ipmi"] = &ipmiResolver{}
}

type secret struct {
	connection string
	backend    string
	host       string
	user       string
	password   string
}

func Assets(opts *options.VulnOpts, v vault.Vault) ([]*asset.Asset, error) {
	resolved := []*asset.Asset{}

	getSecret := func(platformID string) (secret, error) {
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
			connection: creds.Fields["connection"],
			backend:    creds.Fields["backend"],
			host:       creds.Fields["host"],
			user:       creds.Fields["user"],
			password:   creds.Fields["password"],
		}, nil
	}

	for i := range opts.Assets {
		asset := opts.Assets[i]

		secret, err := getSecret(asset.ReferenceID)
		if err == nil {
			asset.Connection = secret.connection
			asset.Password = secret.password
		}

		m := scheme.FindStringSubmatch(asset.Connection)
		if len(m) < 3 {
			return nil, errors.New("unsupported connection string: " + asset.Connection)
		}

		resolverId := m[1]
		r, ok := resolver[resolverId]
		if !ok {
			return nil, errors.New("unsupported backend: " + resolverId)
		}

		resolverAssets, err := r.Resolve(asset, opts)
		if err != nil {
			return nil, err
		}

		for ai := range resolverAssets {
			asset := resolverAssets[ai]

			if v != nil {
				// iterate over each platform id
				for pid := range asset.ReferenceIDs {
					platformId := asset.ReferenceIDs[pid]
					secret, err := getSecret(platformId)
					if err == nil {
						// TODO: we should only overwrite the data that is available
						// ensure there are no duplicates
						if secret.backend == "ssh" {
							asset.Connections = append(asset.Connections, &transports.TransportConfig{
								Platformid: platformId,
								Backend:    transports.TransportBackend_CONNECTION_SSH,
								Host:       secret.host, // that should come from the
								User:       secret.user,
								Password:   secret.password,
								Port:       "",
								Insecure:   true,
							})
						}
					}
				}
			}

			resolved = append(resolved, asset)
		}
	}

	return resolved, nil
}
