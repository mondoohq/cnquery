// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"errors"
	"net/http"
	"os"
	"strings"
	"time"

	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	mondoovault "go.mondoo.com/mql/v13/providers-sdk/v1/vault"
)

const (
	// DefaultAPIBaseURL is the root of the Confluent Cloud management API.
	DefaultAPIBaseURL = "https://api.confluent.cloud"

	// requestTimeout bounds a single API call.
	requestTimeout = 60 * time.Second

	// OptionAPIKey names the Cloud API key. The matching secret arrives as a
	// credential rather than an option, since it is the secret half.
	OptionAPIKey = "api-key"
	// OptionKafkaAPIKey names a cluster-scoped Kafka API key used for the
	// per-cluster REST endpoints (topics and ACLs). The Cloud API key is not
	// accepted there.
	OptionKafkaAPIKey = "kafka-api-key"
	// OptionBaseURL overrides the management API root. It exists for testing
	// and for Confluent deployments served from a different domain.
	OptionBaseURL = "base-url"

	// credentialUserKafka tags the Kafka API secret so it can be told apart
	// from the Cloud API secret, which arrives as a credential of the same
	// type.
	credentialUserKafka = "kafka-api-secret"

	// EnvAPIKey and EnvAPISecret carry the Cloud API key pair.
	EnvAPIKey    = "CONFLUENT_CLOUD_API_KEY"
	EnvAPISecret = "CONFLUENT_CLOUD_API_SECRET"

	// EnvKafkaAPIKey and EnvKafkaAPISecret carry a cluster-scoped Kafka API key
	// pair that applies to every cluster. A per-cluster pair overrides it, see
	// KafkaCredentialsFor.
	EnvKafkaAPIKey    = "CONFLUENT_KAFKA_API_KEY"
	EnvKafkaAPISecret = "CONFLUENT_KAFKA_API_SECRET"
)

// ConfluentConnection holds the credentials and HTTP client for one Confluent
// Cloud organization.
type ConfluentConnection struct {
	plugin.Connection
	Conf  *inventory.Config
	asset *inventory.Asset

	apiKey    string
	apiSecret string
	baseURL   string
	client    *http.Client

	kafkaKey    string
	kafkaSecret string

	organizationID string
}

func NewConfluentConnection(id uint32, asset *inventory.Asset, conf *inventory.Config) (*ConfluentConnection, error) {
	conn := &ConfluentConnection{
		Connection: plugin.NewConnection(id, asset),
		Conf:       conf,
		asset:      asset,
		baseURL:    DefaultAPIBaseURL,
		client:     &http.Client{Timeout: requestTimeout},
	}

	if base := option(conf, OptionBaseURL); base != "" {
		conn.baseURL = strings.TrimRight(base, "/")
	}

	cloudSecret, kafkaSecret := credentialsFromConf(conf)

	conn.apiKey = option(conf, OptionAPIKey)
	if conn.apiKey == "" {
		conn.apiKey = os.Getenv(EnvAPIKey)
	}
	conn.apiSecret = cloudSecret
	if conn.apiSecret == "" {
		conn.apiSecret = os.Getenv(EnvAPISecret)
	}
	if conn.apiKey == "" || conn.apiSecret == "" {
		return nil, errors.New("a Confluent Cloud API key and secret are required (set " +
			EnvAPIKey + " and " + EnvAPISecret + ", or pass --api-key and --api-secret)")
	}

	// The Kafka API key is optional. It is only needed by the per-cluster REST
	// endpoints, and a scan that never reads topics or ACLs does not need one.
	conn.kafkaKey = option(conf, OptionKafkaAPIKey)
	if conn.kafkaKey == "" {
		conn.kafkaKey = os.Getenv(EnvKafkaAPIKey)
	}
	conn.kafkaSecret = kafkaSecret
	if conn.kafkaSecret == "" {
		conn.kafkaSecret = os.Getenv(EnvKafkaAPISecret)
	}

	return conn, nil
}

// credentialsFromConf pulls the Cloud API secret and the Kafka API secret out
// of the configured credentials. Both arrive as password credentials, so the
// user name is what tells them apart: the Kafka secret is tagged, a bare
// password is the Cloud API secret.
func credentialsFromConf(conf *inventory.Config) (cloudSecret, kafkaSecret string) {
	if conf == nil {
		return "", ""
	}
	for _, cred := range conf.Credentials {
		if cred == nil || len(cred.Secret) == 0 {
			continue
		}
		if cred.Type != mondoovault.CredentialType_password {
			continue
		}
		if strings.EqualFold(cred.User, credentialUserKafka) {
			kafkaSecret = string(cred.Secret)
			continue
		}
		cloudSecret = string(cred.Secret)
	}
	return cloudSecret, kafkaSecret
}

func (c *ConfluentConnection) Name() string { return "confluent" }

func (c *ConfluentConnection) Asset() *inventory.Asset { return c.asset }

// BaseURL is the root of the management API.
func (c *ConfluentConnection) BaseURL() string { return c.baseURL }

// OrganizationID is the organization the API key belongs to. It is resolved
// once during connect and used to build the asset identifier.
func (c *ConfluentConnection) OrganizationID() string { return c.organizationID }

// SetOrganizationID records the organization resolved during connect.
func (c *ConfluentConnection) SetOrganizationID(id string) { c.organizationID = id }

// KafkaCredentialsFor returns the Kafka API key pair to use for one cluster.
//
// A per-cluster pair set through the environment wins over the connection-wide
// pair, so an organization with several clusters can be scanned with one key
// each. The environment variable name is derived from the cluster ID, see
// KafkaEnvSuffix.
func (c *ConfluentConnection) KafkaCredentialsFor(clusterID string) (key, secret string) {
	if suffix := KafkaEnvSuffix(clusterID); suffix != "" {
		key = os.Getenv(EnvKafkaAPIKey + "_" + suffix)
		secret = os.Getenv(EnvKafkaAPISecret + "_" + suffix)
		if key != "" && secret != "" {
			return key, secret
		}
	}
	return c.kafkaKey, c.kafkaSecret
}

// KafkaEnvSuffix renders a cluster ID as the suffix of a per-cluster
// environment variable, for example "lkc-abc123" becomes "LKC_ABC123" so the
// full name reads CONFLUENT_KAFKA_API_KEY_LKC_ABC123. A cluster ID carrying a
// character that cannot appear in an environment variable name yields an empty
// suffix, which disables the per-cluster lookup rather than reading an
// unrelated variable.
func KafkaEnvSuffix(clusterID string) string {
	if clusterID == "" {
		return ""
	}
	var b strings.Builder
	for _, r := range clusterID {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r - 'a' + 'A')
		case r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '-' || r == '_':
			b.WriteRune('_')
		default:
			return ""
		}
	}
	suffix := b.String()
	// A name may not start with a digit, and a suffix made only of separators
	// is not a name at all.
	if suffix == "" || (suffix[0] >= '0' && suffix[0] <= '9') {
		return ""
	}
	if strings.Trim(suffix, "_") == "" {
		return ""
	}
	return suffix
}

func option(conf *inventory.Config, key string) string {
	if conf == nil || conf.Options == nil {
		return ""
	}
	return conf.Options[key]
}
