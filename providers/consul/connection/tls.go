// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"errors"
	"net/url"
	"os"
	"strconv"
	"strings"

	consulapi "github.com/hashicorp/consul/api"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
)

// DefaultAddress is the endpoint a Consul agent serves its HTTP API on when
// nothing else is configured.
const DefaultAddress = "http://127.0.0.1:8500"

// NormalizeAddress validates the Consul endpoint and returns it without a
// trailing slash, along with its host[:port]. A bare host is assumed to be
// HTTP, because that is what the Consul client and CLI assume and what an
// unconfigured agent serves. The port is filled in when the address omits it,
// so two spellings of the same agent do not become two assets.
func NormalizeAddress(raw string) (address string, host string, err error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", "", errors.New("a Consul address is required")
	}
	if !strings.Contains(trimmed, "://") {
		trimmed = "http://" + trimmed
	}

	parsed, err := url.Parse(trimmed)
	if err != nil {
		return "", "", err
	}
	switch parsed.Scheme {
	case "http", "https":
	default:
		// A unix socket address is a legitimate Consul deployment, but it
		// carries no host to build an asset identity from, so it is refused
		// rather than silently identified as some other agent.
		return "", "", errors.New("a Consul address must use http or https, got " + parsed.Scheme)
	}
	if parsed.Hostname() == "" {
		return "", "", errors.New("a Consul address must include a host")
	}

	if parsed.Port() == "" {
		if parsed.Scheme == "https" {
			parsed.Host = parsed.Host + ":8501"
		} else {
			parsed.Host = parsed.Host + ":8500"
		}
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
func applyTLSConfig(cfg *consulapi.Config, conf *inventory.Config) error {
	skipVerify, err := skipVerifyFromConf(conf)
	if err != nil {
		return err
	}
	if skipVerify {
		cfg.TLSConfig.InsecureSkipVerify = true
	}

	caCert := option(conf, OptionCACert)
	if caCert == "" {
		caCert = os.Getenv(consulapi.HTTPCAFile)
	}
	if caCert == "" {
		return nil
	}

	if isPEM(caCert) {
		// The Consul client accepts inline material directly, so there is no
		// need to spill the PEM to a temporary file that would then be left
		// behind for every connection.
		cfg.TLSConfig.CAPem = []byte(caCert)
		cfg.TLSConfig.CAFile = ""
		return nil
	}
	cfg.TLSConfig.CAFile = caCert
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
