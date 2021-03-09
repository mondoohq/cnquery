package vsphere

//import (
//	"go.mondoo.io/mondoo/motor/transports"
//	"testing"
//
//	"github.com/stretchr/testify/assert"
//	"github.com/stretchr/testify/require"
//)
//
//func TestVsphereResolver(t *testing.T) {
//	r := Resolver{}
//	assets, err := r.Resolve(&transports.TransportConfig{
//		Backend:  transports.TransportBackend_CONNECTION_VSPHERE,
//		User:     "root",
//		Host:     "192.168.87.7",
//		Password: "password1!",
//		Discover: &transports.Discovery{
//			Targets: []string{"all"},
//		},
//	})
//	require.NoError(t, err)
//	assert.Equal(t, 9, len(assets)) // api + esx + vm
//}
