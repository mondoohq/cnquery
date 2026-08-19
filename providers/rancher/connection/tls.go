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
	"time"

	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
)

// NormalizeURL validates the Rancher endpoint and returns it without a trailing
// slash, along with its host[:port]. A bare host is assumed to be HTTPS, since
// serving Rancher over plaintext is a misconfiguration rather than a deployment
// style worth defaulting to. A "/v3" suffix is trimmed, because operators copy
// the endpoint out of the API browser as often as out of the address bar.
func NormalizeURL(raw string) (address string, host string, err error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", "", errors.New("a Rancher URL is required")
	}
	if !strings.Contains(trimmed, "://") {
		trimmed = "https://" + trimmed
	}

	parsed, err := url.Parse(trimmed)
	if err != nil {
		return "", "", err
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", "", errors.New("a Rancher URL must use http or https, got " + parsed.Scheme)
	}
	if parsed.Host == "" {
		return "", "", errors.New("a Rancher URL must include a host")
	}

	parsed.Path = strings.TrimRight(parsed.Path, "/")
	for _, suffix := range []string{"/v3", "/v1"} {
		if strings.HasSuffix(parsed.Path, suffix) {
			parsed.Path = strings.TrimSuffix(parsed.Path, suffix)
			break
		}
	}
	parsed.RawQuery = ""
	parsed.Fragment = ""

	return strings.TrimRight(parsed.String(), "/"), parsed.Host, nil
}

// newHTTPClient builds the transport the API client uses, trusting an
// operator-supplied authority and honoring an explicit request to skip
// verification. The certificate authority arrives either as a path or as the
// PEM itself, because an inventory file carries the material inline while a
// shell invocation points at a file.
func newHTTPClient(conf *inventory.Config, timeout time.Duration) (*http.Client, error) {
	skipVerify, err := skipVerifyFromConf(conf)
	if err != nil {
		return nil, err
	}

	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
		//nolint:gosec // opt-in only, for lab servers using a self-signed certificate
		InsecureSkipVerify: skipVerify,
	}

	caCert := option(conf, OptionCACert)
	if caCert == "" {
		caCert = os.Getenv("RANCHER_CACERT")
	}
	if caCert != "" {
		pool, err := certPool(caCert)
		if err != nil {
			return nil, err
		}
		tlsConfig.RootCAs = pool
	}

	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = tlsConfig

	return &http.Client{Transport: transport, Timeout: timeout}, nil
}

// certPool adds the operator-supplied authority to the system trust store.
// Material that is neither readable nor valid PEM is an error rather than a
// no-op, so a mangled certificate cannot leave the connection quietly trusting
// only the system roots and failing later for an unrelated-looking reason.
func certPool(caCert string) (*x509.CertPool, error) {
	pemData := []byte(caCert)
	if !isPEM(caCert) {
		content, err := os.ReadFile(caCert)
		if err != nil {
			return nil, err
		}
		pemData = content
	}

	pool, err := x509.SystemCertPool()
	if err != nil || pool == nil {
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
