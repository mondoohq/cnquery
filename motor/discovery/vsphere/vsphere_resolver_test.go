package vsphere

import (
	"context"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.io/mondoo/motor/asset"
	"go.mondoo.io/mondoo/motor/providers"
	"go.mondoo.io/mondoo/motor/providers/vsphere/vsimulator"
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
	assets, err := r.Resolve(context.Background(), &asset.Asset{}, &providers.Config{
		Backend:  providers.ProviderType_VSPHERE,
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
		Discover: &providers.Discovery{
			Targets: []string{"all"},
		},
	}, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, 9, len(assets)) // api + esx + vm
}
