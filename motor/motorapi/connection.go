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
	case ConnectionBackend_CONNECTION_MOCK:
		return "mock://" + conn.Path
	default:
		log.Warn().Str("backend", conn.Backend.String()).Msg("backend is not supported yet")
		return ""
	}
}
