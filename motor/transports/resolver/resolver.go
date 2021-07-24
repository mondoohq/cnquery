package resolver

import (
	"fmt"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/motor"
	"go.mondoo.io/mondoo/motor/motorid"
	"go.mondoo.io/mondoo/motor/transports"
	"go.mondoo.io/mondoo/motor/transports/arista"
	aws_transport "go.mondoo.io/mondoo/motor/transports/aws"
	"go.mondoo.io/mondoo/motor/transports/azure"
	"go.mondoo.io/mondoo/motor/transports/container"
	"go.mondoo.io/mondoo/motor/transports/equinix"
	"go.mondoo.io/mondoo/motor/transports/fs"
	"go.mondoo.io/mondoo/motor/transports/gcp"
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
)

func New(t *transports.TransportConfig, userIdDetectors ...string) (*motor.Motor, error) {
	return ResolveTransport(t, userIdDetectors...)
}

// ResolveTransport establishes a motor connection by using the provided transport configuration
// By default it uses the id detector mechanisms provided by the transport. User can overwride that
// behaviour by optionally passing id detector identifier
func ResolveTransport(tc *transports.TransportConfig, userIdDetectors ...string) (*motor.Motor, error) {
	var m *motor.Motor
	var name string
	var identifier []string
	var labels map[string]string
	idDetectors := []string{}

	switch tc.Backend {
	case transports.TransportBackend_CONNECTION_MOCK:
		log.Debug().Msg("connection> load mock transport")
		trans, err := mock.NewFromToml(tc)
		if err != nil {
			return nil, err
		}

		m, err = motor.New(trans)
		if err != nil {
			return nil, err
		}

		idDetectors = append(idDetectors, "machineid")
		idDetectors = append(idDetectors, "hostname")
	case transports.TransportBackend_CONNECTION_LOCAL_OS:
		log.Debug().Msg("connection> load local transport")
		trans, err := local.New()
		if err != nil {
			return nil, err
		}

		m, err = motor.New(trans, motor.WithRecoding(tc.Record))
		if err != nil {
			return nil, err
		}

		idDetectors = append(idDetectors, "machineid")
		idDetectors = append(idDetectors, "hostname")
	case transports.TransportBackend_CONNECTION_TAR:
		log.Debug().Msg("connection> load tar transport")
		trans, err := tar.New(tc)
		if err != nil {
			return nil, err
		}

		m, err = motor.New(trans, motor.WithRecoding(tc.Record))
		if err != nil {
			return nil, err
		}

		if len(trans.Identifier()) > 0 {
			identifier = append(identifier, trans.Identifier())
		}
	case transports.TransportBackend_CONNECTION_CONTAINER_REGISTRY:
		log.Debug().Msg("connection> load container registry transport")
		trans, err := container.NewContainerRegistryImage(tc)
		if err != nil {
			return nil, err
		}
		m, err = motor.New(trans, motor.WithRecoding(tc.Record))
		if err != nil {
			return nil, err
		}

		name = trans.PlatformName()
		labels = trans.Labels()
		if len(trans.Identifier()) > 0 {
			identifier = append(identifier, trans.Identifier())
		}
	case transports.TransportBackend_CONNECTION_DOCKER_ENGINE_CONTAINER:
		log.Debug().Msg("connection> load docker engine container transport")
		trans, err := container.NewDockerEngineContainer(tc)
		if err != nil {
			return nil, err
		}
		m, err = motor.New(trans, motor.WithRecoding(tc.Record))
		if err != nil {
			return nil, err
		}

		name = trans.PlatformName()
		labels = trans.Labels()
		if len(trans.Identifier()) > 0 {
			identifier = append(identifier, trans.Identifier())
		}
	case transports.TransportBackend_CONNECTION_DOCKER_ENGINE_IMAGE:
		log.Debug().Msg("connection> load docker engine image transport")
		trans, err := container.NewDockerEngineImage(tc)
		if err != nil {
			return nil, err
		}
		m, err = motor.New(trans, motor.WithRecoding(tc.Record))
		if err != nil {
			return nil, err
		}

		name = trans.PlatformName()
		labels = trans.Labels()
		if len(trans.Identifier()) > 0 {
			identifier = append(identifier, trans.Identifier())
		}
	case transports.TransportBackend_CONNECTION_SSH:
		log.Debug().Msg("connection> load ssh transport")
		trans, err := ssh.New(tc)
		if err != nil {
			return nil, err
		}

		m, err = motor.New(trans, motor.WithRecoding(tc.Record))
		if err != nil {
			return nil, err
		}

		idDetectors = append(idDetectors, "machineid")
		idDetectors = append(idDetectors, "hostname")
		idDetectors = append(idDetectors, "ssh-hostkey")
	case transports.TransportBackend_CONNECTION_WINRM:
		log.Debug().Msg("connection> load winrm transport")
		trans, err := winrm.New(tc)
		if err != nil {
			return nil, err
		}

		m, err = motor.New(trans, motor.WithRecoding(tc.Record))
		if err != nil {
			return nil, err
		}

		idDetectors = append(idDetectors, "machineid")
	case transports.TransportBackend_CONNECTION_VSPHERE:
		log.Debug().Msg("connection> load vsphere transport")
		trans, err := vsphere.New(tc)
		if err != nil {
			return nil, err
		}

		m, err = motor.New(trans)
		if err != nil {
			return nil, err
		}

		id, err := trans.Identifier()
		if err == nil && len(id) > 0 {
			identifier = append(identifier, id)
		}
	case transports.TransportBackend_CONNECTION_ARISTAEOS:
		log.Debug().Msg("connection> load arista eos transport")
		trans, err := arista.New(tc)
		if err != nil {
			return nil, err
		}
		m, err = motor.New(trans)
		if err != nil {
			return nil, err
		}

		id, err := trans.Identifier()
		if err == nil && len(id) > 0 {
			identifier = append(identifier, id)
		}
	case transports.TransportBackend_CONNECTION_AWS:
		log.Debug().Msg("connection> load aws transport")
		trans, err := aws_transport.New(tc, aws_transport.TransportOptions(tc.Options)...)
		if err != nil {
			return nil, err
		}
		m, err = motor.New(trans)
		if err != nil {
			return nil, err
		}

		id, err := trans.Identifier()
		if err == nil && len(id) > 0 {
			identifier = append(identifier, id)
		}
	case transports.TransportBackend_CONNECTION_GCP:
		log.Debug().Msg("connection> load gcp transport")
		trans, err := gcp.New(tc)
		if err != nil {
			return nil, err
		}
		m, err = motor.New(trans)
		if err != nil {
			return nil, err
		}

		id, err := trans.Identifier()
		if err == nil && len(id) > 0 {
			identifier = append(identifier, id)
		}
	case transports.TransportBackend_CONNECTION_AZURE:
		log.Debug().Msg("connection> load azure transport")
		trans, err := azure.New(tc)
		if err != nil {
			return nil, err
		}
		m, err = motor.New(trans)
		if err != nil {
			return nil, err
		}

		id, err := trans.Identifier()
		if err == nil && len(id) > 0 {
			identifier = append(identifier, id)
		}
	case transports.TransportBackend_CONNECTION_MS365:
		log.Debug().Msg("connection> load microsoft 365 transport")
		trans, err := ms365.New(tc)
		if err != nil {
			return nil, err
		}
		m, err = motor.New(trans)
		if err != nil {
			return nil, err
		}

		id, err := trans.Identifier()
		if err == nil && len(id) > 0 {
			identifier = append(identifier, id)
		}
	case transports.TransportBackend_CONNECTION_IPMI:
		log.Debug().Msg("connection> load ipmi transport")
		trans, err := ipmi.New(tc)
		if err != nil {
			return nil, err
		}
		m, err = motor.New(trans)
		if err != nil {
			return nil, err
		}

		id, err := trans.Identifier()
		if err == nil && len(id) > 0 {
			identifier = append(identifier, id)
		}
	case transports.TransportBackend_CONNECTION_VSPHERE_VM:
		trans, err := vmwareguestapi.New(tc)
		if err != nil {
			return nil, err
		}
		m, err = motor.New(trans, motor.WithRecoding(tc.Record))
		if err != nil {
			return nil, err
		}

		idDetectors = append(idDetectors, "machineid")
		idDetectors = append(idDetectors, "hostname")
	case transports.TransportBackend_CONNECTION_FS:
		trans, err := fs.New(tc)
		if err != nil {
			return nil, err
		}
		m, err = motor.New(trans, motor.WithRecoding(tc.Record))
		if err != nil {
			return nil, err
		}

		idDetectors = append(idDetectors, "machineid")
		idDetectors = append(idDetectors, "hostname")
	case transports.TransportBackend_CONNECTION_EQUINIX_METAL:
		trans, err := equinix.New(tc)
		if err != nil {
			return nil, err
		}
		m, err = motor.New(trans)
		if err != nil {
			return nil, err
		}
		id, err := trans.Identifier()
		if err == nil && len(id) > 0 {
			identifier = append(identifier, id)
		}
	case transports.TransportBackend_CONNECTION_K8S:
		trans, err := k8s_transport.New(tc)
		if err != nil {
			return nil, err
		}
		m, err = motor.New(trans)
		if err != nil {
			return nil, err
		}
		id, err := trans.Identifier()
		if err == nil && len(id) > 0 {
			identifier = append(identifier, id)
		}
	default:
		return nil, fmt.Errorf("connection> unsupported backend '%s', only docker://, local://, tar://, ssh:// are allowed", tc.Backend)
	}

	if len(userIdDetectors) > 0 {
		log.Info().Strs("id-detector", userIdDetectors).Msg("user provided platform detector ids")
		idDetectors = userIdDetectors
	}

	// some platforms are requiring ids only and have no id detector
	if len(idDetectors) > 0 {
		p, err := m.Platform()
		if err != nil {
			return nil, err
		}

		ids, err := motorid.GatherIDs(m.Transport, p, idDetectors)
		if err == nil {
			identifier = append(identifier, ids...)
		}
	}

	if len(identifier) == 0 {
		return nil, errors.New("could not find a valid platform identifier")
	}

	m.Meta = motor.ResolverMetadata{
		Name:       name,
		Identifier: identifier,
		Labels:     labels,
	}

	return m, nil
}
