// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package inventory

import (
	"encoding/json"
	"errors"
	"strings"
)

// FIXME: this file can be deleted in v10

const (
	ProviderID_LOCAL                = "local"
	ProviderID_WINRM                = "winrm"
	ProviderID_SSH                  = "ssh"
	ProviderID_DOCKER               = "docker"
	ProviderID_DOCKER_IMAGE         = "docker+image"
	ProviderID_DOCKER_CONTAINER     = "docker+container"
	ProviderID_TAR                  = "tar"
	ProviderID_K8S                  = "k8s"
	ProviderID_GCR                  = "gcr" // TODO: this is not part of the transports, merge with cr
	ProviderID_GCP                  = "gcp"
	ProviderID_CONTAINER_REGISTRY   = "cr"
	ProviderID_AZURE                = "az"
	ProviderID_AWS                  = "aws"
	ProviderID_AWS_SSM              = "aws+ssm"
	ProviderID_VAGRANT              = "vagrant"
	ProviderID_MOCK                 = "mock"
	ProviderID_VSPHERE              = "vsphere"
	ProviderID_VSPHERE_VM           = "vsphere+vm"
	ProviderID_ARISTA               = "arista"
	ProviderID_MS365                = "ms365"
	ProviderID_IPMI                 = "ipmi"
	ProviderID_FS                   = "fs"
	ProviderID_EQUINIX              = "equinix"
	ProviderID_GITHUB               = "github"
	ProviderID_AWS_EC2_EBS          = "aws-ec2-ebs"
	ProviderID_GITLAB               = "gitlab"
	ProviderID_TERRAFORM            = "terraform"
	ProviderID_HOST                 = "host"
	ProviderID_TLS                  = "tls"
	ProviderID_OKTA                 = "okta"
	ProviderID_GOOGLE_WORKSPACE     = "googleworkspace"
	ProviderID_SLACK                = "slack"
	ProviderID_VCD                  = "vcd"
	ProviderID_OCI                  = "oci"
	ProviderID_OPCUA                = "opc-ua"
	ProviderID_GCP_COMPUTE_INSTANCE = "gcp-compute-instance"

	// NOTE: its not mapped directly to a transport, it is transformed into ssh
	ProviderID_AWS_EC2_INSTANCE_CONNECT = "aws-ec2-connect"
	ProviderID_AWS_EC2_SSM_SESSION      = "aws-ec2-ssm"
	ProviderID_TERRAFORM_STATE          = "tfstate"
)

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
	ProviderID_OKTA:                     ProviderType_OKTA,
	ProviderID_GOOGLE_WORKSPACE:         ProviderType_GOOGLE_WORKSPACE,
	ProviderID_SLACK:                    ProviderType_SLACK,
	ProviderID_VCD:                      ProviderType_VCD,
	ProviderID_OCI:                      ProviderType_OCI,
	ProviderID_OPCUA:                    ProviderType_OPCUA,
	ProviderID_GCP_COMPUTE_INSTANCE:     ProviderType_GCP_COMPUTE_INSTANCE_SNAPSHOT,
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

func connBackendToType(backend ProviderType) string {
	switch backend {
	case ProviderType_LOCAL_OS:
		return "os"
	case ProviderType_DOCKER_ENGINE_IMAGE:
		return "docker-image"
	case ProviderType_DOCKER_ENGINE_CONTAINER:
		return "docker-container"
	case ProviderType_SSH:
		return "ssh"
	case ProviderType_WINRM:
		return "winrm"
	case ProviderType_AWS_SSM_RUN_COMMAND:
		return "aws-ssm-run-command"
	case ProviderType_CONTAINER_REGISTRY:
		return "container-registry"
	case ProviderType_TAR:
		return "tar"
	case ProviderType_MOCK:
		return "mock"
	case ProviderType_VSPHERE:
		return "vsphere"
	case ProviderType_ARISTAEOS:
		return "arista-eos"
	case ProviderType_AWS:
		return "aws"
	case ProviderType_GCP:
		return "gcp"
	case ProviderType_AZURE:
		return "azure"
	case ProviderType_MS365:
		return "ms365"
	case ProviderType_IPMI:
		return "ipmi"
	case ProviderType_VSPHERE_VM:
		return "vsphere-vm"
	case ProviderType_FS:
		return "fs"
	case ProviderType_K8S:
		return "k8s"
	case ProviderType_EQUINIX_METAL:
		return "equinix-metal"
	case ProviderType_DOCKER:
		return "docker"
	case ProviderType_GITHUB:
		return "github"
	case ProviderType_VAGRANT:
		return "vagrant"
	case ProviderType_AWS_EC2_EBS:
		return "aws-ec2-ebs"
	case ProviderType_GITLAB:
		return "gitlab"
	case ProviderType_TERRAFORM:
		return "terraform"
	case ProviderType_HOST:
		return "host"
	case ProviderType_UNKNOWN:
		return "unknown"
	case ProviderType_OKTA:
		return "okta"
	case ProviderType_GOOGLE_WORKSPACE:
		return "google-workspace"
	case ProviderType_SLACK:
		return "slack"
	case ProviderType_VCD:
		return "vcd"
	case ProviderType_OCI:
		return "oci"
	case ProviderType_OPCUA:
		return "opcua"
	case ProviderType_GCP_COMPUTE_INSTANCE_SNAPSHOT:
		return "gcp-compute-instance-snapshot"
	default:
		return ""
	}
}
