package transports

import (
	"errors"
	"net/url"

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

func NewTransportConfig(b TransportBackend, opts ...TransportConfigOption) (*TransportConfig, error) {
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

func NewTransportFromUrl(uri string, opts ...TransportConfigOption) (*TransportConfig, string, error) {
	if uri == "" {
		return nil, "", errors.New("uri cannot be empty")
	}

	u, err := url.Parse(uri)
	if err != nil {
		return nil, "", err
	}

	b, err := MapSchemeBackend(u.Scheme)
	if err != nil {
		return nil, "", err
	}

	t := &TransportConfig{
		Backend:     b,
		Host:        u.Hostname(),
		Port:        u.Port(),
		Path:        u.Path,
		Options:     map[string]string{},
		Credentials: []*vault.Credential{},
	}

	for i := range opts {
		err = opts[i](t)
		if err != nil {
			return nil, "", err
		}
	}
	return t, u.User.Username(), nil
}

func (cc *TransportConfig) AddCredential(c *vault.Credential) {
	cc.Credentials = append(cc.Credentials, c)
}
