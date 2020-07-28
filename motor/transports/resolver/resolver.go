package resolver

import (
	"fmt"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/motor"
	docker_discovery "go.mondoo.io/mondoo/motor/discovery/docker_engine"
	"go.mondoo.io/mondoo/motor/motorid"
	"go.mondoo.io/mondoo/motor/platform"
	"go.mondoo.io/mondoo/motor/transports"
	"go.mondoo.io/mondoo/motor/transports/arista"
	"go.mondoo.io/mondoo/motor/transports/docker/docker_engine"
	"go.mondoo.io/mondoo/motor/transports/docker/image"
	"go.mondoo.io/mondoo/motor/transports/docker/snapshot"
	"go.mondoo.io/mondoo/motor/transports/local"
	"go.mondoo.io/mondoo/motor/transports/mock"
	"go.mondoo.io/mondoo/motor/transports/ssh"
	"go.mondoo.io/mondoo/motor/transports/tar"
	"go.mondoo.io/mondoo/motor/transports/vsphere"
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
	case transports.TransportBackend_CONNECTION_CONTAINER_TAR:
		trans, info, err := containertar(endpoint)
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
	case transports.TransportBackend_CONNECTION_CONTAINER_REGISTRY:
		trans, info, err := containerregistry(endpoint)
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
	case transports.TransportBackend_CONNECTION_DOCKER_ENGINE_CONTAINER:
		trans, info, err := dockerenginecontainer(endpoint)
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
	case transports.TransportBackend_CONNECTION_DOCKER_ENGINE_IMAGE:
		trans, info, err := dockerengineimage(endpoint)
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
	case transports.TransportBackend_CONNECTION_VSPHERE:
		log.Debug().Msg("connection> load vsphere transport")
		trans, err := vsphere.New(endpoint)
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

		id, err := trans.Identifier()
		if err == nil && len(id) > 0 {
			identifier = append(identifier, id)
		}
	case transports.TransportBackend_CONNECTION_ARISTAEOS:
		log.Debug().Msg("connection> load aristaeos transport")
		trans, err := arista.New(endpoint)
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
	default:
		return nil, fmt.Errorf("connection> unsupported backend '%s', only docker://, local://, tar://, ssh:// are allowed", endpoint.Backend)
	}

	p, err := m.Platform()
	if err != nil {
		return nil, err
	}

	ids, err := motorid.GatherIDs(m.Transport, p, idDetectors)
	if err == nil {
		identifier = append(identifier, ids...)
	} else {
		log.Error().Err(err).Msg("could not gather the requested platform identifier")
	}

	m.Meta = motor.MetaInfo{
		Name:       name,
		Identifier: identifier,
		Labels:     labels,
	}

	return m, err
}

// TODO: move individual docker handling to specific transport
type DockerInfo struct {
	Name       string
	Identifier string
	Labels     map[string]string
}

func containerregistry(endpoint *transports.TransportConfig) (transports.Transport, DockerInfo, error) {
	// load container image from remote directoryload tar file into backend
	ref, err := name.ParseReference(endpoint.Host, name.WeakValidation)
	if err == nil {
		log.Debug().Str("ref", ref.Name()).Msg("found valid container registry reference")

		registryOpts := []image.Option{image.WithInsecure(endpoint.Insecure)}
		if len(endpoint.BearerToken) > 0 {
			log.Debug().Msg("enable bearer authentication for image")
			registryOpts = append(registryOpts, image.WithAuthenticator(&authn.Bearer{Token: endpoint.BearerToken}))
		}

		// image.WithAuthenticator()
		img, rc, err := image.LoadFromRegistry(ref, registryOpts...)
		if err != nil {
			return nil, DockerInfo{}, err
		}

		var identifier string
		hash, err := img.Digest()
		if err == nil {
			identifier = docker_discovery.MondooContainerImageID(hash.String())
		}

		transport, err := image.New(rc)
		return transport, DockerInfo{
			Name:       docker_discovery.ShortContainerImageID(hash.String()),
			Identifier: identifier,
		}, err
	}
	log.Debug().Str("image", endpoint.Host).Msg("Could not detect a valid repository url")
	return nil, DockerInfo{}, err
}

func dockerenginecontainer(endpoint *transports.TransportConfig) (transports.Transport, DockerInfo, error) {
	// could be an image id/name, container id/name or a short reference to an image in docker engine
	ded, err := docker_discovery.NewDockerEngineDiscovery()
	if err != nil {
		return nil, DockerInfo{}, err
	}

	ci, err := ded.ContainerInfo(endpoint.Host)
	if err != nil {
		return nil, DockerInfo{}, err
	}

	if ci.Running {
		log.Debug().Msg("found running container " + ci.ID)
		transport, err := docker_engine.New(ci.ID)
		return transport, DockerInfo{
			Name:       docker_discovery.ShortContainerImageID(ci.ID),
			Identifier: docker_discovery.MondooContainerID(ci.ID),
			Labels:     ci.Labels,
		}, err
	} else {
		log.Debug().Msg("found stopped container " + ci.ID)
		transport, err := snapshot.NewFromDockerEngine(ci.ID)
		return transport, DockerInfo{
			Name:       docker_discovery.ShortContainerImageID(ci.ID),
			Identifier: docker_discovery.MondooContainerID(ci.ID),
			Labels:     ci.Labels,
		}, err
	}
}

func dockerengineimage(endpoint *transports.TransportConfig) (transports.Transport, DockerInfo, error) {
	// could be an image id/name, container id/name or a short reference to an image in docker engine
	ded, err := docker_discovery.NewDockerEngineDiscovery()
	if err != nil {
		return nil, DockerInfo{}, err
	}

	ii, err := ded.ImageInfo(endpoint.Host)
	if err != nil {
		return nil, DockerInfo{}, err
	}

	log.Debug().Msg("found docker engine image " + ii.ID)
	img, rc, err := image.LoadFromDockerEngine(ii.ID)
	if err != nil {
		return nil, DockerInfo{}, err
	}

	var identifier string
	hash, err := img.Digest()
	if err == nil {
		identifier = docker_discovery.MondooContainerImageID(hash.String())
	}

	transport, err := image.New(rc)
	return transport, DockerInfo{
		Name:       ii.Name,
		Identifier: identifier,
		Labels:     ii.Labels,
	}, err
}

func containertar(endpoint *transports.TransportConfig) (transports.Transport, DockerInfo, error) {
	log.Debug().Msg("found local docker/image file")

	// try to load docker image tarball
	img, err := tarball.ImageFromPath(endpoint.Host, nil)
	if err == nil {
		log.Debug().Msg("detected docker image")
		var identifier string

		hash, err := img.Digest()
		if err == nil {
			identifier = docker_discovery.MondooContainerImageID(hash.String())
		} else {
			log.Warn().Err(err).Msg("could not determine referenceid")
		}

		rc := mutate.Extract(img)
		transport, err := image.New(rc)
		return transport, DockerInfo{
			Identifier: identifier,
		}, err
	}

	log.Debug().Msg("detected docker container snapshot")
	transport, err := snapshot.NewFromFile(endpoint.Host)
	return transport, DockerInfo{}, err
}
