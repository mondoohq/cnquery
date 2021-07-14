package inventory

import (
	"context"
	"encoding/json"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/motor/asset"
	"go.mondoo.io/mondoo/motor/transports"
	"go.mondoo.io/mondoo/motor/vault"
)

type SecretManager interface {
	GetSecretMetadata(a *asset.Asset) (*CredentialQueryResponse, error)
	EnrichConnection(a *asset.Asset, secMeta *CredentialQueryResponse) error
}

func NewVaultSecretManager(v vault.Vault, secretMetadataQuery string) (SecretManager, error) {
	sq, err := NewCredentialQueryRunner(secretMetadataQuery)
	if err != nil {
		return nil, err
	}

	return &VaultSecretManager{
		vault: v,
		sq:    sq,
	}, nil
}

// VaultSecretManager
//
// 1. we will call the lumi runtime function to retrieve the secret metadata, to determine the correct values
//    asset metadata  (labels and platform and connection type (e.g. ssh) is passed in as a property
// 2. we use the secret metadata to retrieve the secret from vault
type VaultSecretManager struct {
	vault vault.Vault
	sq    *CredentialQueryRunner
}

func (vsm *VaultSecretManager) GetSecretMetadata(a *asset.Asset) (*CredentialQueryResponse, error) {
	if vsm.vault == nil {
		return nil, nil
	}

	// this is where we get the vault configuration query and evaluate it against the asset data
	// if vault and secret function is set, run the additional handling
	return vsm.sq.SecretId(a)
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

func (vsm *VaultSecretManager) EnrichConnection(a *asset.Asset, secretMetadata *CredentialQueryResponse) error {
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

	// add all secrets from second config
	if len(tc2.Credentials) > 0 {
		tc1.Credentials = append(tc1.Credentials, tc2.Credentials...)
	}
}

// parses the secret and its meta-information into a transport.Config object
func parseSecret(secretMetadata *CredentialQueryResponse, secret string) (*transports.TransportConfig, error) {
	tc := &transports.TransportConfig{}

	backendValue := ""

	switch secretMetadata.SecretFormat {
	case "", "password": // if no format was provided, we assume its password
		credential := transports.NewPasswordCredential(secretMetadata.User, secret)
		tc.AddCredential(credential)
	case "private_key":
		credential := transports.NewPrivateKeyCredential(secretMetadata.User, []byte(secret), nil)
		tc.AddCredential(credential)
	case "json":
		jsonSecret := make(map[string]string)
		err := json.Unmarshal([]byte(secret), &jsonSecret)
		if err != nil {
			return nil, err
		}

		if bkd, ok := jsonSecret["backend"]; ok {
			backendValue = bkd
		}

		if host, ok := jsonSecret["host"]; ok {
			tc.Host = host
		}

		user := ""
		if usr, ok := jsonSecret["user"]; ok {
			user = usr
		}

		if pwd, ok := jsonSecret["password"]; ok {
			credential := transports.NewPasswordCredential(user, pwd)
			tc.AddCredential(credential)
		}

		if privK, ok := jsonSecret["private_key"]; ok {
			credential := transports.NewPrivateKeyCredential(user, []byte(privK), nil)
			tc.AddCredential(credential)
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
	tc.Backend = backend

	return tc, nil
}
