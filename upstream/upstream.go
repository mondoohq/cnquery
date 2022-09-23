package upstream

import (
	"errors"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/ranger-rpc"
	guard_cert_auth "go.mondoo.com/ranger-rpc/plugins/authentication/cert"
	"go.mondoo.com/ranger-rpc/plugins/rangerguard/crypto"
)

//go:generate protoc --proto_path=../../:. --go_out=. --go_opt=paths=source_relative --rangerrpc_out=. upstream.proto

const agents_issuer = "mondoo/ams"

func NewServiceAccountRangerPlugin(credentials *ServiceAccountCredentials) (ranger.ClientPlugin, error) {
	if credentials == nil {
		return nil, errors.New("agent credentials must be set")
	}

	// verify that we can read the private key
	privateKey, err := crypto.PrivateKeyFromBytes([]byte(credentials.PrivateKey))
	if err != nil {
		return nil, errors.New("cannot load retrieved key: " + err.Error())
	}

	log.Debug().Str("kid", credentials.Mrn).Str("issuer", agents_issuer).Msg("initialize client authentication")

	// configure authentication plugin, since the server only accepts authenticated calls
	return guard_cert_auth.NewRangerPlugin(guard_cert_auth.ClientConfig{
		PrivateKey: privateKey,
		Issuer:     agents_issuer,
		Kid:        credentials.Mrn,
		Subject:    credentials.Mrn,
	})
}
