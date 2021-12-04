package vsphere

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.io/mondoo/motor/transports"
	"go.mondoo.io/mondoo/motor/transports/vsphere/vsimulator"
	"go.mondoo.io/mondoo/motor/vault"
)

func TestVsphereResolver(t *testing.T) {
	vs, err := vsimulator.New()
	require.NoError(t, err)
	defer vs.Close()

	port, err := strconv.Atoi(vs.Server.URL.Port())
	require.NoError(t, err)

	// start vsphere discover
	r := Resolver{}
	assets, err := r.Resolve(&transports.TransportConfig{
		Backend:  transports.TransportBackend_CONNECTION_VSPHERE,
		Host:     vs.Server.URL.Hostname(),
		Port:     int32(port),
		Insecure: true, // allows self-signed certificates
		Credentials: []*vault.Credential{
			{
				Type:   vault.CredentialType_password,
				User:   vsimulator.Username,
				Secret: []byte(vsimulator.Password),
			},
		},
		Discover: &transports.Discovery{
			Targets: []string{"all"},
		},
	}, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, 9, len(assets)) // api + esx + vm
}
