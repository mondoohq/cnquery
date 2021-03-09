package discovery

import (
	"context"
	"encoding/json"

	"go.mondoo.io/mondoo/stringx"
	"regexp"
	"strings"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/motor/asset"
	"go.mondoo.io/mondoo/motor/discovery/aws"
	"go.mondoo.io/mondoo/motor/discovery/azure"
	"go.mondoo.io/mondoo/motor/discovery/container_registry"
	"go.mondoo.io/mondoo/motor/discovery/docker_engine"
	"go.mondoo.io/mondoo/motor/discovery/equinix"
	"go.mondoo.io/mondoo/motor/discovery/gcp"
	"go.mondoo.io/mondoo/motor/discovery/instance"
	"go.mondoo.io/mondoo/motor/discovery/ipmi"
	"go.mondoo.io/mondoo/motor/discovery/k8s"
	"go.mondoo.io/mondoo/motor/discovery/local"
	"go.mondoo.io/mondoo/motor/discovery/mock"
	"go.mondoo.io/mondoo/motor/discovery/ms365"
	"go.mondoo.io/mondoo/motor/discovery/vagrant"
	"go.mondoo.io/mondoo/motor/discovery/vsphere"
	"go.mondoo.io/mondoo/motor/platform"
	"go.mondoo.io/mondoo/motor/transports"
	"go.mondoo.io/mondoo/motor/vault"
)

var scheme = regexp.MustCompile(`^(.*?):\/\/(.*)$`)

type Resolver interface {
	Name() string
	ParseConnectionURL(url string, opts ...transports.TransportConfigOption) (*transports.TransportConfig, error)
	Resolve(t *transports.TransportConfig) ([]*asset.Asset, error)
	AvailableDiscoveryTargets() []string
}

var resolver map[string]Resolver

func init() {
	resolver = map[string]Resolver{
		transports.SCHEME_LOCAL:              &local.Resolver{},
		transports.SCHEME_WINRM:              &instance.Resolver{},
		transports.SCHEME_SSH:                &instance.Resolver{},
		transports.SCHEME_DOCKER:             &docker_engine.Resolver{},
		transports.SCHEME_DOCKER_IMAGE:       &docker_engine.Resolver{},
		transports.SCHEME_DOCKER_CONTAINER:   &docker_engine.Resolver{},
		transports.SCHEME_DOCKER_TAR:         &docker_engine.Resolver{},
		transports.SCHEME_TAR:                &instance.Resolver{},
		transports.SCHEME_K8S:                &k8s.Resolver{},
		transports.SCHEME_GCR:                &gcp.GcrResolver{},
		transports.SCHEME_GCP:                &gcp.GcpResolver{},
		transports.SCHEME_CONTAINER_REGISTRY: &container_registry.Resolver{},
		transports.SCHEME_AZURE:              &azure.Resolver{},
		transports.SCHEME_AWS:                &aws.Resolver{},
		transports.SCHEME_VAGRANT:            &vagrant.Resolver{},
		transports.SCHEME_MOCK:               &mock.Resolver{},
		transports.SCHEME_VSPHERE:            &vsphere.Resolver{},
		transports.SCHEME_VSPHERE_VM:         &vsphere.VMGuestResolver{},
		transports.SCHEME_ARISTA:             &instance.Resolver{},
		transports.SCHEME_MS365:              &ms365.Resolver{},
		transports.SCHEME_IPMI:               &ipmi.Resolver{},
		transports.SCHEME_FS:                 &instance.Resolver{},
		transports.SCHEME_EQUINIX:            &equinix.Resolver{},
	}
}

func getSecret(v vault.Vault, keyID string) (string, error) {
	log.Info().Str("key-id", keyID).Msg("get secret")
	cred, err := v.Get(context.Background(), &vault.CredentialID{
		Key: keyID,
	})
	if err != nil || cred == nil {
		log.Info().Msg("could not find the id")
		return "", err
	}
	return cred.Secret, nil
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

func enrichAssetWithVaultData(v vault.Vault, a *asset.Asset, secretInfo *secretInfo) {
	if v == nil || secretInfo == nil {
		return
	}
	secret, err := getSecret(v, secretInfo.secretID)
	if len(secret) > 0 {
		for i := range a.Connections {
			connection := a.Connections[i]
			if secretInfo.connectionType == "winrm" {
				connection.Backend = transports.TransportBackend_CONNECTION_WINRM
			}
			if secretInfo.user != "" {
				connection.User = secretInfo.user
			}
			switch secretInfo.secretFormat {
			case "private_key":
				connection.PrivateKeyBytes = []byte(secret)
			case "password":
				connection.Password = secret
			case "json":
				err = parseJsonByFields([]byte(secret), secretInfo, connection)
				if err != nil {
					log.Error().Msgf("unable to parse json secret for %v", secretInfo)
				}
			default:
				log.Error().Msgf("unsupported secret format %s requested", secretInfo.secretFormat)
			}

		}
	}

	return
}

func parseJsonByFields(secret []byte, secretInfo *secretInfo, connection *transports.TransportConfig) error {
	if secretInfo.secretFormat != "json" || len(secretInfo.jsonFields) == 0 {
		return errors.New("invalid configuration")
	}
	jsonSecret := make(map[string]string)
	err := json.Unmarshal(secret, &jsonSecret)
	if err != nil {
		return err
	}
	for i := range secretInfo.jsonFields {
		jsonField := secretInfo.jsonFields[i]
		switch jsonField {
		case "user":
			connection.User = jsonSecret["user"]
		case "password":
			connection.Password = jsonSecret["password"]
		case "private_key":
			connection.PrivateKeyBytes = []byte(jsonSecret["private_key"])
		}
	}
	return nil
}

func ResolveAsset(root *asset.Asset, v vault.Vault) ([]*asset.Asset, error) {
	resolved := []*asset.Asset{}

	for i := range root.Connections {
		tc := root.Connections[i]

		resolverId := tc.Backend.Scheme()
		r, ok := resolver[resolverId]
		if !ok {
			return nil, errors.New("unsupported backend: " + resolverId)
		}

		log.Debug().Str("resolver", r.Name()).Msg("run resolver")

		// check that all discovery options are supported and show a user warning
		availableTargets := r.AvailableDiscoveryTargets()
		for i := range tc.Discover.Targets {
			target := tc.Discover.Targets[i]
			if !stringx.Contains(availableTargets, target) {
				log.Warn().Str("resolver", r.Name()).Msgf("resolver does not support discovery target '%s', the following are supported: %s", target, strings.Join(availableTargets, ","))
			}
		}

		// resolve assets
		resolvedAssets, err := r.Resolve(tc)
		if err != nil {
			return nil, err
		}

		for ai := range resolvedAssets {
			asset := resolvedAssets[ai]

			// if vault and secret function is set, run the additional handling
			if v != nil {
				// this is where we get the vault configuration query and evaluate it against the asset data
				secretInfo := getAssetSecretInfo(&assetMatchInfo{labels: asset.GetLabels(), platform: asset.Platform})
				if secretInfo != nil {
					// if it does match a configuration, enrich asset with information from vault
					enrichAssetWithVaultData(v, asset, secretInfo)
				}
			}

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

type assetMatchInfo struct {
	labels   map[string]string
	platform *platform.Platform
}

type secretInfo struct {
	user           string   // user associated with the secret
	secretID       string   // id to use to fetch the secret from the source vault
	secretFormat   string   // private_key, password, or json
	jsonFields     []string // only for json, the fields we should desconstruct the json object into. all fields and values assumed to be of string type.
	connectionType string   // default to ssh, user specified
}

// just for now, this goes away when we are using the query with lumi
type queryConfiguration struct {
	MatchKey      string
	MatchValue    string
	SecretId      string
	User          string
	SecretFormat  string
	JsonFields    []string
	MatchPlatform string
	Hierarchy     int
}

func getAssetSecretInfo(asset *assetMatchInfo) *secretInfo {
	// here is where we will call the lumi runtime function
	// give it the asset match information (labels and platform and connection type (e.g. ssh)) + the user-defined vault config query
	// it returns the secretinfo as defined in the vault config query
	return nil
}
