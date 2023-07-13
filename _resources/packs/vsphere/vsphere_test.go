package vsphere_test

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/motor"
	"go.mondoo.com/cnquery/motor/providers"
	provider "go.mondoo.com/cnquery/motor/providers/vsphere"
	"go.mondoo.com/cnquery/motor/providers/vsphere/vsimulator"
	"go.mondoo.com/cnquery/motor/vault"
	"go.mondoo.com/cnquery/resources/packs/testutils"
	pack "go.mondoo.com/cnquery/resources/packs/vsphere"
)

func vsphereTestQuery(t *testing.T, query string) []*llx.RawResult {
	vs, err := vsimulator.New()
	require.NoError(t, err)
	defer vs.Close()

	port, err := strconv.Atoi(vs.Server.URL.Port())
	require.NoError(t, err)

	p, err := provider.New(&providers.Config{
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
	})
	require.NoError(t, err)

	m, err := motor.New(p)
	require.NoError(t, err)

	x := testutils.InitTester(m, pack.Registry)
	return x.TestQuery(t, query)
}

func TestResource_Vsphere(t *testing.T) {
	t.Run("vsphere datacenter", func(t *testing.T) {
		res := vsphereTestQuery(t, "vsphere.datacenters[0].hosts[0].name")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, string("DC0_H0"), res[0].Data.Value)
	})
}
