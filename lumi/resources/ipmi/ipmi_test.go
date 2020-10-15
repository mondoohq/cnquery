package ipmi

// import (
// 	"testing"

// 	"github.com/stretchr/testify/assert"
// 	"github.com/stretchr/testify/require"
// )

// func TestIpmiDockerSimulator(t *testing.T) {
// 	c := &Connection{Hostname: "127.0.0.1", Port: 623, Username: "ADMIN", Password: "ADMIN", Interface: "lanplus"}
// 	client, err := NewIpmiClient(c)
// 	require.NoError(t, err)
// 	err = client.Open()

// 	id, err := client.DeviceID()

// 	expected := &DeviceID{
// 		DeviceID:           int64(0),
// 		DeviceRevision:     int64(3),
// 		ProvidesDeviceSDRs: false,
// 		DeviceAvailable:    true,
// 		FirmwareRevision:   "9.08",
// 		IpmiVersion:        int64(2),
// 		ManufacturerID:     int64(4753),
// 		ManufacturerName:   "Unknown",
// 		ProductID:          int64(3842),
// 		ProductName:        "Unknown",
// 		AdditionalDeviceSupport: AdditionalDeviceSupport{
// 			SensorDevice:        true,
// 			SDRRepositoryDevice: true,
// 			SELDevice:           true,
// 			FRUInventoryDevice:  true,
// 			IPMBEventReceiver:   true,
// 			ChassisDevice:       true,
// 		},
// 	}
// 	assert.Equal(t, expected, id)
// }
