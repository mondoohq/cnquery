// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
)

// NormalizeEndpoint validates the MinIO endpoint and returns its host[:port]
// along with whether it is served over TLS. A bare host is assumed to be HTTPS,
// since serving object storage over plaintext exposes both the access keys and
// the objects, and is a misconfiguration rather than a deployment style worth
// defaulting to.
func NormalizeEndpoint(raw string) (host string, secure bool, err error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", false, errors.New("a MinIO endpoint is required")
	}
	if !strings.Contains(trimmed, "://") {
		trimmed = "https://" + trimmed
	}

	parsed, err := url.Parse(trimmed)
	if err != nil {
		return "", false, err
	}
	switch parsed.Scheme {
	case "http":
		secure = false
	case "https":
		secure = true
	default:
		return "", false, errors.New("a MinIO endpoint must use http or https, got " + parsed.Scheme)
	}
	if parsed.Host == "" {
		return "", false, errors.New("a MinIO endpoint must include a host")
	}
	if parsed.Path != "" && parsed.Path != "/" {
		return "", false, errors.New("a MinIO endpoint must not carry a path, got " + parsed.Path)
	}

	return parsed.Host, secure, nil
}

// newTransport builds the HTTP transport the S3 and admin clients share. It
// trusts an operator-supplied authority and honors an explicit request to skip
// verification. A plaintext endpoint needs no TLS configuration, but supplying
// one anyway is an operator mistake worth reporting rather than ignoring.
func newTransport(conf *inventory.Config, secure bool) (http.RoundTripper, error) {
	skipVerify, err := skipVerifyFromConf(conf)
	if err != nil {
		return nil, err
	}

	caCert := option(conf, OptionCACert)
	if caCert == "" {
		caCert = os.Getenv("MINIO_CACERT")
	}

	if !secure {
		if caCert != "" || skipVerify {
			return nil, errors.New("TLS options were supplied for a plain HTTP endpoint; " +
				"use an https:// endpoint or drop --" + OptionCACert + " and --" + OptionTLSSkipVerify)
		}
		return httpTransport.Clone(), nil
	}

	transport := httpTransport.Clone()
	tlsConfig := &tls.Config{MinVersion: tls.VersionTLS12}
	if skipVerify {
		tlsConfig.InsecureSkipVerify = true
	}

	if caCert != "" {
		pool, err := certPool(caCert)
		if err != nil {
			return nil, err
		}
		tlsConfig.RootCAs = pool
	}

	transport.TLSClientConfig = tlsConfig
	return transport, nil
}

// certPool builds a trust store from operator-supplied certificate material.
// The material arrives either as a path or as the PEM itself, because an
// inventory file carries it inline while a shell invocation points at a file.
// Material that is not valid PEM is an error rather than a no-op, so a mangled
// certificate cannot leave the connection quietly trusting only the system
// roots and failing later for an unrelated-looking reason.
func certPool(caCert string) (*x509.CertPool, error) {
	pemData := []byte(caCert)
	if !isPEM(caCert) {
		data, err := os.ReadFile(caCert)
		if err != nil {
			return nil, err
		}
		pemData = data
	}

	pool, err := x509.SystemCertPool()
	if err != nil {
		pool = x509.NewCertPool()
	}
	if !pool.AppendCertsFromPEM(pemData) {
		return nil, errors.New("the value of " + OptionCACert + " is not a valid PEM certificate")
	}
	return pool, nil
}

// skipVerifyFromConf reads the tls-skip-verify option. An unparseable value is
// an error rather than a silent false, so a typo cannot leave the operator
// believing verification was disabled when it was not.
func skipVerifyFromConf(conf *inventory.Config) (bool, error) {
	raw := option(conf, OptionTLSSkipVerify)
	if raw == "" {
		return false, nil
	}
	value, err := strconv.ParseBool(raw)
	if err != nil {
		return false, errors.New("invalid value for " + OptionTLSSkipVerify + ": " + raw)
	}
	return value, nil
}

// isPEM reports whether the value is certificate material rather than a path.
func isPEM(value string) bool {
	return strings.Contains(value, "-----BEGIN")
}
