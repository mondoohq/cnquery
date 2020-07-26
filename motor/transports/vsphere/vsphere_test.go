package vsphere

// TODO: include simulator to run tests in CI
// import (
// 	"testing"
// 	"go.mondoo.io/mondoo/motor/transports"
// 	"github.com/stretchr/testify/require"
// 	"github.com/stretchr/testify/assert"
// )

// func TestVSphereTransport(t *testing.T) {
// 	trans, err := New(&transports.TransportConfig{
// 		Backend: transports.TransportBackend_CONNECTION_VSPHERE,
// 		Host: "127.0.0.1:8990",
// 		User: "user",
// 		Password: "pass",
// 		// Host: "192.168.56.102",
// 		// User: "root",
// 		// Password: "password1!",
// 	})
// 	require.NoError(t, err)

// 	ver := trans.Client().ServiceContent.About
// 	assert.Equal(t, "6.5", ver.ApiVersion)
// }
