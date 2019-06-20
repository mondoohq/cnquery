package resolver

import (
	"errors"
	"strings"

	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/motor/motorid/hostname"
	motor "go.mondoo.io/mondoo/motor/motoros"
	"go.mondoo.io/mondoo/motor/motoros/local"
	"go.mondoo.io/mondoo/motor/motoros/mock"
	"go.mondoo.io/mondoo/motor/motoros/ssh"
	"go.mondoo.io/mondoo/motor/motoros/tar"
	"go.mondoo.io/mondoo/motor/motoros/types"
	"go.mondoo.io/mondoo/motor/motoros/winrm"
	gossh "golang.org/x/crypto/ssh"
)

func New(endpoint *types.Endpoint) (*motor.Motor, []string, error) {
	m, identifier, err := ResolveTransport(endpoint)
	if err != nil {
		return nil, nil, err
	}
	return m, identifier, err
}

func NewFromUrl(uri string) (*motor.Motor, []string, error) {
	t := &types.Endpoint{}
	err := t.ParseFromURI(uri)
	if err != nil {
		return nil, nil, err
	}
	return New(t)
}

func NewWithUrlAndKey(uri string, key string) (*motor.Motor, []string, error) {
	t := &types.Endpoint{
		PrivateKeyPath: key,
	}
	err := t.ParseFromURI(uri)
	if err != nil {
		return nil, nil, err
	}
	return New(t)
}

func ResolveTransport(endpoint *types.Endpoint) (*motor.Motor, []string, error) {
	var m *motor.Motor
	var identifier []string
	var err error

	switch endpoint.Backend {
	case "mock":
		log.Debug().Msg("connection> load mock transport")
		trans, err := mock.New()
		if err != nil {
			return nil, nil, err
		}

		m, err = motor.New(trans)
		if err != nil {
			return nil, nil, err
		}
	case "nodejs":
		log.Debug().Msg("connection> load nodejs transport")
		// NOTE: while similar to local transport, the ids are completely different
		trans, err := local.New()
		if err != nil {
			return nil, nil, err
		}

		m, err = motor.New(trans)
		if err != nil {
			return nil, nil, err
		}
	case "local":
		log.Debug().Msg("connection> load local transport")
		trans, err := local.New()
		if err != nil {
			return nil, nil, err
		}

		m, err = motor.New(trans)
		if err != nil {
			return nil, nil, err
		}

		// NOTE: we need to be careful with hostname's since they are not required to be unique
		hostname, hostErr := hostname.Hostname(m)
		if hostErr == nil && len(hostname) > 0 {
			identifier = append(identifier, "//platformid.api.mondoo.app/hostname/"+hostname)
		}
	case "tar":
		log.Debug().Msg("connection> load tar transport")
		// TODO: we need to generate an artifact id
		trans, err := tar.New(endpoint)
		if err != nil {
			return nil, nil, err
		}

		m, err = motor.New(trans)
		if err != nil {
			return nil, nil, err
		}
	case "docker":
		log.Debug().Msg("connection> load docker transport")
		var id string
		trans, id, err := ResolveDockerTransport(endpoint)
		if err != nil {
			return nil, nil, err
		}
		identifier = append(identifier, id)
		m, err = motor.New(trans)
		if err != nil {
			return nil, nil, err
		}
	case "ssh":
		log.Debug().Msg("connection> load ssh transport")
		trans, sshErr := ssh.New(endpoint)
		if sshErr != nil {
			return nil, nil, err
		}
		if sshErr == nil && trans != nil {
			fingerprint := gossh.FingerprintSHA256(trans.HostKey)
			fingerprint = strings.Replace(fingerprint, ":", "-", 1)
			identifier = append(identifier, "//platformid.api.mondoo.app/runtime/ssh/hostkey/"+fingerprint)
		}

		m, err = motor.New(trans)
		if err != nil {
			return nil, nil, err
		}
	case "winrm":
		log.Debug().Msg("connection> load winrm transport")
		trans, err := winrm.New(endpoint)
		if err != nil {
			return nil, nil, err
		}

		m, err = motor.New(trans)
		if err != nil {
			return nil, nil, err
		}

		// NOTE: we need to be careful with hostname's since they are not required to be unique
		hostname, hostErr := hostname.Hostname(m)
		if hostErr == nil && len(hostname) > 0 {
			identifier = append(identifier, "//platformid.api.mondoo.app/hostname/"+hostname)
		}
	case "":
		return nil, nil, errors.New("connection type is required, try `-t backend://` (docker://, local://, tar://, ssh://)")
	default:
		return nil, nil, errors.New("connection> unsupported backend, only docker://, local://, tar://, ssh:// are allowed'" + endpoint.Backend + "'")
	}

	return m, identifier, err
}
