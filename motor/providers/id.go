package providers

import (
	"encoding/json"
	"errors"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
)

const (
	ProviderID_LOCAL              = "local"
	ProviderID_WINRM              = "winrm"
	ProviderID_SSH                = "ssh"
	ProviderID_DOCKER             = "docker"
	ProviderID_DOCKER_IMAGE       = "docker+image"
	ProviderID_DOCKER_CONTAINER   = "docker+container"
	ProviderID_TAR                = "tar"
	ProviderID_K8S                = "k8s"
	ProviderID_GCR                = "gcr" // TODO: this is not part of the transports, merge with cr
	ProviderID_GCP                = "gcp"
	ProviderID_CONTAINER_REGISTRY = "cr"
	ProviderID_AZURE              = "az"
	ProviderID_AWS                = "aws"
	ProviderID_AWS_SSM            = "aws+ssm"
	ProviderID_VAGRANT            = "vagrant"
	ProviderID_MOCK               = "mock"
	ProviderID_VSPHERE            = "vsphere"
	ProviderID_VSPHERE_VM         = "vsphere+vm"
	ProviderID_ARISTA             = "arista"
	ProviderID_MS365              = "ms365"
	ProviderID_IPMI               = "ipmi"
	ProviderID_FS                 = "fs"
	ProviderID_EQUINIX            = "equinix"
	ProviderID_GITHUB             = "github"
	ProviderID_AWS_EC2_EBS        = "aws-ec2-ebs"
	ProviderID_GITLAB             = "gitlab"
	ProviderID_TERRAFORM          = "terraform"
	ProviderID_HOST               = "host"
	ProviderID_TLS                = "tls"

	// NOTE: its not mapped directly to a transport, it is transformed into ssh
	ProviderID_AWS_EC2_INSTANCE_CONNECT = "aws-ec2-connect"
	ProviderID_AWS_EC2_SSM_SESSION      = "aws-ec2-ssm"
	ProviderID_TERRAFORM_STATE          = "tfstate"
)

var ProviderType_id = map[ProviderType]string{
	ProviderType_LOCAL_OS:                ProviderID_LOCAL,
	ProviderType_SSH:                     ProviderID_SSH,
	ProviderType_WINRM:                   ProviderID_WINRM,
	ProviderType_DOCKER:                  ProviderID_DOCKER,
	ProviderType_DOCKER_ENGINE_IMAGE:     ProviderID_DOCKER_IMAGE,
	ProviderType_DOCKER_ENGINE_CONTAINER: ProviderID_DOCKER_CONTAINER,
	ProviderType_AWS_SSM_RUN_COMMAND:     ProviderID_AWS_SSM,
	ProviderType_CONTAINER_REGISTRY:      ProviderID_CONTAINER_REGISTRY,
	ProviderType_TAR:                     ProviderID_TAR,
	ProviderType_MOCK:                    ProviderID_MOCK,
	ProviderType_VSPHERE:                 ProviderID_VSPHERE,
	ProviderType_ARISTAEOS:               ProviderID_ARISTA,
	ProviderType_AWS:                     ProviderID_AWS,
	ProviderType_GCP:                     ProviderID_GCP,
	ProviderType_AZURE:                   ProviderID_AZURE,
	ProviderType_MS365:                   ProviderID_MS365,
	ProviderType_IPMI:                    ProviderID_IPMI,
	ProviderType_VSPHERE_VM:              ProviderID_VSPHERE_VM,
	ProviderType_FS:                      ProviderID_FS,
	ProviderType_K8S:                     ProviderID_K8S,
	ProviderType_EQUINIX_METAL:           ProviderID_EQUINIX,
	ProviderType_GITHUB:                  ProviderID_GITHUB,
	ProviderType_VAGRANT:                 ProviderID_VAGRANT,
	ProviderType_AWS_EC2_EBS:             ProviderID_AWS_EC2_EBS,
	ProviderType_GITLAB:                  ProviderID_GITLAB,
	ProviderType_TERRAFORM:               ProviderID_TERRAFORM,
	ProviderType_HOST:                    ProviderID_HOST,
	ProviderType_TERRAFORM_STATE:         ProviderID_TERRAFORM_STATE,
}

var ProviderType_idvalue = map[string]ProviderType{
	ProviderID_LOCAL:                    ProviderType_LOCAL_OS,
	ProviderID_SSH:                      ProviderType_SSH,
	ProviderID_WINRM:                    ProviderType_WINRM,
	ProviderID_DOCKER:                   ProviderType_DOCKER,
	ProviderID_DOCKER_IMAGE:             ProviderType_DOCKER_ENGINE_IMAGE,
	ProviderID_DOCKER_CONTAINER:         ProviderType_DOCKER_ENGINE_CONTAINER,
	ProviderID_AWS_SSM:                  ProviderType_AWS_SSM_RUN_COMMAND,
	ProviderID_CONTAINER_REGISTRY:       ProviderType_CONTAINER_REGISTRY,
	ProviderID_TAR:                      ProviderType_TAR,
	ProviderID_MOCK:                     ProviderType_MOCK,
	ProviderID_VSPHERE:                  ProviderType_VSPHERE,
	ProviderID_ARISTA:                   ProviderType_ARISTAEOS,
	ProviderID_AWS:                      ProviderType_AWS,
	ProviderID_GCP:                      ProviderType_GCP,
	ProviderID_AZURE:                    ProviderType_AZURE,
	ProviderID_MS365:                    ProviderType_MS365,
	ProviderID_IPMI:                     ProviderType_IPMI,
	ProviderID_VSPHERE_VM:               ProviderType_VSPHERE_VM,
	ProviderID_FS:                       ProviderType_FS,
	ProviderID_K8S:                      ProviderType_K8S,
	ProviderID_EQUINIX:                  ProviderType_EQUINIX_METAL,
	ProviderID_GITHUB:                   ProviderType_GITHUB,
	ProviderID_VAGRANT:                  ProviderType_VAGRANT,
	ProviderID_AWS_EC2_EBS:              ProviderType_AWS_EC2_EBS,
	ProviderID_GITLAB:                   ProviderType_GITLAB,
	ProviderID_TERRAFORM:                ProviderType_TERRAFORM,
	ProviderID_HOST:                     ProviderType_HOST,
	ProviderID_AWS_EC2_INSTANCE_CONNECT: ProviderType_SSH,
	ProviderID_AWS_EC2_SSM_SESSION:      ProviderType_SSH,
	ProviderID_TERRAFORM_STATE:          ProviderType_TERRAFORM_STATE,
}

func (x ProviderType) Id() string {
	s, ok := ProviderType_id[x]
	if ok {
		return s
	}
	log.Warn().Str("backend", x.String()).Msg("cannot return scheme for backend")
	return strconv.Itoa(int(x))
}

// UnmarshalJSON parses either an int or a string representation of
// CredentialType into the struct
func (s *ProviderType) UnmarshalJSON(data []byte) error {
	// check if we have a number
	var code int32
	err := json.Unmarshal(data, &code)
	if err == nil {
		*s = ProviderType(code)
	} else {
		var name string
		err = json.Unmarshal(data, &name)
		code, ok := ProviderType_idvalue[strings.TrimSpace(name)]
		if !ok {
			return errors.New("unknown backend value: " + string(data))
		}
		*s = code
	}
	return nil
}

func GetProviderType(name string) (ProviderType, error) {
	s, ok := ProviderType_idvalue[name]
	if ok {
		return s, nil
	}

	return ProviderType_LOCAL_OS, errors.New("unknown provider name: " + name)
}
