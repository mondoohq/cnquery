package resolver

import (
	"errors"

	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/motor"
	"go.mondoo.io/mondoo/motor/docker"
	"go.mondoo.io/mondoo/motor/local"
	"go.mondoo.io/mondoo/motor/mock"
	"go.mondoo.io/mondoo/motor/ssh"
	"go.mondoo.io/mondoo/motor/tar"
	"go.mondoo.io/mondoo/motor/types"
	"go.mondoo.io/mondoo/motor/winrm"
)

func New(endpoint *types.Endpoint) (*motor.Motor, error) {
	trans, err := ResolveTransport(endpoint)
	if err != nil {
		return nil, err
	}
	return motor.New(trans)
}

func NewFromUrl(uri string) (*motor.Motor, error) {
	t := &types.Endpoint{}
	err := t.ParseFromURI(uri)
	if err != nil {
		return nil, err
	}
	return New(t)
}

func NewWithUrlAndKey(uri string, key string) (*motor.Motor, error) {
	t := &types.Endpoint{
		PrivateKeyPath: key,
	}
	err := t.ParseFromURI(uri)
	if err != nil {
		return nil, err
	}
	return New(t)
}

func ResolveTransport(endpoint *types.Endpoint) (types.Transport, error) {
	var err error

	var trans types.Transport
	switch endpoint.Backend {
	case "mock":
		log.Debug().Msg("connection> load mock transport")
		trans, err = mock.New()
	case "local":
		log.Debug().Msg("connection> load local transport")
		trans, err = local.New()
	case "tar":
		log.Debug().Msg("connection> load tar transport")
		trans, err = tar.New(endpoint)
	case "docker":
		log.Debug().Msg("connection> load docker transport")
		trans, err = docker.New(endpoint)
	case "ssh":
		log.Debug().Msg("connection> load ssh transport")
		trans, err = ssh.New(endpoint)
	case "winrm":
		log.Debug().Msg("connection> load winrm transport")
		trans, err = winrm.New(endpoint)
	case "":
		return nil, errors.New("connection type is required, try `-t backend://` (docker://, local://, tar://, ssh://)")
	default:
		return nil, errors.New("connection> unsupported backend, only docker://, local://, tar://, ssh:// are allowed'" + endpoint.Backend + "'")
	}

	return trans, err
}
