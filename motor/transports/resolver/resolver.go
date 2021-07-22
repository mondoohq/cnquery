package resolver

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"

	"go.mondoo.io/mondoo/motor/motorid/containerid"

	"github.com/cockroachdb/errors"
	"go.mondoo.io/mondoo/motor/transports/equinix"
	"go.mondoo.io/mondoo/motor/transports/fs"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/motor"
	docker_discovery "go.mondoo.io/mondoo/motor/discovery/docker_engine"
	"go.mondoo.io/mondoo/motor/motorid"
	"go.mondoo.io/mondoo/motor/transports"
	"go.mondoo.io/mondoo/motor/transports/arista"
	aws_transport "go.mondoo.io/mondoo/motor/transports/aws"
	"go.mondoo.io/mondoo/motor/transports/azure"
	"go.mondoo.io/mondoo/motor/transports/docker/docker_engine"
	"go.mondoo.io/mondoo/motor/transports/docker/image"
	"go.mondoo.io/mondoo/motor/transports/docker/snapshot"
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
		trans, info, err := containerregistry(tc)
		if err != nil {
			return nil, err
		}
		m, err = motor.New(trans, motor.WithRecoding(tc.Record))
		if err != nil {
			return nil, err
		}

		name = info.Name
		labels = info.Labels

		// TODO: can we make the id optional here, we may want to use an approach that is similar to ssh
		if len(info.Identifier) > 0 {
			identifier = append(identifier, info.Identifier)
		}
	case transports.TransportBackend_CONNECTION_DOCKER_ENGINE_CONTAINER:
		log.Debug().Msg("connection> load docker engine container transport")
		trans, info, err := dockerenginecontainer(tc)
		if err != nil {
			return nil, err
		}
		m, err = motor.New(trans, motor.WithRecoding(tc.Record))
		if err != nil {
			return nil, err
		}

		name = info.Name
		labels = info.Labels

		// TODO: can we make the id optional here, we may want to use an approach that is similar to ssh
		if len(info.Identifier) > 0 {
			identifier = append(identifier, info.Identifier)
		}
	case transports.TransportBackend_CONNECTION_DOCKER_ENGINE_IMAGE:
		log.Debug().Msg("connection> load docker engine image transport")
		trans, info, err := dockerengineimage(tc)
		if err != nil {
			return nil, err
		}
		m, err = motor.New(trans, motor.WithRecoding(tc.Record))
		if err != nil {
			return nil, err
		}

		name = info.Name
		labels = info.Labels

		// TODO: can we make the id optional here, we may want to use an approach that is similar to ssh
		if len(info.Identifier) > 0 {
			identifier = append(identifier, info.Identifier)
		}
	case transports.TransportBackend_CONNECTION_DOCKER_ENGINE_TAR:
		log.Debug().Msg("connection> load docker tar transport")
		trans, info, err := containertar(tc)
		if err != nil {
			return nil, err
		}
		m, err = motor.New(trans, motor.WithRecoding(tc.Record))
		if err != nil {
			return nil, err
		}

		name = info.Name
		labels = info.Labels

		// TODO: can we make the id optional here, we may want to use an approach that is similar to ssh
		if len(info.Identifier) > 0 {
			identifier = append(identifier, info.Identifier)
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

// TODO: move individual docker handling to specific transport
type DockerInfo struct {
	Name       string
	Identifier string
	Labels     map[string]string
}

func containerregistry(tc *transports.TransportConfig) (transports.Transport, DockerInfo, error) {
	// load container image from remote directoryload tar file into backend
	ref, err := name.ParseReference(tc.Host, name.WeakValidation)
	if err == nil {
		log.Debug().Str("ref", ref.Name()).Msg("found valid container registry reference")

		registryOpts := []image.Option{image.WithInsecure(tc.Insecure)}
		if len(tc.BearerToken) > 0 {
			log.Debug().Msg("enable bearer authentication for image")
			registryOpts = append(registryOpts, image.WithAuthenticator(&authn.Bearer{Token: tc.BearerToken}))
		}

		// image.WithAuthenticator()
		img, rc, err := image.LoadFromRegistry(ref, registryOpts...)
		if err != nil {
			return nil, DockerInfo{}, err
		}

		var identifier string
		hash, err := img.Digest()
		if err == nil {
			identifier = containerid.MondooContainerImageID(hash.String())
		}

		transport, err := image.New(rc)
		return transport, DockerInfo{
			Name:       containerid.ShortContainerImageID(hash.String()),
			Identifier: identifier,
		}, err
	}
	log.Debug().Str("image", tc.Host).Msg("Could not detect a valid repository url")
	return nil, DockerInfo{}, err
}

func dockerenginecontainer(tc *transports.TransportConfig) (transports.Transport, DockerInfo, error) {
	// could be an image id/name, container id/name or a short reference to an image in docker engine
	ded, err := docker_discovery.NewDockerEngineDiscovery()
	if err != nil {
		return nil, DockerInfo{}, err
	}

	ci, err := ded.ContainerInfo(tc.Host)
	if err != nil {
		return nil, DockerInfo{}, err
	}

	if ci.Running {
		log.Debug().Msg("found running container " + ci.ID)
		transport, err := docker_engine.New(ci.ID)
		return transport, DockerInfo{
			Name:       containerid.ShortContainerImageID(ci.ID),
			Identifier: containerid.MondooContainerID(ci.ID),
			Labels:     ci.Labels,
		}, err
	} else {
		log.Debug().Msg("found stopped container " + ci.ID)
		transport, err := snapshot.NewFromDockerEngine(ci.ID)
		return transport, DockerInfo{
			Name:       containerid.ShortContainerImageID(ci.ID),
			Identifier: containerid.MondooContainerID(ci.ID),
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
		identifier = containerid.MondooContainerImageID(hash.String())
	}

	transport, err := image.New(rc)
	return transport, DockerInfo{
		Name:       ii.Name,
		Identifier: identifier,
		Labels:     ii.Labels,
	}, err
}

// check if the tar is an image or container
func containertar(endpoint *transports.TransportConfig) (transports.Transport, DockerInfo, error) {
	log.Debug().Msg("found local docker/image file")

	// try to load docker image tarball
	img, err := tarball.ImageFromPath(endpoint.Host, nil)
	if err == nil {
		log.Debug().Msg("detected docker image")
		var identifier string

		hash, err := img.Digest()
		if err == nil {
			identifier = containerid.MondooContainerImageID(hash.String())
		} else {
			log.Warn().Err(err).Msg("could not determine platform id")
		}

		rc := mutate.Extract(img)
		transport, err := image.New(rc)
		return transport, DockerInfo{
			Identifier: identifier,
		}, err
	}

	log.Debug().Msg("detected docker container snapshot")

	// generate sha sum of tar file
	f, err := os.Open(endpoint.Host)
	if err != nil {
		return nil, DockerInfo{}, errors.Wrap(err, "cannot read container tar to generate hash")
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return nil, DockerInfo{}, errors.Wrap(err, "cannot read container tar to generate hash")
	}

	hash := hex.EncodeToString(h.Sum(nil))

	transport, err := snapshot.NewFromFile(endpoint.Host)
	return transport, DockerInfo{
		Identifier: "//platformid.api.mondoo.app/runtime/docker/snapshot/" + hash,
	}, err
}
