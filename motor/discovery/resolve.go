package discovery

import (
	"context"
	"encoding/json"
	"regexp"
	"sort"

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
	"go.mondoo.io/mondoo/motor/platform"
	"go.mondoo.io/mondoo/motor/transports"
	"go.mondoo.io/mondoo/motor/vault"
)

var scheme = regexp.MustCompile(`^(.*?):\/\/(.*)$`)

type Resolver interface {
	Name() string
	ParseConnectionURL(url string, opts ...transports.TransportConfigOption) (*transports.TransportConfig, error)
	Resolve(t *transports.TransportConfig, opts map[string]string) ([]*asset.Asset, error)
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
		t := root.Connections[i]

		resolverId := t.Backend.Scheme()
		r, ok := resolver[resolverId]
		if !ok {
			return nil, errors.New("unsupported backend: " + resolverId)
		}

		resp, err := r.Resolve(t, root.Options)
		if err != nil {
			return nil, err
		}

		for ai := range resp {
			asset := resp[ai]

			// this is where we get the vault configuration query and evaluate it against the asset data
			secretInfo := resolveAssetToVaultConfiguration(&assetMatchInfo{labels: asset.GetLabels(), platform: asset.Platform})
			if secretInfo != nil {
				// if it does match a configuration, enrich asset with information from vault
				enrichAssetWithVaultData(v, asset, secretInfo)
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

func resolveAssetToVaultConfiguration(asset *assetMatchInfo) *secretInfo {
	// here is where we will call the lumi runtime function
	// give it the asset match information (labels and platform and connection type (e.g. ssh)) + the user-defined vault config query
	// it returns the secretinfo as defined in the vault config query

	// this is the go code that will be replaced by the lumi stuff
	configurations := []queryConfiguration{}
	config1 := queryConfiguration{
		User:         "ec2-user",
		MatchKey:     "secretsmanager",
		MatchValue:   "secret_id",
		SecretFormat: "private_key",
		Hierarchy:    1,
	}
	configurations = append(configurations, config1)
	config2 := queryConfiguration{
		User:          "ubuntu",
		MatchPlatform: "ubuntu",
		SecretId:      "arn:aws:secretsmanager:us-east-2:921877552404:secret:test2-lTHSUJ",
		SecretFormat:  "private_key",
		Hierarchy:     2,
	}
	configurations = append(configurations, config2)

	config3 := queryConfiguration{
		User:         "ec2-user",
		MatchKey:     "env",
		MatchValue:   "test",
		SecretId:     "arn:aws:secretsmanager:us-east-2:921877552404:secret:test3-pK8sjF",
		SecretFormat: "private_key",
		Hierarchy:    3,
	}
	configurations = append(configurations, config3)

	sort.SliceStable(configurations, func(i, j int) bool {
		return configurations[i].Hierarchy < configurations[j].Hierarchy
	})
	for i := range configurations {
		configuration := configurations[i]
		if configuration.MatchPlatform != "" {
			// for now, assuming platform == platform name - should be extended
			if asset.platform.Name == configuration.MatchPlatform {
				return &secretInfo{
					user:           configuration.User,
					secretID:       configuration.SecretId,
					secretFormat:   configuration.SecretFormat,
					jsonFields:     configuration.JsonFields,
					connectionType: "winrm",
				}
			}
		}
		if configuration.MatchKey != "" {
			val := asset.labels[configuration.MatchKey]
			if len(val) > 0 {
				if configuration.MatchValue == "secret_id" {
					// user has specified "secret_id" keyword, telling us to look at the value of the tag for the secret id
					return &secretInfo{
						user:           configuration.User,
						secretID:       val,
						secretFormat:   configuration.SecretFormat,
						jsonFields:     configuration.JsonFields,
						connectionType: "ssh",
					}
				}
				// user has specified a match value
				if configuration.MatchValue != "" {
					if val == configuration.MatchValue {
						return &secretInfo{
							user:           configuration.User,
							secretID:       configuration.SecretId,
							secretFormat:   configuration.SecretFormat,
							jsonFields:     configuration.JsonFields,
							connectionType: "ssh",
						}
					}
				}
				// user specificied match key but no value, match
				return &secretInfo{
					user:           configuration.User,
					secretID:       configuration.SecretId,
					secretFormat:   configuration.SecretFormat,
					jsonFields:     configuration.JsonFields,
					connectionType: "ssh",
				}
			}
		}
	}
	// end code block
	return nil
}
