package resolver

// import (
// 	"testing"

// 	"github.com/stretchr/testify/assert"
// 	"github.com/stretchr/testify/require"
// 	"go.mondoo.io/mondoo/apps/mondoo/cmd/options"
// )

// func TestVsphereResolver(t *testing.T) {
// 	r := vsphereResolver{}
// 	assets, err := r.Resolve(&options.VulnOptsAsset{
// 		// Connection: "vsphere://user@127.0.0.1:8990",
// 		// Password:   "pass",
// 		Connection: "vsphere://root@192.168.56.102",
// 		Password:   "password1!",
// 	}, &options.VulnOpts{})
// 	require.NoError(t, err)
// 	assert.Equal(t, 9, len(assets)) // api + esx + vm
// }
