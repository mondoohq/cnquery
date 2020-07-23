package resolver

import (
	"fmt"

	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/motor"
	"go.mondoo.io/mondoo/motor/motorid"
	"go.mondoo.io/mondoo/motor/platform"
	"go.mondoo.io/mondoo/motor/transports"
	"go.mondoo.io/mondoo/motor/transports/local"
	"go.mondoo.io/mondoo/motor/transports/mock"
	"go.mondoo.io/mondoo/motor/transports/ssh"
	"go.mondoo.io/mondoo/motor/transports/tar"
	"go.mondoo.io/mondoo/motor/transports/winrm"
)

type EndpointOption func(endpoint *transports.TransportConfig)

func WithIdentityFile(identityFile string) EndpointOption {
	return func(endpoint *transports.TransportConfig) {
		endpoint.IdentityFiles = append(endpoint.IdentityFiles, identityFile)
	}
}

func WithPassword(password string) EndpointOption {
	return func(endpoint *transports.TransportConfig) {
		endpoint.Password = password
	}
}

func WithSudo() EndpointOption {
	return func(endpoint *transports.TransportConfig) {
		endpoint.Sudo = &transports.Sudo{
			Active: true,
		}
	}
}

func WithInsecure() EndpointOption {
	return func(endpoint *transports.TransportConfig) {
		endpoint.Insecure = true
	}
}

func New(endpoint *transports.TransportConfig, idDetectors ...string) (*motor.Motor, error) {
	return ResolveTransport(endpoint, idDetectors)
}

func NewFromUrl(uri string, opts ...EndpointOption) (*motor.Motor, error) {
	t := &transports.TransportConfig{}
	err := t.ParseFromURI(uri)
	if err != nil {
		return nil, err
	}

	for i := range opts {
		opts[i](t)
	}
	return New(t)
}

func NewWithUrlAndKey(uri string, key string) (*motor.Motor, error) {
	t := &transports.TransportConfig{
		IdentityFiles: []string{key},
	}
	err := t.ParseFromURI(uri)
	if err != nil {
		return nil, err
	}
	return New(t)
}

func ResolveTransport(endpoint *transports.TransportConfig, idDetectors []string) (*motor.Motor, error) {
	var m *motor.Motor
	var name string
	var identifier []string
	var labels map[string]string
	var err error

	switch endpoint.Backend {
	case transports.TransportBackend_CONNECTION_MOCK:
		log.Debug().Msg("connection> load mock transport")
		trans, err := mock.NewFromToml(endpoint)
		if err != nil {
			return nil, err
		}

		m, err = motor.New(trans)
		if err != nil {
			return nil, err
		}
		if endpoint.Record {
			m.ActivateRecorder()
		}
	// case "nodejs":
	// 	log.Debug().Msg("connection> load nodejs transport")
	// 	// NOTE: while similar to local transport, the ids are completely different
	// 	trans, err := local.New()
	// 	if err != nil {
	// 		return nil, err
	// 	}

	// 	m, err = motor.New(trans)
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// 	if endpoint.Record {
	// 		m.ActivateRecorder()
	// 	}
	case transports.TransportBackend_CONNECTION_LOCAL_OS:
		log.Debug().Msg("connection> load local transport")
		trans, err := local.New()
		if err != nil {
			return nil, err
		}

		m, err = motor.New(trans)
		if err != nil {
			return nil, err
		}

		if endpoint.Record {
			m.ActivateRecorder()
		}

		pi, err := m.Platform()
		if err == nil && pi.IsFamily(platform.FAMILY_WINDOWS) {
			idDetectors = append(idDetectors, "machineid")
		} else {
			idDetectors = append(idDetectors, "hostname")
		}
	case transports.TransportBackend_CONNECTION_TAR:
		log.Debug().Msg("connection> load tar transport")
		// TODO: we need to generate an artifact id
		trans, err := tar.New(endpoint)
		if err != nil {
			return nil, err
		}

		m, err = motor.New(trans)
		if err != nil {
			return nil, err
		}

		if endpoint.Record {
			m.ActivateRecorder()
		}
	case transports.TransportBackend_CONNECTION_DOCKER_CONTAINER:
		fallthrough
	case transports.TransportBackend_CONNECTION_DOCKER_REGISTRY:
		fallthrough
	case transports.TransportBackend_CONNECTION_DOCKER_IMAGE:
		log.Debug().Str("backend", endpoint.Backend.String()).Str("host", endpoint.Host).Str("path", endpoint.Path).Msg("connection> load docker transport")
		trans, info, err := ResolveDockerTransport(endpoint)
		if err != nil {
			return nil, err
		}
		m, err = motor.New(trans)
		if err != nil {
			return nil, err
		}

		if endpoint.Record {
			m.ActivateRecorder()
		}

		name = info.Name
		labels = info.Labels

		// TODO: can we make the id optional here, we may want to use an approach that is similar to ssh
		if len(info.Identifier) > 0 {
			identifier = append(identifier, info.Identifier)
		}
	case transports.TransportBackend_CONNECTION_SSH:
		log.Debug().Msg("connection> load ssh transport")
		trans, err := ssh.New(endpoint)
		if err != nil {
			return nil, err
		}

		m, err = motor.New(trans)
		if err != nil {
			return nil, err
		}

		if endpoint.Record {
			m.ActivateRecorder()
		}

		// for windows, we also collect the machine id
		pi, err := m.Platform()
		if err == nil && pi.IsFamily(platform.FAMILY_WINDOWS) {
			idDetectors = append(idDetectors, "machineid")
		}

		idDetectors = append(idDetectors, "ssh-hostkey")
	case transports.TransportBackend_CONNECTION_WINRM:
		log.Debug().Msg("connection> load winrm transport")
		trans, err := winrm.New(endpoint)
		if err != nil {
			return nil, err
		}

		m, err = motor.New(trans)
		if err != nil {
			return nil, err
		}

		if endpoint.Record {
			m.ActivateRecorder()
		}

		idDetectors = append(idDetectors, "machineid")
	default:
		return nil, fmt.Errorf("connection> unsupported backend '%s', only docker://, local://, tar://, ssh:// are allowed", endpoint.Backend)
	}

	p, err := m.Platform()
	if err != nil {
		return nil, err
	}

	ids, err := motorid.GatherIDs(m.Transport, p, idDetectors)
	if err != nil {
		log.Error().Err(err).Msg("could not gather the requested platform identifier")
	} else {
		identifier = append(identifier, ids...)
	}

	m.Meta = motor.MetaInfo{
		Name:       name,
		Identifier: identifier,
		Labels:     labels,
	}

	return m, err
}
