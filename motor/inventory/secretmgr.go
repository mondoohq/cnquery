package inventory

import (
	"context"
	"encoding/json"

	"go.mondoo.io/mondoo/llx"
	"go.mondoo.io/mondoo/motor/vault"
	"go.mondoo.io/mondoo/policy/executor"

	"github.com/cockroachdb/errors"
	"github.com/mitchellh/mapstructure"
	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/motor/asset"
	"go.mondoo.io/mondoo/motor/transports"
	"go.mondoo.io/mondoo/types"
)

type SecretMetadata struct {
	Backend      string `json:"backend,omitempty"`      // default to ssh, user specified
	User         string `json:"user,omitempty"`         // user associated with the secret
	Host         string `json:"host,omitempty"`         // overwrite of the host
	SecretID     string `json:"secretID,omitempty"`     // id to use to fetch the secret from the source vault
	SecretFormat string `json:"secretFormat,omitempty"` // private_key, password, or json
}

type SecretManager interface {
	GetSecretMetadata(a *asset.Asset) (*SecretMetadata, error)
	EnrichConnection(a *asset.Asset, secMeta *SecretMetadata) error
}

func NewVaultSecretManager(v vault.Vault, secretMetadataQuery string) (SecretManager, error) {
	e, err := executor.NewEmbeddedExecutor()
	if err != nil {
		return nil, err
	}

	// just empty props to ensure we can compile
	props := map[string]*llx.Primitive{
		"mrn":      llx.StringPrimitive(""),
		"name":     llx.StringPrimitive(""),
		"labels":   llx.MapData(map[string]interface{}{}, types.String).Result().Data,
		"platform": llx.MapData(map[string]interface{}{}, types.String).Result().Data,
	}
	_, err = e.Compile(secretMetadataQuery, props)
	if err != nil {
		return nil, errors.Wrap(err, "could not compile the secret metadata function")
	}

	return &VaultSecretManager{
		e:                   e,
		vault:               v,
		secretMetadataQuery: secretMetadataQuery,
	}, nil
}

// VaultSecretManager
//
// 1. we will call the lumi runtime function to retrieve the secret metadata, to determine the correct values
//    asset metadata  (labels and platform and connection type (e.g. ssh) is passed in as a property
// 2. we use the secret metadata to retrieve the secret from vault
type VaultSecretManager struct {
	e                   *executor.EmbeddedExecutor
	vault               vault.Vault
	secretMetadataQuery string
}

func (vsm *VaultSecretManager) GetSecretMetadata(a *asset.Asset) (*SecretMetadata, error) {
	if vsm.vault == nil {
		return nil, nil
	}

	// this is where we get the vault configuration query and evaluate it against the asset data
	// if vault and secret function is set, run the additional handling

	// map labels to props
	labelProps := map[string]interface{}{}
	labels := a.GetLabels()
	for k, v := range labels {
		labelProps[k] = v
	}

	// map platform to props
	var platformProps map[string]interface{}
	if a.Platform != nil {
		platformProps = map[string]interface{}{
			"name":    a.Platform.Name,
			"release": a.Platform.Release,
			"arch":    a.Platform.Arch,
		}
	} else {
		platformProps = map[string]interface{}{}
	}

	props := map[string]*llx.Primitive{
		"mrn":      llx.StringPrimitive(a.Mrn),
		"name":     llx.StringPrimitive(a.Name),
		"labels":   llx.MapData(labelProps, types.String).Result().Data,
		"platform": llx.MapData(platformProps, types.String).Result().Data,
	}

	value, err := vsm.e.Run(vsm.secretMetadataQuery, props)
	if err != nil {
		return nil, err
	}

	sMeta := &SecretMetadata{}
	decoder, _ := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Metadata: nil,
		Result:   sMeta,
		TagName:  "json",
	})
	err = decoder.Decode(value)

	return sMeta, err
}

func (vsm *VaultSecretManager) GetSecret(keyID string) (string, error) {
	cred, err := vsm.vault.Get(context.Background(), &vault.SecretID{
		Key: keyID,
	})
	if err != nil {
		log.Error().Msgf("unable to retrieve secret id: %s", keyID)
		return "", err
	}
	if cred == nil {
		return "", errors.New("could not find the id: " + keyID)
	}
	log.Debug().Str("key-id", keyID).Msg("retrieved secret")
	return string(cred.Secret), nil
}

func (vsm *VaultSecretManager) EnrichConnection(a *asset.Asset, secretMetadata *SecretMetadata) error {
	if vsm.vault == nil || a == nil || secretMetadata == nil {
		return nil
	}
	log.Debug().Str("key-id", secretMetadata.SecretID).Str("format", secretMetadata.SecretFormat).Str("ssh", secretMetadata.Backend).Str("host", secretMetadata.Host).Str("user", secretMetadata.User).Msg("use secret for asset")
	secret, err := vsm.GetSecret(secretMetadata.SecretID)
	if err != nil {
		return err
	}

	if len(secret) == 0 {
		return nil
	}

	// parses the secret into a connection object
	secretConnection, err := parseSecret(secretMetadata, secret)
	if err != nil {
		return err
	}

	log.Info().Str("secret", secretMetadata.SecretID).Msg("use secret from vault for asset")
	// merge the data but relatively smart, if the backend connection was found, enrich the existing since it
	// most-likely does not make sense to have 1+ ssh connections
	found := false
	for i := range a.Connections {
		conn := a.Connections[i]
		if conn.Backend == secretConnection.Backend {
			// merge the connection object values
			mergeConnectionValues(conn, secretConnection)
			found = true
			break
		}
	}

	// if nothing was found, create a new connection at asset
	if !found {
		a.Connections = append(a.Connections, secretConnection)
	}

	return nil
}

// mergeConnection merges the values of tc2 into tc1
func mergeConnectionValues(tc1 *transports.TransportConfig, tc2 *transports.TransportConfig) {
	if tc1 == nil {
		panic("cannot merge connections. you cannot merge a nil connection")
	}
	if tc2 == nil {
		return
	}

	if tc2.Host != "" {
		tc1.Host = tc2.Host
	}

	if tc2.User != "" {
		tc1.User = tc2.User
	}

	if tc2.Password != "" {
		tc1.Password = tc2.Password
	}

	if len(tc2.PrivateKeyBytes) > 0 {
		tc1.PrivateKeyBytes = tc2.PrivateKeyBytes
	}
}

// parses the secret and its meta-information into a transport.Config object
func parseSecret(secretMetadata *SecretMetadata, secret string) (*transports.TransportConfig, error) {
	connection := &transports.TransportConfig{}

	backendValue := ""

	if secretMetadata.User != "" {
		connection.User = secretMetadata.User
	}

	switch secretMetadata.SecretFormat {
	case "", "password": // if no format was provided, we assume its password
		connection.Password = secret
	case "private_key":
		connection.PrivateKeyBytes = []byte(secret)
	case "json":
		jsonSecret := make(map[string]string)
		err := json.Unmarshal([]byte(secret), &jsonSecret)
		if err != nil {
			return nil, err
		}

		if bkd, ok := jsonSecret["backend"]; ok {
			backendValue = bkd
		}

		if usr, ok := jsonSecret["user"]; ok {
			connection.User = usr
		}

		if host, ok := jsonSecret["host"]; ok {
			connection.Host = host
		}

		if pwd, ok := jsonSecret["password"]; ok {
			connection.Password = pwd
		}

		if privK, ok := jsonSecret["private_key"]; ok {
			connection.PrivateKeyBytes = []byte(privK)
		}
	default:
		return nil, errors.New("unsupported secret format " + secretMetadata.SecretFormat + " requested")
	}

	// metadata always overwrite the secret information
	if secretMetadata.Backend != "" {
		backendValue = secretMetadata.Backend
	}

	backend, err := transports.MapSchemeBackend(backendValue)
	if err != nil {
		return nil, err
	}
	connection.Backend = backend

	return connection, nil
}
