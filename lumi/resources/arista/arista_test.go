package arista

// import (
// 	"fmt"
// 	"testing"

// 	"github.com/aristanetworks/goeapi"
// 	"github.com/stretchr/testify/assert"
// 	"github.com/stretchr/testify/require"
// )

// func TestAristaConnection(t *testing.T) {
// 	// connect to our device
// 	node, err := goeapi.Connect("http", "localhost", "admin", "", 8080)
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	eos := Eos{node: node}

// 	config := eos.RunningConfig()
// 	assert.True(t, len(config) > 0)

// 	systemConfig := eos.SystemConfig()
// 	assert.Equal(t, 2, len(systemConfig))
// 	assert.Equal(t, "sw4", systemConfig["hostname"])

// 	ifaces := eos.IPInterfaces()
// 	assert.Equal(t, 2, len(ifaces))

// 	res, err := eos.Stp()
// 	require.NoError(t, err)
// 	fmt.Printf("%v", res)

// 	res2, err := eos.StpInterfaceDetails("0", "Ethernet1")
// 	require.NoError(t, err)
// 	fmt.Printf("%v", res2)
// }
