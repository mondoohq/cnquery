package providers

import (
	"strings"

	"go.mondoo.com/cnquery/stringx"

	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/proto"
)

func (cfg *Config) Clone() *Config {
	if cfg == nil {
		return nil
	}
	return proto.Clone(cfg).(*Config)
}

func (cfg *Config) ToUrl() string {
	switch cfg.Backend {
	case ProviderType_SSH:
		return ProviderID_SSH + "://" + cfg.Host
	case ProviderType_DOCKER_ENGINE_CONTAINER:
		if len(cfg.Host) > 12 {
			return "docker://" + cfg.Host[:12]
		}
		return ProviderID_DOCKER_CONTAINER + "://" + cfg.Host
	case ProviderType_DOCKER_ENGINE_IMAGE:
		if strings.HasPrefix(cfg.Host, "sha256:") {
			host := strings.Replace(cfg.Host, "sha256:", "", -1)
			if len(host) > 12 {
				return "docker://" + host[:12]
			}
			return ProviderID_DOCKER_IMAGE + "://" + host
		}
		// eg. docker://centos:8
		return ProviderID_DOCKER_IMAGE + "://" + cfg.Host
	case ProviderType_CONTAINER_REGISTRY:
		return ProviderID_CONTAINER_REGISTRY + "://" + cfg.Host + cfg.Path
	case ProviderType_LOCAL_OS:
		return ProviderID_LOCAL
	case ProviderType_WINRM:
		return ProviderID_WINRM + "://" + cfg.Host
	case ProviderType_AWS_SSM_RUN_COMMAND:
		return "aws-ssm://" + cfg.Host
	case ProviderType_TAR:
		return ProviderID_TAR + "://" + cfg.Path
	case ProviderType_MOCK:
		return ProviderID_MOCK + "://" + cfg.Path
	case ProviderType_VSPHERE:
		return ProviderID_VSPHERE + "://" + cfg.Host
	case ProviderType_VSPHERE_VM:
		return ProviderID_VSPHERE_VM + "://" + cfg.Host
	case ProviderType_ARISTAEOS:
		return ProviderID_ARISTA + "://" + cfg.Host
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
		if _, ok := cfg.Options["tls"]; ok {
			return ProviderID_TLS + "://" + cfg.Host
		}
		return ProviderID_HOST + "://" + cfg.Host
	case ProviderType_TERRAFORM_STATE:
		return ProviderID_TERRAFORM_STATE
	default:
		log.Warn().Str("provider", cfg.Backend.String()).Msg("cannot render provider name")
		return ""
	}
}

func (cfg *Config) IncludesDiscoveryTarget(target string) bool {
	if cfg.Discover == nil {
		return false
	}

	return stringx.Contains(cfg.Discover.Targets, target)
}
