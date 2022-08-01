package vmwareguestapi

// import (
// 	"io/ioutil"
// 	"testing"

// 	"github.com/stretchr/testify/assert"
// 	"github.com/stretchr/testify/require"
// 	"go.mondoo.io/mondoo/motor/transports"
// )

// func TestRunCommand(t *testing.T) {
// 	trans, err := New(&transports.TransportConfig{
// 		Backend:  transports.TransportBackend_CONNECTION_VSPHERE_VM,
// 		Host:     "192.168.178.139",
// 		User:     "root",
// 		Password: "password1!",
// 		Options: map[string]string{
// 			"inventoryPath": "/ha-datacenter/vm/example-centos",
// 			"guestUser":     "root",
// 			"guestPassword": "vagrant",
// 		},
// 	})
// 	require.NoError(t, err)
// 	cmd, err := trans.RunCommand("uname -s")
// 	require.NoError(t, err)
// 	data, err := ioutil.ReadAll(cmd.Stdout)
// 	require.NoError(t, err)
// 	assert.Equal(t, "Linux\n", string(data))

// 	cmd, err = trans.RunCommand("cat /etc/os-release")
// 	require.NoError(t, err)
// 	data, err = ioutil.ReadAll(cmd.Stdout)
// 	require.NoError(t, err)
// 	assert.Equal(t, 393, len(string(data)))

// 	f, err := trans.FS().Open("/etc/os-release")
// 	data, err = ioutil.ReadAll(f)
// 	require.NoError(t, err)
// 	assert.Equal(t, 393, len(string(data)))
// }
