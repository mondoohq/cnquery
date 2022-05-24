package resolver

import (
	"fmt"

	"go.mondoo.io/mondoo/motor/transports/network"
	"go.mondoo.io/mondoo/motor/transports/terraform"

	"go.mondoo.io/mondoo/motor/vault"

	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/motor"
	"go.mondoo.io/mondoo/motor/transports"
	"go.mondoo.io/mondoo/motor/transports/arista"
	aws_transport "go.mondoo.io/mondoo/motor/transports/aws"
	"go.mondoo.io/mondoo/motor/transports/awsec2ebs"
	"go.mondoo.io/mondoo/motor/transports/azure"
	"go.mondoo.io/mondoo/motor/transports/container"
	"go.mondoo.io/mondoo/motor/transports/equinix"
	"go.mondoo.io/mondoo/motor/transports/fs"
	"go.mondoo.io/mondoo/motor/transports/gcp"
	"go.mondoo.io/mondoo/motor/transports/github"
	"go.mondoo.io/mondoo/motor/transports/gitlab"
	"go.mondoo.io/mondoo/motor/transports/ipmi"
	k8s_transport "go.mondoo.io/mondoo/motor/transports/k8s"
	"go.mondoo.io/mondoo/motor/transports/local"
	"go.mondoo.io/mondoo/motor/transports/mock"
	"go.mondoo.io/mondoo/motor/transports/ms365"
	"go.mondoo.io/mondoo/motor/transports/ssh"
	"go.mondoo.io/mondoo/motor/transports/tar"
	"go.mondoo.io/mondoo/motor/transports/vmwareguestapi"
	"go.mondoo.io/mondoo/motor/transports/vsphere"
	"go.mondoo.io/mondoo/motor/transports/winrm"
	"google.golang.org/protobuf/proto"
)

var transportDevelopmentStatus = map[transports.TransportBackend]string{
	transports.TransportBackend_CONNECTION_GITHUB:      "experimental",
	transports.TransportBackend_CONNECTION_AWS_EC2_EBS: "experimental",
}

func warnIncompleteFeature(backend transports.TransportBackend) {
	if transportDevelopmentStatus[backend] != "" {
		log.Warn().Str("feature", backend.String()).Str("status", transportDevelopmentStatus[backend]).Msg("WARNING: you are using an early access feature")
	}
}

// NewMotorConnection establishes a motor connection by using the provided transport configuration
// By default, it uses the id detector mechanisms provided by the transport. User can overwrite that
// behaviour by optionally passing id detector identifier
func NewMotorConnection(tc *transports.TransportConfig, credentialFn func(cred *vault.Credential) (*vault.Credential, error)) (*motor.Motor, error) {
	log.Debug().Msg("establish motor connection")
	var m *motor.Motor

	warnIncompleteFeature(tc.Backend)

	// we clone the config here, and replace all credential references with the real references
	// the clone is important so that credentials are not leaked outside of the function
	resolvedConfig := proto.Clone(tc).(*transports.TransportConfig)
	resolvedCredentials := []*vault.Credential{}
	for i := range resolvedConfig.Credentials {
		credential := resolvedConfig.Credentials[i]
		if credential.SecretId != "" && credentialFn != nil {
			resolvedCredential, err := credentialFn(credential)
			if err != nil {
				log.Debug().Str("secret-id", credential.SecretId).Err(err).Msg("could not fetch secret for motor connection")
				return nil, err
			}
			credential = resolvedCredential
		}
		resolvedCredentials = append(resolvedCredentials, credential)
	}
	resolvedConfig.Credentials = resolvedCredentials

	// establish connection
	switch resolvedConfig.Backend {
	case transports.TransportBackend_CONNECTION_MOCK:
		log.Debug().Msg("connection> load mock transport")
		trans, err := mock.NewFromToml(resolvedConfig)
		if err != nil {
			return nil, err
		}

		m, err = motor.New(trans)
		if err != nil {
			return nil, err
		}
	case transports.TransportBackend_CONNECTION_LOCAL_OS:
		log.Debug().Msg("connection> load local transport")
		trans, err := local.New()
		if err != nil {
			return nil, err
		}

		m, err = motor.New(trans, motor.WithRecoding(resolvedConfig.Record))
		if err != nil {
			return nil, err
		}
	case transports.TransportBackend_CONNECTION_TAR:
		log.Debug().Msg("connection> load tar transport")
		trans, err := tar.New(resolvedConfig)
		if err != nil {
			return nil, err
		}

		m, err = motor.New(trans, motor.WithRecoding(resolvedConfig.Record))
		if err != nil {
			return nil, err
		}
	case transports.TransportBackend_CONNECTION_CONTAINER_REGISTRY:
		log.Debug().Msg("connection> load container registry transport")
		trans, err := container.NewContainerRegistryImage(resolvedConfig)
		if err != nil {
			return nil, err
		}
		m, err = motor.New(trans, motor.WithRecoding(resolvedConfig.Record))
		if err != nil {
			return nil, err
		}
	case transports.TransportBackend_CONNECTION_DOCKER_ENGINE_CONTAINER:
		log.Debug().Msg("connection> load docker engine container transport")
		trans, err := container.NewDockerEngineContainer(resolvedConfig)
		if err != nil {
			return nil, err
		}
		m, err = motor.New(trans, motor.WithRecoding(resolvedConfig.Record))
		if err != nil {
			return nil, err
		}
	case transports.TransportBackend_CONNECTION_DOCKER_ENGINE_IMAGE:
		log.Debug().Msg("connection> load docker engine image transport")
		trans, err := container.NewDockerEngineImage(resolvedConfig)
		if err != nil {
			return nil, err
		}
		m, err = motor.New(trans, motor.WithRecoding(resolvedConfig.Record))
		if err != nil {
			return nil, err
		}
	case transports.TransportBackend_CONNECTION_SSH:
		log.Debug().Msg("connection> load ssh transport")
		trans, err := ssh.New(resolvedConfig)
		if err != nil {
			return nil, err
		}

		m, err = motor.New(trans, motor.WithRecoding(resolvedConfig.Record))
		if err != nil {
			return nil, err
		}
	case transports.TransportBackend_CONNECTION_WINRM:
		log.Debug().Msg("connection> load winrm transport")
		trans, err := winrm.New(resolvedConfig)
		if err != nil {
			return nil, err
		}

		m, err = motor.New(trans, motor.WithRecoding(resolvedConfig.Record))
		if err != nil {
			return nil, err
		}
	case transports.TransportBackend_CONNECTION_VSPHERE:
		log.Debug().Msg("connection> load vsphere transport")
		trans, err := vsphere.New(resolvedConfig)
		if err != nil {
			return nil, err
		}

		m, err = motor.New(trans)
		if err != nil {
			return nil, err
		}
	case transports.TransportBackend_CONNECTION_ARISTAEOS:
		log.Debug().Msg("connection> load arista eos transport")
		trans, err := arista.New(resolvedConfig)
		if err != nil {
			return nil, err
		}
		m, err = motor.New(trans)
		if err != nil {
			return nil, err
		}
	case transports.TransportBackend_CONNECTION_AWS:
		log.Debug().Msg("connection> load aws transport")
		trans, err := aws_transport.New(resolvedConfig, aws_transport.TransportOptions(resolvedConfig.Options)...)
		if err != nil {
			return nil, err
		}
		m, err = motor.New(trans)
		if err != nil {
			return nil, err
		}
	case transports.TransportBackend_CONNECTION_GCP:
		log.Debug().Msg("connection> load gcp transport")
		trans, err := gcp.New(resolvedConfig)
		if err != nil {
			return nil, err
		}
		m, err = motor.New(trans)
		if err != nil {
			return nil, err
		}
	case transports.TransportBackend_CONNECTION_AZURE:
		log.Debug().Msg("connection> load azure transport")
		trans, err := azure.New(resolvedConfig)
		if err != nil {
			return nil, err
		}
		m, err = motor.New(trans)
		if err != nil {
			return nil, err
		}
	case transports.TransportBackend_CONNECTION_MS365:
		log.Debug().Msg("connection> load microsoft 365 transport")
		trans, err := ms365.New(resolvedConfig)
		if err != nil {
			return nil, err
		}
		m, err = motor.New(trans)
		if err != nil {
			return nil, err
		}
	case transports.TransportBackend_CONNECTION_IPMI:
		log.Debug().Msg("connection> load ipmi transport")
		trans, err := ipmi.New(resolvedConfig)
		if err != nil {
			return nil, err
		}
		m, err = motor.New(trans)
		if err != nil {
			return nil, err
		}
	case transports.TransportBackend_CONNECTION_VSPHERE_VM:
		trans, err := vmwareguestapi.New(resolvedConfig)
		if err != nil {
			return nil, err
		}
		m, err = motor.New(trans, motor.WithRecoding(resolvedConfig.Record))
		if err != nil {
			return nil, err
		}
	case transports.TransportBackend_CONNECTION_FS:
		trans, err := fs.New(resolvedConfig)
		if err != nil {
			return nil, err
		}
		m, err = motor.New(trans, motor.WithRecoding(resolvedConfig.Record))
		if err != nil {
			return nil, err
		}
	case transports.TransportBackend_CONNECTION_EQUINIX_METAL:
		trans, err := equinix.New(resolvedConfig)
		if err != nil {
			return nil, err
		}
		m, err = motor.New(trans)
		if err != nil {
			return nil, err
		}
	case transports.TransportBackend_CONNECTION_K8S:
		trans, err := k8s_transport.New(resolvedConfig)
		if err != nil {
			return nil, err
		}
		m, err = motor.New(trans)
		if err != nil {
			return nil, err
		}
	case transports.TransportBackend_CONNECTION_GITHUB:
		trans, err := github.New(resolvedConfig)
		if err != nil {
			return nil, err
		}
		m, err = motor.New(trans)
		if err != nil {
			return nil, err
		}
	case transports.TransportBackend_CONNECTION_GITLAB:
		trans, err := gitlab.New(tc)
		if err != nil {
			return nil, err
		}
		m, err = motor.New(trans)
		if err != nil {
			return nil, err
		}
	case transports.TransportBackend_CONNECTION_AWS_EC2_EBS:
		trans, err := awsec2ebs.New(tc)
		if err != nil {
			return nil, err
		}
		// TODO (jaym) before merge: The ebs transport is being lost. This
		// is problematic. It will break the platform id detection being added
		m, err = motor.New(trans.FsTransport)
		if err != nil {
			return nil, err
		}
	case transports.TransportBackend_CONNECTION_TERRAFORM:
		trans, err := terraform.New(tc)
		if err != nil {
			return nil, err
		}
		m, err = motor.New(trans)
		if err != nil {
			return nil, err
		}
	case transports.TransportBackend_CONNECTION_HOST:
		trans, err := network.New(tc)
		if err != nil {
			return nil, err
		}
		m, err = motor.New(trans)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("connection> unsupported backend '%s'", resolvedConfig.Backend)
	}

	return m, nil
}
