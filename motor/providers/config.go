package providers

import (
	"strings"

	"go.mondoo.io/mondoo/stringx"

	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/proto"
)

func (conn *TransportConfig) Clone() *TransportConfig {
	if conn == nil {
		return nil
	}
	return proto.Clone(conn).(*TransportConfig)
}

func (conn *TransportConfig) ToUrl() string {
	switch conn.Backend {
	case TransportBackend_CONNECTION_SSH:
		return SCHEME_SSH + "://" + conn.Host
	case TransportBackend_CONNECTION_DOCKER_ENGINE_CONTAINER:
		if len(conn.Host) > 12 {
			return "docker://" + conn.Host[:12]
		}
		return SCHEME_DOCKER_CONTAINER + "://" + conn.Host
	case TransportBackend_CONNECTION_DOCKER_ENGINE_IMAGE:
		if strings.HasPrefix(conn.Host, "sha256:") {
			host := strings.Replace(conn.Host, "sha256:", "", -1)
			if len(host) > 12 {
				return "docker://" + host[:12]
			}
			return SCHEME_DOCKER_IMAGE + "://" + host
		}
		// eg. docker://centos:8
		return SCHEME_DOCKER_IMAGE + "://" + conn.Host
	case TransportBackend_CONNECTION_CONTAINER_REGISTRY:
		return SCHEME_CONTAINER_REGISTRY + "://" + conn.Host + conn.Path
	case TransportBackend_CONNECTION_LOCAL_OS:
		return SCHEME_LOCAL
	case TransportBackend_CONNECTION_WINRM:
		return SCHEME_WINRM + "://" + conn.Host
	case TransportBackend_CONNECTION_AWS_SSM_RUN_COMMAND:
		return "aws-ssm://" + conn.Host
	case TransportBackend_CONNECTION_TAR:
		return SCHEME_TAR + "://" + conn.Path
	case TransportBackend_CONNECTION_MOCK:
		return SCHEME_MOCK + "://" + conn.Path
	case TransportBackend_CONNECTION_VSPHERE:
		return SCHEME_VSPHERE + "://" + conn.Host
	case TransportBackend_CONNECTION_VSPHERE_VM:
		return SCHEME_VSPHERE_VM + "://" + conn.Host
	case TransportBackend_CONNECTION_ARISTAEOS:
		return SCHEME_ARISTA + "://" + conn.Host
	case TransportBackend_CONNECTION_AWS:
		return SCHEME_AWS
	case TransportBackend_CONNECTION_AZURE:
		return SCHEME_AZURE
	case TransportBackend_CONNECTION_MS365:
		return SCHEME_MS365
	case TransportBackend_CONNECTION_IPMI:
		return SCHEME_IPMI
	case TransportBackend_CONNECTION_FS:
		return SCHEME_FS
	case TransportBackend_CONNECTION_EQUINIX_METAL:
		return SCHEME_EQUINIX
	case TransportBackend_CONNECTION_K8S:
		return SCHEME_K8S
	case TransportBackend_CONNECTION_GITHUB:
		return SCHEME_GITHUB
	case TransportBackend_CONNECTION_GITLAB:
		return SCHEME_GITLAB
	case TransportBackend_CONNECTION_GCP:
		return SCHEME_GCP
	case TransportBackend_CONNECTION_AWS_EC2_EBS:
		return SCHEME_AWS_EC2_EBS
	case TransportBackend_CONNECTION_TERRAFORM:
		return SCHEME_TERRAFORM
	case TransportBackend_CONNECTION_HOST:
		if _, ok := conn.Options["tls"]; ok {
			return SCHEME_TLS + "://" + conn.Host
		}
		return SCHEME_HOST + "://" + conn.Host
	default:
		log.Warn().Str("backend", conn.Backend.String()).Msg("cannot render backend config")
		return ""
	}
}

func (conn *TransportConfig) IncludesDiscoveryTarget(target string) bool {
	if conn.Discover == nil {
		return false
	}

	return stringx.Contains(conn.Discover.Targets, target)
}
