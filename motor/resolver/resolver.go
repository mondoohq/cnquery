package resolver

import (
	"errors"

	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/motor"
	"go.mondoo.io/mondoo/motor/local"
	"go.mondoo.io/mondoo/motor/mock"
	"go.mondoo.io/mondoo/motor/types"
)

func New(endpoint *types.Endpoint) (*motor.Motor, error) {
	c := &motor.Motor{}

	trans, err := ResolveTransport(endpoint)
	if err != nil {
		return nil, errors.New("could not resolve backend " + err.Error())
	}
	c.Transport = trans

	return c, err
}

func NewFromUrl(uri string) (*motor.Motor, error) {
	t := &types.Endpoint{}
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
	default:
		return nil, errors.New("resolver> unsupported endpoint '" + endpoint.Backend + "'")
	}

	return trans, err
}
