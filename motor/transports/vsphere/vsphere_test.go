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

func TestVSphereTransport(t *testing.T) {
	vs, err := vsimulator.New()
	require.NoError(t, err)
	defer vs.Close()

	port := vs.Server.URL.Port()
	portNum, err := strconv.Atoi(port)
	require.NoError(t, err)

	trans, err := New(&transports.TransportConfig{
		Backend:  transports.TransportBackend_CONNECTION_VSPHERE,
		Host:     vs.Server.URL.Hostname(),
		Port:     int32(portNum),
		Insecure: true, // allows self-signed certificates
		Credentials: []*vault.Credential{
			{
				Type:   vault.CredentialType_password,
				User:   vsimulator.Username,
				Secret: []byte(vsimulator.Password),
			},
		},
	})
	require.NoError(t, err)

	ver := trans.Client().ServiceContent.About
	assert.Equal(t, "6.5", ver.ApiVersion)
}
