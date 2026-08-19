// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"crypto/x509"
	"errors"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	vaultapi "github.com/hashicorp/vault/api"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
)

// NormalizeAddress validates the Vault endpoint and returns it without a
// trailing slash, along with its host[:port]. A bare host is assumed to be
// HTTPS, since serving Vault over plaintext is a misconfiguration rather than a
// deployment style worth defaulting to.
func NormalizeAddress(raw string) (address string, host string, err error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", "", errors.New("a Vault address is required")
	}
	if !strings.Contains(trimmed, "://") {
		trimmed = "https://" + trimmed
	}

	parsed, err := url.Parse(trimmed)
	if err != nil {
		return "", "", err
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", "", errors.New("a Vault address must use http or https, got " + parsed.Scheme)
	}
	if parsed.Host == "" {
		return "", "", errors.New("a Vault address must include a host")
	}

	parsed.Path = strings.TrimRight(parsed.Path, "/")
	parsed.RawQuery = ""
	parsed.Fragment = ""

	return strings.TrimRight(parsed.String(), "/"), parsed.Host, nil
}

// applyTLSConfig trusts an operator-supplied authority and honors an explicit
// request to skip verification. The certificate authority arrives either as a
// path or as the PEM itself, because an inventory file carries the material
// inline while a shell invocation points at a file.
func applyTLSConfig(cfg *vaultapi.Config, conf *inventory.Config) error {
	skipVerify, err := skipVerifyFromConf(conf)
	if err != nil {
		return err
	}
	tlsConfig := &vaultapi.TLSConfig{Insecure: skipVerify}

	caCert := option(conf, OptionCACert)
	if caCert == "" {
		caCert = os.Getenv("VAULT_CACERT")
	}

	// Inline material is applied to the transport afterwards. The Vault
	// client's TLS configuration takes paths only, and spilling the PEM to a
	// temporary file would leave one behind for every connection.
	inlinePEM := ""
	if caCert != "" {
		if isPEM(caCert) {
			inlinePEM = caCert
		} else {
			tlsConfig.CACert = caCert
		}
	}

	if err := cfg.ConfigureTLS(tlsConfig); err != nil {
		return err
	}

	if inlinePEM == "" {
		return nil
	}
	return trustInlinePEM(cfg, inlinePEM)
}

// trustInlinePEM adds inline certificate material to the transport's trust
// store. Material that is not valid PEM is an error rather than a no-op, so a
// mangled certificate cannot leave the connection quietly trusting only the
// system roots and failing later for an unrelated-looking reason.
func trustInlinePEM(cfg *vaultapi.Config, pemData string) error {
	transport, ok := cfg.HttpClient.Transport.(*http.Transport)
	if !ok || transport.TLSClientConfig == nil {
		return errors.New("cannot apply an inline certificate authority to the HTTP transport")
	}

	pool := transport.TLSClientConfig.RootCAs
	if pool == nil {
		systemPool, err := x509.SystemCertPool()
		if err != nil {
			systemPool = x509.NewCertPool()
		}
		pool = systemPool
	}

	if !pool.AppendCertsFromPEM([]byte(pemData)) {
		return errors.New("the value of " + OptionCACert + " is not a valid PEM certificate")
	}
	transport.TLSClientConfig.RootCAs = pool
	return nil
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
