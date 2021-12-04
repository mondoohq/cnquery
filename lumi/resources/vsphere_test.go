package resources_test

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.io/mondoo/llx"
	"go.mondoo.io/mondoo/motor"
	"go.mondoo.io/mondoo/motor/transports"
	"go.mondoo.io/mondoo/motor/transports/vsphere"
	"go.mondoo.io/mondoo/motor/transports/vsphere/vsimulator"
	"go.mondoo.io/mondoo/motor/vault"
)

func vsphereTestQuery(t *testing.T, query string) []*llx.RawResult {
	vs, err := vsimulator.New()
	require.NoError(t, err)
	defer vs.Close()

	port, err := strconv.Atoi(vs.Server.URL.Port())
	require.NoError(t, err)

	trans, err := vsphere.New(&transports.TransportConfig{
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
	})
	require.NoError(t, err)

	m, err := motor.New(trans)
	require.NoError(t, err)

	executor := initExecutor(m)
	return testQueryWithExecutor(t, executor, query, nil)
}

func TestResource_Vsphere(t *testing.T) {
	t.Run("vsphere datacenter", func(t *testing.T) {
		res := vsphereTestQuery(t, "vsphere.datacenters[0].hosts[0].name")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, string("DC0_H0"), res[0].Data.Value)
	})
}
