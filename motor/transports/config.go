package transports

import (
	"errors"
	"net/url"
	"strconv"
	"strings"

	"github.com/golang/protobuf/proto"
	"github.com/rs/zerolog/log"
)

func (conn *TransportConfig) Clone() *TransportConfig {
	if conn == nil {
		return nil
	}
	return proto.Clone(conn).(*TransportConfig)
}

// ParseFromURI will pars a URI and return the proper endpoint
// valid URIs are:
// - local:// (default)
func (t *TransportConfig) ParseFromURI(uri string) error {
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
	case "vsphere":
		return TransportBackend_CONNECTION_VSPHERE, nil
	case "aristaeos":
		return TransportBackend_CONNECTION_ARISTAEOS, nil
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
	case TransportBackend_CONNECTION_DOCKER_ENGINE_CONTAINER:
		if len(conn.Host) > 12 {
			return "docker://" + conn.Host[:12]
		}
		return "docker://" + conn.Host
	case TransportBackend_CONNECTION_DOCKER_ENGINE_IMAGE:
		if strings.HasPrefix(conn.Host, "sha256:") {
			host := strings.Replace(conn.Host, "sha256:", "", -1)
			if len(host) > 12 {
				return "docker://" + host[:12]
			}
			return "docker://" + host
		}
		// eg. docker://centos:8
		return "docker://" + conn.Host
	case TransportBackend_CONNECTION_LOCAL_OS:
		return "local://"
	case TransportBackend_CONNECTION_WINRM:
		return "winrm://" + conn.Host
	case TransportBackend_CONNECTION_AWS_SSM_RUN_COMMAND:
		return "aws-ssm://" + conn.Host
	case TransportBackend_CONNECTION_CONTAINER_REGISTRY:
		return "docker://" + conn.Host + conn.Path
	case TransportBackend_CONNECTION_TAR:
		return "tar://" + conn.Path
	case TransportBackend_CONNECTION_MOCK:
		return "mock://" + conn.Path
	case TransportBackend_CONNECTION_VSPHERE:
		return "vsphere://" + conn.Host
	case TransportBackend_CONNECTION_ARISTAEOS:
		return "aristaeos://" + conn.Host
	default:
		log.Warn().Str("backend", conn.Backend.String()).Msg("cannot render backend config")
		return ""
	}
}
