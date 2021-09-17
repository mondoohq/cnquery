package vsphere

import (
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
	s := model.Service.NewServer()
	defer s.Close()

	// start vsphere discover
	r := Resolver{}
	assets, err := r.Resolve(&transports.TransportConfig{
		Backend: transports.TransportBackend_CONNECTION_VSPHERE,
		Host:    s.URL.Hostname(),
		Port:    s.URL.Port(),

		Credentials: []*vault.Credential{
			{
				Type:   vault.CredentialType_password,
				User:   username,
				Secret: []byte(password),
			},
		},
		Options: map[string]string{
			"protocol": "http", // only required for testing
		},
		Discover: &transports.Discovery{
			Targets: []string{"all"},
		},
	}, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, 9, len(assets)) // api + esx + vm
}
