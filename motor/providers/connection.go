package providers

import (
	"errors"
	"net/url"
	"strconv"
	"strings"

	"go.mondoo.io/mondoo/motor/vault"
)

type TransportConfigOption func(t *TransportConfig) error

func WithCredential(credential *vault.Credential) TransportConfigOption {
	return func(cc *TransportConfig) error {
		cc.AddCredential(credential)
		return nil
	}
}

func WithSudo() TransportConfigOption {
	return func(endpoint *TransportConfig) error {
		endpoint.Sudo = &Sudo{
			Active: true,
		}
		return nil
	}
}

func WithInsecure() TransportConfigOption {
	return func(endpoint *TransportConfig) error {
		endpoint.Insecure = true
		return nil
	}
}

func NewProviderConfig(b ProviderType, opts ...TransportConfigOption) (*TransportConfig, error) {
	t := &TransportConfig{
		Backend: b,
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

func NewProviderFromUrl(uri string, opts ...TransportConfigOption) (*TransportConfig, string, error) {
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

	b, err := GetProviderType(scheme)
	if err != nil {
		return nil, "", err
	}

	t := &TransportConfig{
		Backend:     b,
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

func (cc *TransportConfig) AddCredential(c *vault.Credential) {
	cc.Credentials = append(cc.Credentials, c)
}
