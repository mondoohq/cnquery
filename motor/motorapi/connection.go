package motorapi

import (
	"strings"

	"github.com/rs/zerolog/log"
)

func (conn *Connection) ToUrl() string {
	switch conn.Backend {
	case ConnectionBackend_CONNECTION_SSH:
		return "ssh://" + conn.Host
	case ConnectionBackend_CONNECTION_DOCKER_CONTAINER:
		return "docker://" + conn.Host[:12]
	case ConnectionBackend_CONNECTION_DOCKER_IMAGE:
		if strings.HasPrefix(conn.Host, "sha256:") {
			host := strings.Replace(conn.Host, "sha256:", "", -1)
			return "docker://" + host[:12]
		}
		// eg. docker://centos:8
		return "docker://" + conn.Host
	case ConnectionBackend_CONNECTION_LOCAL_OS:
		return "local://"
	case ConnectionBackend_CONNECTION_WINRM:
		return "winrm://" + conn.Host
	case ConnectionBackend_CONNECTION_AWS_SSM_RUN_COMMAND:
		return "aws-ssm://" + conn.Host
	case ConnectionBackend_CONNECTION_DOCKER_REGISTRY:
		return "docker://" + conn.Host + conn.Path
	case ConnectionBackend_CONNECTION_TAR:
		return "tar://" + conn.Path
	case ConnectionBackend_CONNECTION_MOCK:
		return "mock://" + conn.Path
	default:
		log.Warn().Str("backend", conn.Backend.String()).Msg("backend is not supported yet")
		return ""
	}
}

func (connection *Connection) ToEndpoint() *Endpoint {
	t := &Endpoint{
		Backend:       assetBackentToMotorBackend(connection.Backend),
		Host:          connection.Host,
		Port:          connection.Port,
		Path:          connection.Path,
		User:          connection.User,
		IdentityFiles: []string{connection.IdentityFile},
		Password:      connection.Password,
		Insecure:      connection.Insecure,
		BearerToken:   connection.BearerToken,
		Sudo:          &Sudo{Active: connection.Sudo},
		// TODO: connection does not expose if the connection should be recorded
		// Record:        record,
	}
	return t
}

// TODO: handle ssm and docker registry
func assetBackentToMotorBackend(backend ConnectionBackend) Backend {
	var motorBackend Backend
	switch backend {
	case ConnectionBackend_CONNECTION_DOCKER_CONTAINER:
		fallthrough
	case ConnectionBackend_CONNECTION_DOCKER_REGISTRY:
		fallthrough
	case ConnectionBackend_CONNECTION_DOCKER_IMAGE:
		motorBackend = BackendDocker
	case ConnectionBackend_CONNECTION_SSH:
		motorBackend = BackendSSH
	case ConnectionBackend_CONNECTION_WINRM:
		motorBackend = BackendWinrm
	case ConnectionBackend_CONNECTION_LOCAL_OS:
		motorBackend = BackendLocal
	case ConnectionBackend_CONNECTION_TAR:
		motorBackend = BackendTAR
	case ConnectionBackend_CONNECTION_MOCK:
		motorBackend = BackendMock
	}
	return motorBackend
}
