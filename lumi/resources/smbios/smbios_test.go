package smbios

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.io/mondoo/motor"
	"go.mondoo.io/mondoo/motor/transports"
	"go.mondoo.io/mondoo/motor/transports/mock"
)

func TestManagerCentos(t *testing.T) {
	mock, err := mock.NewFromToml(&transports.TransportConfig{Backend: transports.TransportBackend_CONNECTION_MOCK, Path: "./testdata/centos.toml"})
	require.NoError(t, err)
	m, err := motor.New(mock)
	require.NoError(t, err)

	mm, err := ResolveManager(m)
	require.NoError(t, err)
	biosInfo, err := mm.Info()
	require.NoError(t, err)
	assert.Equal(t, &SmBIOSInfo{
		BIOS: BiosInfo{
			Vendor:      "innotek GmbH",
			Version:     "VirtualBox",
			ReleaseDate: "12/01/2006"},
		SysInfo: SysInfo{
			Vendor:       "innotek GmbH",
			Model:        "VirtualBox",
			Version:      "1.2",
			SerialNumber: "0",
			UUID:         "64f118d3-0060-4a4c-bf1f-a11d655c4d6f",
			Familiy:      "Virtual Machine",
			SKU:          ""},
		BaseBoardInfo: BaseBoardInfo{
			Vendor:       "Oracle Corporation",
			Model:        "VirtualBox",
			Version:      "1.2",
			SerialNumber: "0",
			AssetTag:     ""},
		ChassisInfo: ChassisInfo{
			Vendor:       "Oracle Corporation",
			Model:        "",
			Version:      "",
			SerialNumber: "",
			AssetTag:     "",
			Type:         "1"},
	}, biosInfo)
}
