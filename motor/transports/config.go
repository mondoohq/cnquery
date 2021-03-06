package transports

import (
	"errors"
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
	case TransportBackend_CONNECTION_VSPHERE_VM:
		return "vsphere+vm://" + conn.Host
	case TransportBackend_CONNECTION_ARISTAEOS:
		return "aristaeos://" + conn.Host
	case TransportBackend_CONNECTION_AWS:
		return "aws://"
	case TransportBackend_CONNECTION_AZURE:
		return "azure://"
	case TransportBackend_CONNECTION_MS365:
		return "ms365://"
	case TransportBackend_CONNECTION_IPMI:
		return "ipmi://"
	case TransportBackend_CONNECTION_FS:
		return "fs://"
	case TransportBackend_CONNECTION_EQUINIX_METAL:
		return "equinix://"
	default:
		log.Warn().Str("backend", conn.Backend.String()).Msg("cannot render backend config")
		return ""
	}
}
