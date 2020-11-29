package transports

import (
	"net/url"

	"github.com/cockroachdb/errors"
)

// parseTransportURI will parse a URI and return the proper transport config
func parseTransportURI(uri string) (*TransportConfig, error) {
	if uri == "" {
		return nil, errors.New("uri cannot be empty")
	}

	u, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}

	b, err := MapSchemeBackend(u.Scheme)
	if err != nil {
		return nil, err
	}

	t := &TransportConfig{
		Backend: b,
		Path:    u.Path,
		Host:    u.Hostname(),
		Port:    u.Port(),
		Options: map[string]string{},
	}

	// extract username and password
	if u.User != nil {
		t.User = u.User.Username()
		// we do not support passwords encoded in the url
		// pwd, ok := u.User.Password()
		// if ok {
		// 	t.Password = pwd
		// }
	}

	return t, nil
}

type TransportConfigOption func(t *TransportConfig)

func WithIdentityFile(identityFile string) TransportConfigOption {
	return func(endpoint *TransportConfig) {
		endpoint.IdentityFiles = append(endpoint.IdentityFiles, identityFile)
	}
}

func WithPassword(password string) TransportConfigOption {
	return func(endpoint *TransportConfig) {
		endpoint.Password = password
	}
}

func WithSudo() TransportConfigOption {
	return func(endpoint *TransportConfig) {
		endpoint.Sudo = &Sudo{
			Active: true,
		}
	}
}

func WithInsecure() TransportConfigOption {
	return func(endpoint *TransportConfig) {
		endpoint.Insecure = true
	}
}

func NewTransportFromUrl(uri string, opts ...TransportConfigOption) (*TransportConfig, error) {
	t, err := parseTransportURI(uri)
	if err != nil {
		return nil, err
	}

	for i := range opts {
		opts[i](t)
	}
	return t, nil
}
