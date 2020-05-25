package types

import (
	"errors"
	"net/url"
	"strconv"
	"strings"
)

type Backend string

func (b Backend) String() string {
	return string(b)
}

const (
	BackendDocker Backend = "docker"
	BackendSSH    Backend = "ssh"
	BackendTAR    Backend = "tar"
	BackendLocal  Backend = "local"
	BackendWinrm  Backend = "winrm"
	BackendMock   Backend = "mock"
)

type Sudo struct {
	Active bool   `json:"active"`
	User   string `json:"user"`
	Shell  string `json:"shell"`
}

// Endpoint that motor interacts with
type Endpoint struct {
	URI      string
	Backend  Backend `json:"backend"`
	User     string  `json:"user"`
	Password string  `json:"password"`
	Host     string  `json:"host"`
	// Ports are not int by default, eg. docker://centos:latest parses a string as port
	// Therefore it is up to the transport to convert the port to what they need
	Port          string   `json:"port"`
	Path          string   `json:"path"`
	IdentityFiles []string `json:"identity_files"`
	BearerToken   string   `json:"bearer_token"`
	Insecure      bool     `json:"insecure"` // disable ssl/tls checks
	Sudo          *Sudo    `json:"sudo"`
}

// ParseFromURI will pars a URI and return the proper endpoint
// valid URIs are:
// - local:// (default)
func (t *Endpoint) ParseFromURI(uri string) error {
	// special handling for docker since it is not a valid url
	if strings.HasPrefix(uri, "docker://") {
		t.Backend = BackendDocker
		t.Host = strings.Replace(uri, "docker://", "", 1)
		return nil
	}

	if uri == "" {
		return errors.New("uri cannot be empty")
	}

	u, err := url.Parse(uri)
	if err != nil {
		return err
	}

	t.Backend = Backend(u.Scheme)
	t.Path = u.Path
	t.Host = u.Hostname()

	// extract username and password
	if u.User != nil {
		t.User = u.User.Username()
		// we do not support passwords encoded in the url
		// pwd, ok := u.User.Password()
		// if ok {
		// 	t.Password = pwd
		// }
	}

	t.Port = u.Port()
	return nil
}

// returns the port number if parsable
func (t *Endpoint) IntPort() (int, error) {
	var port int
	var err error

	// try to extract port
	if len(t.Port) > 0 {
		portInt, err := strconv.ParseInt(t.Port, 10, 32)
		if err != nil {
			return port, errors.New("invalid port " + t.Port)
		}
		port = int(portInt)
	}
	return port, err
}
