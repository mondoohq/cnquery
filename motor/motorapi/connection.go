package motorapi

import (
	"errors"
	"net/url"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
)

// ParseFromURI will pars a URI and return the proper endpoint
// valid URIs are:
// - local:// (default)
func (t *TransportConfig) ParseFromURI(uri string) error {
	// special handling for docker since it is not a valid url
	if strings.HasPrefix(uri, "docker://") {
		t.Backend = TransportBackend_CONNECTION_DOCKER_CONTAINER
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

	b, err := backend(u.Scheme)
	if err != nil {
		return err
	}

	t.Backend = b
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

func backend(scheme string) (TransportBackend, error) {
	switch scheme {
	case "ssh":
		return TransportBackend_CONNECTION_SSH, nil
	case "docker":
		// TODO: lets figure out if this is good enough
		// NOTE: this only works because they all convert back to the same types Backend
		// TODO: this may be dangerous and may lead to unexpected behaviour
		return TransportBackend_CONNECTION_DOCKER_IMAGE, nil
	case "local":
		return TransportBackend_CONNECTION_LOCAL_OS, nil
	case "winrm":
		return TransportBackend_CONNECTION_WINRM, nil
	case "aws-ssm":
		return TransportBackend_CONNECTION_AWS_SSM_RUN_COMMAND, nil
	case "tar":
		return TransportBackend_CONNECTION_TAR, nil
	case "mock":
		return TransportBackend_CONNECTION_MOCK, nil
	}

	return TransportBackend_CONNECTION_LOCAL_OS, errors.New("unknown connection scheme: " + scheme)
}

// returns the port number if parsable
func (c *TransportConfig) IntPort() (int, error) {
	var port int
	var err error

	// try to extract port
	if len(c.Port) > 0 {
		portInt, err := strconv.ParseInt(c.Port, 10, 32)
		if err != nil {
			return port, errors.New("invalid port " + c.Port)
		}
		port = int(portInt)
	}
	return port, err
}

func (conn *TransportConfig) ToUrl() string {
	switch conn.Backend {
	case TransportBackend_CONNECTION_SSH:
		return "ssh://" + conn.Host
	case TransportBackend_CONNECTION_DOCKER_CONTAINER:
		return "docker://" + conn.Host[:12]
	case TransportBackend_CONNECTION_DOCKER_IMAGE:
		if strings.HasPrefix(conn.Host, "sha256:") {
			host := strings.Replace(conn.Host, "sha256:", "", -1)
			return "docker://" + host[:12]
		}
		// eg. docker://centos:8
		return "docker://" + conn.Host
	case TransportBackend_CONNECTION_LOCAL_OS:
		return "local://"
	case TransportBackend_CONNECTION_WINRM:
		return "winrm://" + conn.Host
	case TransportBackend_CONNECTION_AWS_SSM_RUN_COMMAND:
		return "aws-ssm://" + conn.Host
	case TransportBackend_CONNECTION_DOCKER_REGISTRY:
		return "docker://" + conn.Host + conn.Path
	case TransportBackend_CONNECTION_TAR:
		return "tar://" + conn.Path
	case TransportBackend_CONNECTION_MOCK:
		return "mock://" + conn.Path
	default:
		log.Warn().Str("backend", conn.Backend.String()).Msg("backend is not supported yet")
		return ""
	}
}
