package types

import (
	"errors"
	"net/url"
	"strconv"
)

// Endpoint that motor interacts with
type Endpoint struct {
	URI            string
	Backend        string `json:"backend"`
	User           string `json:"user"`
	Password       string `json:"password"`
	Host           string `json:"host"`
	Port           int    `json:"port"`
	Path           string `json:"path"`
	PrivateKeyPath string `json:"private_key"`
}

// ParseFromURI will pars a URI and return the proper endpoint
// valid URIs are:
// - local:// (default)
func (t *Endpoint) ParseFromURI(uri string) error {
	if uri == "" {
		return errors.New("uri cannot be empty")
	}

	u, err := url.Parse(uri)
	if err != nil {
		return err
	}

	t.Backend = u.Scheme
	t.Path = u.Path
	t.Host = u.Hostname()

	// extract username and password
	if u.User != nil {
		t.User = u.User.Username()
		pwd, ok := u.User.Password()
		if ok {
			t.Password = pwd
		}
	}

	// try to extract port
	port := u.Port()
	if len(port) > 0 {
		portInt, err := strconv.ParseInt(port, 10, 32)
		if err != nil {
			return errors.New("invalid port " + port)
		}
		t.Port = int(portInt)
	}

	return nil
}
