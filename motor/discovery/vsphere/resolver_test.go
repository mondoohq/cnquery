package vsphere

import (
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/vmware/govmomi/simulator"

	"go.mondoo.io/mondoo/motor/vault"

	"go.mondoo.io/mondoo/motor/transports"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVsphereResolver(t *testing.T) {
	// start vsphere simulator
	// see https://pkg.go.dev/github.com/vmware/govmomi/simulator#pkg-overview
	const username = "my-username"
	const password = "my-password"
	model := simulator.VPX()
	defer model.Remove()
	err := model.Create()
	require.NoError(t, err)
	model.Service.Listen = &url.URL{
		User: url.UserPassword(username, password),
	}

	// use the httptest tls generation instead of writing our own
	tlsSrv := httptest.NewTLSServer(nil)
	tls := tlsSrv.TLS
	tlsSrv.Close()
	model.Service.TLS = tls
	s := model.Service.NewServer()
	defer s.Close()

	// start vsphere discover
	r := Resolver{}
	assets, err := r.Resolve(&transports.TransportConfig{
		Backend:  transports.TransportBackend_CONNECTION_VSPHERE,
		Host:     s.URL.Hostname(),
		Port:     s.URL.Port(),
		Insecure: true, // allows self-signed certificates
		Credentials: []*vault.Credential{
			{
				Type:   vault.CredentialType_password,
				User:   username,
				Secret: []byte(password),
			},
		},
		Discover: &transports.Discovery{
			Targets: []string{"all"},
		},
	}, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, 9, len(assets)) // api + esx + vm
}
