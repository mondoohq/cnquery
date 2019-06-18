package resolver

import (
	"errors"
	"strings"

	"github.com/rs/zerolog/log"
	motor "go.mondoo.io/mondoo/motor/motoros"
	"go.mondoo.io/mondoo/motor/motoros/local"
	"go.mondoo.io/mondoo/motor/motoros/mock"
	"go.mondoo.io/mondoo/motor/motoros/ssh"
	"go.mondoo.io/mondoo/motor/motoros/tar"
	"go.mondoo.io/mondoo/motor/motoros/types"
	"go.mondoo.io/mondoo/motor/motoros/winrm"
	gossh "golang.org/x/crypto/ssh"
)

func New(endpoint *types.Endpoint) (*motor.Motor, string, error) {
	trans, identifier, err := ResolveTransport(endpoint)
	if err != nil {
		return nil, "", err
	}
	m, err := motor.New(trans)
	return m, identifier, err
}

func NewFromUrl(uri string) (*motor.Motor, string, error) {
	t := &types.Endpoint{}
	err := t.ParseFromURI(uri)
	if err != nil {
		return nil, "", err
	}
	return New(t)
}

func NewWithUrlAndKey(uri string, key string) (*motor.Motor, string, error) {
	t := &types.Endpoint{
		PrivateKeyPath: key,
	}
	err := t.ParseFromURI(uri)
	if err != nil {
		return nil, "", err
	}
	return New(t)
}

func ResolveTransport(endpoint *types.Endpoint) (types.Transport, string, error) {
	var err error
	var identifier string

	var trans types.Transport
	switch endpoint.Backend {
	case "mock":
		log.Debug().Msg("connection> load mock transport")
		trans, err = mock.New()
	case "local", "nodejs":
		log.Debug().Msg("connection> load local transport")
		// TODO: we need to generate an artifact id
		trans, err = local.New()
	case "tar":
		log.Debug().Msg("connection> load tar transport")
		// TODO: we need to generate an artifact id
		trans, err = tar.New(endpoint)
	case "docker":
		log.Debug().Msg("connection> load docker transport")
		trans, identifier, err = ResolveDockerTransport(endpoint)
	case "ssh":
		log.Debug().Msg("connection> load ssh transport")
		sshTrans, sshErr := ssh.New(endpoint)
		if sshErr == nil && sshTrans != nil {
			fingerprint := gossh.FingerprintSHA256(sshTrans.HostKey)
			fingerprint = strings.Replace(fingerprint, ":", "-", 1)
			identifier = "//sytemidentifier.api.mondoo.app/runtime/ssh/hostkey/" + fingerprint
		}
		trans = sshTrans
		err = sshErr
	case "winrm":
		log.Debug().Msg("connection> load winrm transport")
		trans, err = winrm.New(endpoint)
	case "":
		return nil, "", errors.New("connection type is required, try `-t backend://` (docker://, local://, tar://, ssh://)")
	default:
		return nil, "", errors.New("connection> unsupported backend, only docker://, local://, tar://, ssh:// are allowed'" + endpoint.Backend + "'")
	}

	return trans, identifier, err
}
