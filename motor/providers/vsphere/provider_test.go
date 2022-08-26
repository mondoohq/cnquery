package vsphere

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/motor/providers"
	"go.mondoo.com/cnquery/motor/providers/vsphere/vsimulator"
	"go.mondoo.com/cnquery/motor/vault"
)

func TestVSphereTransport(t *testing.T) {
	vs, err := vsimulator.New()
	require.NoError(t, err)
	defer vs.Close()

	port := vs.Server.URL.Port()
	portNum, err := strconv.Atoi(port)
	require.NoError(t, err)

	p, err := New(&providers.Config{
		Backend:  providers.ProviderType_VSPHERE,
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

	ver := p.Client().ServiceContent.About
	assert.Equal(t, "6.5", ver.ApiVersion)
}
