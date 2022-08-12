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
	case ProviderType_SSH:
		return ProviderID_SSH + "://" + conn.Host
	case ProviderType_DOCKER_ENGINE_CONTAINER:
		if len(conn.Host) > 12 {
			return "docker://" + conn.Host[:12]
		}
		return ProviderID_DOCKER_CONTAINER + "://" + conn.Host
	case ProviderType_DOCKER_ENGINE_IMAGE:
		if strings.HasPrefix(conn.Host, "sha256:") {
			host := strings.Replace(conn.Host, "sha256:", "", -1)
			if len(host) > 12 {
				return "docker://" + host[:12]
			}
			return ProviderID_DOCKER_IMAGE + "://" + host
		}
		// eg. docker://centos:8
		return ProviderID_DOCKER_IMAGE + "://" + conn.Host
	case ProviderType_CONTAINER_REGISTRY:
		return ProviderID_CONTAINER_REGISTRY + "://" + conn.Host + conn.Path
	case ProviderType_LOCAL_OS:
		return ProviderID_LOCAL
	case ProviderType_WINRM:
		return ProviderID_WINRM + "://" + conn.Host
	case ProviderType_AWS_SSM_RUN_COMMAND:
		return "aws-ssm://" + conn.Host
	case ProviderType_TAR:
		return ProviderID_TAR + "://" + conn.Path
	case ProviderType_MOCK:
		return ProviderID_MOCK + "://" + conn.Path
	case ProviderType_VSPHERE:
		return ProviderID_VSPHERE + "://" + conn.Host
	case ProviderType_VSPHERE_VM:
		return ProviderID_VSPHERE_VM + "://" + conn.Host
	case ProviderType_ARISTAEOS:
		return ProviderID_ARISTA + "://" + conn.Host
	case ProviderType_AWS:
		return ProviderID_AWS
	case ProviderType_AZURE:
		return ProviderID_AZURE
	case ProviderType_MS365:
		return ProviderID_MS365
	case ProviderType_IPMI:
		return ProviderID_IPMI
	case ProviderType_FS:
		return ProviderID_FS
	case ProviderType_EQUINIX_METAL:
		return ProviderID_EQUINIX
	case ProviderType_K8S:
		return ProviderID_K8S
	case ProviderType_GITHUB:
		return ProviderID_GITHUB
	case ProviderType_GITLAB:
		return ProviderID_GITLAB
	case ProviderType_GCP:
		return ProviderID_GCP
	case ProviderType_AWS_EC2_EBS:
		return ProviderID_AWS_EC2_EBS
	case ProviderType_TERRAFORM:
		return ProviderID_TERRAFORM
	case ProviderType_HOST:
		if _, ok := conn.Options["tls"]; ok {
			return ProviderID_TLS + "://" + conn.Host
		}
		return ProviderID_HOST + "://" + conn.Host
	default:
		log.Warn().Str("provider", conn.Backend.String()).Msg("cannot render provider name")
		return ""
	}
}

func (conn *TransportConfig) IncludesDiscoveryTarget(target string) bool {
	if conn.Discover == nil {
		return false
	}

	return stringx.Contains(conn.Discover.Targets, target)
}
