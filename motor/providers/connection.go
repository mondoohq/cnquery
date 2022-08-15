package providers

import (
	"errors"
	"net/url"
	"strconv"
	"strings"

	"go.mondoo.io/mondoo/motor/vault"
)

type ConfigOption func(t *Config) error

func WithCredential(credential *vault.Credential) ConfigOption {
	return func(cc *Config) error {
		cc.AddCredential(credential)
		return nil
	}
}

func WithSudo() ConfigOption {
	return func(endpoint *Config) error {
		endpoint.Sudo = &Sudo{
			Active: true,
		}
		return nil
	}
}

func WithInsecure() ConfigOption {
	return func(endpoint *Config) error {
		endpoint.Insecure = true
		return nil
	}
}

func NewProviderConfig(pt ProviderType, opts ...ConfigOption) (*Config, error) {
	t := &Config{
		Backend: pt,
	}

	var err error
	for i := range opts {
		err = opts[i](t)
		if err != nil {
			return nil, err
		}
	}

	return t, nil
}

func NewProviderFromUrl(uri string, opts ...ConfigOption) (*Config, string, error) {
	if uri == "" {
		return nil, "", errors.New("uri cannot be empty")
	}

	scheme := uri
	var hostname, username, path string
	var port int
	if strings.Contains(uri, "://") {
		u, err := url.Parse(uri)
		if err != nil {
			return nil, "", err
		}
		scheme = u.Scheme
		hostname = u.Hostname()
		path = u.Path
		username = u.User.Username()

		if u.Port() != "" {
			port, err = strconv.Atoi(u.Port())
			if err != nil {
				return nil, "", err
			}
		}
	}

	pt, err := GetProviderType(scheme)
	if err != nil {
		return nil, "", err
	}

	t := &Config{
		Backend:     pt,
		Host:        hostname,
		Port:        int32(port),
		Path:        path,
		Options:     map[string]string{},
		Credentials: []*vault.Credential{},
	}

	for i := range opts {
		err = opts[i](t)
		if err != nil {
			return nil, "", err
		}
	}
	return t, username, nil
}

func (cc *Config) AddCredential(c *vault.Credential) {
	cc.Credentials = append(cc.Credentials, c)
}
