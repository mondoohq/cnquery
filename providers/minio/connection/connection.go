// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"errors"
	"net/http"
	"os"
	"time"

	madmin "github.com/minio/madmin-go/v4"
	minio "github.com/minio/minio-go/v7"
	miniocreds "github.com/minio/minio-go/v7/pkg/credentials"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	mondoovault "go.mondoo.com/mql/v13/providers-sdk/v1/vault"
)

const (
	// requestTimeout bounds a single API call. MinIO answers admin and bucket
	// metadata requests from memory or from the drives holding its
	// configuration, so a slow response means the deployment is unhealthy
	// rather than the work being large.
	requestTimeout = 60 * time.Second

	// OptionEndpoint names the S3 and admin API endpoint.
	OptionEndpoint = "endpoint"
	// OptionAccessKey carries the access key. The matching secret key arrives
	// as a credential rather than an option, since it is the secret half.
	OptionAccessKey = "access-key"
	// OptionRegion overrides the region the client signs requests for. MinIO
	// accepts any region on a deployment that does not set one.
	OptionRegion = "region"
	// OptionCACert names the certificate authority to trust, either as the PEM
	// itself or as a path to it. MinIO is commonly published under a private
	// authority, and trusting it keeps the certificate checked.
	OptionCACert = "ca-cert"
	// OptionTLSSkipVerify disables certificate verification. It exists for lab
	// deployments using a self-signed certificate and is never appropriate
	// against a production MinIO.
	OptionTLSSkipVerify = "tls-skip-verify"
)

// MinioConnection holds authenticated clients for one MinIO deployment. The S3
// client reads bucket configuration and the admin client reads the deployment,
// identity and logging configuration, so both are needed for a full scan.
type MinioConnection struct {
	plugin.Connection
	Conf  *inventory.Config
	asset *inventory.Asset

	client *minio.Client
	admin  *madmin.AdminClient
	host   string
	secure bool
	region string
}

func NewMinioConnection(id uint32, asset *inventory.Asset, conf *inventory.Config) (*MinioConnection, error) {
	conn := &MinioConnection{
		Connection: plugin.NewConnection(id, asset),
		Conf:       conf,
		asset:      asset,
	}

	endpoint := option(conf, OptionEndpoint)
	if endpoint == "" && conf != nil {
		endpoint = conf.Host
	}
	if endpoint == "" {
		endpoint = os.Getenv("MINIO_ENDPOINT")
	}
	if endpoint == "" {
		return nil, errors.New("a MinIO endpoint is required (set MINIO_ENDPOINT or use --endpoint)")
	}

	host, secure, err := NormalizeEndpoint(endpoint)
	if err != nil {
		return nil, err
	}
	conn.host = host
	conn.secure = secure

	accessKey, secretKey := credentialsFromConf(conf)
	if accessKey == "" {
		accessKey = option(conf, OptionAccessKey)
	}
	if accessKey == "" {
		accessKey = firstNonEmpty(os.Getenv("MINIO_ROOT_USER"), os.Getenv("MINIO_ACCESS_KEY"))
	}
	if secretKey == "" {
		secretKey = firstNonEmpty(os.Getenv("MINIO_ROOT_PASSWORD"), os.Getenv("MINIO_SECRET_KEY"))
	}
	if accessKey == "" || secretKey == "" {
		return nil, errors.New("a MinIO access key and secret key are required " +
			"(set MINIO_ROOT_USER and MINIO_ROOT_PASSWORD, or MINIO_ACCESS_KEY and MINIO_SECRET_KEY, " +
			"or pass --access-key with --secret-key)")
	}

	transport, err := newTransport(conf, secure)
	if err != nil {
		return nil, err
	}

	conn.region = option(conf, OptionRegion)
	if conn.region == "" {
		conn.region = os.Getenv("MINIO_REGION")
	}

	creds := miniocreds.NewStaticV4(accessKey, secretKey, "")
	conn.client, err = minio.New(host, &minio.Options{
		Creds:     creds,
		Secure:    secure,
		Region:    conn.region,
		Transport: transport,
	})
	if err != nil {
		return nil, err
	}

	conn.admin, err = madmin.NewWithOptions(host, &madmin.Options{
		Creds:     creds,
		Secure:    secure,
		Transport: transport,
	})
	if err != nil {
		return nil, err
	}

	return conn, nil
}

// credentialsFromConf pulls the access key and secret key out of the configured
// credentials. The key pair travels as a single credential, with the access key
// in the user field and the secret key in the secret.
func credentialsFromConf(conf *inventory.Config) (accessKey, secretKey string) {
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
		return cred.User, string(cred.Secret)
	}
	return "", ""
}

func (c *MinioConnection) Name() string { return "minio" }

func (c *MinioConnection) Asset() *inventory.Asset { return c.asset }

// Client returns the authenticated S3 client, which reads bucket configuration.
func (c *MinioConnection) Client() *minio.Client { return c.client }

// Admin returns the authenticated admin client, which reads deployment,
// identity and logging configuration.
func (c *MinioConnection) Admin() *madmin.AdminClient { return c.admin }

// Host is the host[:port] of the API endpoint, used to build platform IDs.
func (c *MinioConnection) Host() string { return c.host }

// Secure reports whether the API endpoint is reached over HTTPS.
func (c *MinioConnection) Secure() bool { return c.secure }

// Region is the region the client signs requests for, empty when unset.
func (c *MinioConnection) Region() string { return c.region }

func option(conf *inventory.Config, key string) string {
	if conf == nil || conf.Options == nil {
		return ""
	}
	return conf.Options[key]
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

// httpTransport is the shared base every connection clones, so a scan of many
// deployments does not build a fresh connection pool per asset.
var httpTransport = &http.Transport{
	Proxy:                 http.ProxyFromEnvironment,
	MaxIdleConns:          32,
	IdleConnTimeout:       90 * time.Second,
	TLSHandshakeTimeout:   10 * time.Second,
	ResponseHeaderTimeout: requestTimeout,
}
