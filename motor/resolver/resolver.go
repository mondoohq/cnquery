package resolver

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/motor"
	"go.mondoo.io/mondoo/motor/motorid/awsec2"
	"go.mondoo.io/mondoo/motor/motorid/hostname"
	"go.mondoo.io/mondoo/motor/motorid/machineid"
	"go.mondoo.io/mondoo/motor/platform"
	"go.mondoo.io/mondoo/motor/transports"
	"go.mondoo.io/mondoo/motor/transports/local"
	"go.mondoo.io/mondoo/motor/transports/mock"
	"go.mondoo.io/mondoo/motor/transports/ssh"
	"go.mondoo.io/mondoo/motor/transports/tar"
	"go.mondoo.io/mondoo/motor/transports/winrm"
	gossh "golang.org/x/crypto/ssh"
)

type EndpointOption func(endpoint *transports.Endpoint)

func WithIdentityFile(identityFile string) EndpointOption {
	return func(endpoint *transports.Endpoint) {
		endpoint.IdentityFiles = append(endpoint.IdentityFiles, identityFile)
	}
}

func WithPassword(password string) EndpointOption {
	return func(endpoint *transports.Endpoint) {
		endpoint.Password = password
	}
}

func WithSudo() EndpointOption {
	return func(endpoint *transports.Endpoint) {
		endpoint.Sudo = &transports.Sudo{
			Active: true,
		}
	}
}

func WithInsecure() EndpointOption {
	return func(endpoint *transports.Endpoint) {
		endpoint.Insecure = true
	}
}

func New(endpoint *transports.Endpoint, idDetectors ...string) (*motor.Motor, error) {
	return ResolveTransport(endpoint, idDetectors)
}

func NewFromUrl(uri string, opts ...EndpointOption) (*motor.Motor, error) {
	t := &transports.Endpoint{}
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
	t := &transports.Endpoint{
		IdentityFiles: []string{key},
	}
	err := t.ParseFromURI(uri)
	if err != nil {
		return nil, err
	}
	return New(t)
}

func ResolveTransport(endpoint *transports.Endpoint, idDetectors []string) (*motor.Motor, error) {
	var m *motor.Motor
	var name string
	var identifier []string
	var labels map[string]string
	var err error

	switch endpoint.Backend {
	case transports.BackendMock:
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
	case "nodejs":
		log.Debug().Msg("connection> load nodejs transport")
		// NOTE: while similar to local transport, the ids are completely different
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
	case transports.BackendLocal:
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
	case transports.BackendTAR:
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
	case transports.BackendDocker:
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
	case transports.BackendSSH:
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
	case transports.BackendWinrm:
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
	case "":
		return nil, errors.New("connection type is required, try `-t backend://` (docker://, local://, tar://, ssh://)")
	default:
		return nil, fmt.Errorf("connection> unsupported backend '%s', only docker://, local://, tar://, ssh:// are allowed", endpoint.Backend)
	}

	ids, err := GatherIDs(m, idDetectors)
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

func GatherIDs(m *motor.Motor, idDetectors []string) ([]string, error) {
	var ids []string
	for i := range idDetectors {
		if len(idDetectors[i]) == 0 {
			continue
		}
		id, err := GatherID(m, idDetectors[i])
		if err != nil {
			return nil, err
		}

		if len(id) > 0 {
			ids = append(ids, id)
		}
	}

	return ids, nil
}

func GatherID(m *motor.Motor, idDetector string) (string, error) {

	transport := m.Transport
	// helper for recoding transport to extract the original transport
	recT, ok := transport.(*mock.RecordTransport)
	if ok {
		transport = recT.Watched()
	}

	var identifier string
	switch idDetector {
	case "hostname":
		// NOTE: we need to be careful with hostname's since they are not required to be unique
		hostname, hostErr := hostname.Hostname(m)
		if hostErr == nil && len(hostname) > 0 {
			identifier = "//platformid.api.mondoo.app/hostname/" + hostname
		}
	case "machineid":
		guid, hostErr := machineid.MachineId(m)
		if hostErr == nil && len(guid) > 0 {
			identifier = "//platformid.api.mondoo.app/machineid/" + guid
		}
	case "ssh-hostkey":
		sshTrans, ok := transport.(*ssh.SSHTransport)
		if !ok {
			return "", errors.New("ssh-hostkey id detector is not supported for the transport")
		}
		if sshTrans != nil {
			fingerprint := gossh.FingerprintSHA256(sshTrans.HostKey)
			fingerprint = strings.Replace(fingerprint, ":", "-", 1)
			identifier = "//platformid.api.mondoo.app/runtime/ssh/hostkey/" + fingerprint
		}
	case "awsec2":
		_, ok := transport.(*local.LocalTransport)
		if ok {
			cfg, err := external.LoadDefaultAWSConfig()
			if err != nil {
				return "", errors.Wrap(err, "cannot not determine aws environment")
			}
			metadata := awsec2.NewLocal(cfg)
			mrn, err := metadata.InstanceID()
			if err != nil {
				return "", errors.Wrap(err, "cannot not determine aws ec2 instance id")
			}
			identifier = mrn
		} else {
			pf, err := m.Platform()
			if err != nil {
				return "", errors.Wrap(err, "could not determine platform")
			}

			if pf.IsFamily(platform.FAMILY_LINUX) {
				metadata := awsec2.NewUnix(m)
				mrn, err := metadata.InstanceID()
				if err != nil {
					return "", errors.Wrap(err, "cannot not determine aws ec2 instance id")
				}
				identifier = mrn
			} else {
				return "", errors.New(fmt.Sprintf("awsec2 id detector is not supported for your asset: %s %s", pf.Name, pf.Release))
			}
		}
	default:
		return "", errors.New(fmt.Sprintf("the provided id-detector is not supported: %s", idDetector))
	}
	return identifier, nil
}
