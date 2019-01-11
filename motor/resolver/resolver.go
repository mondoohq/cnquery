package resolver

import (
	"errors"

	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/motor"
	"go.mondoo.io/mondoo/motor/docker"
	"go.mondoo.io/mondoo/motor/local"
	"go.mondoo.io/mondoo/motor/mock"
	"go.mondoo.io/mondoo/motor/tar"
	"go.mondoo.io/mondoo/motor/types"
)

func New(endpoint *types.Endpoint) (*motor.Motor, error) {
	trans, err := ResolveTransport(endpoint)
	if err != nil {
		return nil, errors.New("could not resolve backend " + err.Error())
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
		log.Debug().Msg("resolver> load mock transport")
		trans, err = mock.New()
	case "local":
		log.Debug().Msg("resolver> load local transport")
		trans, err = local.New()
	case "tar":
		log.Debug().Msg("resolver> load tar transport")
		trans, err = tar.New(endpoint)
	case "docker":
		log.Debug().Msg("resolver> load docker transport")
		trans, err = docker.New(endpoint)
	default:
		return nil, errors.New("resolver> unsupported endpoint '" + endpoint.Backend + "'")
	}

	return trans, err
}
