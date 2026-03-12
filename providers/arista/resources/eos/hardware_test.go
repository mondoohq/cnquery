// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package eos

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnvironmentPowerParsing(t *testing.T) {
	data, err := os.ReadFile("./testdata/environment-power.json")
	require.NoError(t, err)

	var envPower showEnvironmentPower
	err = json.Unmarshal(data, &envPower)
	require.NoError(t, err)

	assert.Len(t, envPower.PowerSupplies, 2)

	ps1 := envPower.PowerSupplies["1"]
	assert.Equal(t, "ok", ps1.State)
	assert.Equal(t, "PWR-500AC-R", ps1.ModelName)
	assert.Equal(t, 500, ps1.Capacity)
	assert.InDelta(t, 82.5, ps1.OutputPower, 0.1)
	assert.InDelta(t, 0.85, ps1.InputCurrent, 0.01)
	assert.InDelta(t, 6.8, ps1.OutputCurrent, 0.1)
	assert.InDelta(t, 1209600.5, ps1.Uptime, 0.1)
	assert.True(t, ps1.Managed)

	// Verify nested temp sensors
	assert.Len(t, ps1.TempSensors, 1)
	sensor, ok := ps1.TempSensors["TempSensorP1/1"]
	require.True(t, ok)
	assert.Equal(t, "ok", sensor.Status)
	assert.Equal(t, 32, sensor.Temperature)

	// Verify nested fans
	assert.Len(t, ps1.Fans, 1)
	fan, ok := ps1.Fans["FanP1/1"]
	require.True(t, ok)
	assert.Equal(t, "ok", fan.Status)
	assert.Equal(t, 33, fan.Speed)

	// Verify failed PSU
	ps2 := envPower.PowerSupplies["2"]
	assert.Equal(t, "powerLoss", ps2.State)
	assert.InDelta(t, 0.0, ps2.OutputPower, 0.01)
	assert.Empty(t, ps2.TempSensors)
	assert.Empty(t, ps2.Fans)
}

func TestEnvironmentCoolingParsing(t *testing.T) {
	data, err := os.ReadFile("./testdata/environment-cooling.json")
	require.NoError(t, err)

	var cooling showEnvironmentCooling
	err = json.Unmarshal(data, &cooling)
	require.NoError(t, err)

	assert.Equal(t, "coolingOk", cooling.SystemStatus)
	assert.Equal(t, "frontToBack", cooling.AirflowDirection)
	assert.Equal(t, "automatic", cooling.CoolingMode)

	// Verify fan trays
	require.Len(t, cooling.FanTraySlots, 2)

	tray1 := cooling.FanTraySlots[0]
	assert.Equal(t, "FanTray1", tray1.Label)
	assert.Equal(t, "ok", tray1.Status)
	require.Len(t, tray1.Fans, 2)
	assert.Equal(t, "1/1", tray1.Fans[0].Label)
	assert.Equal(t, "ok", tray1.Fans[0].Status)
	assert.Equal(t, 50, tray1.Fans[0].Speed)
	assert.Equal(t, 50, tray1.Fans[0].ConfiguredSpeed)

	tray2 := cooling.FanTraySlots[1]
	assert.Equal(t, "FanTray2", tray2.Label)
	require.Len(t, tray2.Fans, 1)
	assert.Equal(t, 45, tray2.Fans[0].Speed)
}

func TestEnvironmentCoolingEmpty(t *testing.T) {
	jsonData := `{"systemStatus": "unknownCoolingAlarmLevel", "fanTraySlots": [], "coolingMode": "automatic"}`
	var cooling showEnvironmentCooling
	err := json.Unmarshal([]byte(jsonData), &cooling)
	require.NoError(t, err)
	assert.Equal(t, "unknownCoolingAlarmLevel", cooling.SystemStatus)
	assert.Empty(t, cooling.FanTraySlots)
}

func TestInventoryParsing(t *testing.T) {
	data, err := os.ReadFile("./testdata/inventory.json")
	require.NoError(t, err)

	var inv showInventory
	err = json.Unmarshal(data, &inv)
	require.NoError(t, err)

	// Verify system information
	assert.Equal(t, "DCS-7050TX-48", inv.SystemInformation.Name)
	assert.Equal(t, "48-port 10GbE SFP+ switch", inv.SystemInformation.Description)
	assert.Equal(t, "JPE12345678", inv.SystemInformation.SerialNum)
	assert.Equal(t, "02.01", inv.SystemInformation.HardwareRev)
	assert.Equal(t, "2019-05-15", inv.SystemInformation.MfgDate)

	// Verify power supply slots
	require.Len(t, inv.PowerSupplySlots, 1)
	psu := inv.PowerSupplySlots["1"]
	assert.Equal(t, "PowerSupply1", psu.Name)
	assert.Equal(t, "PSU987654", psu.SerialNum)

	// Verify fan tray slots
	require.Len(t, inv.FanTraySlots, 1)
	fan := inv.FanTraySlots["1"]
	assert.Equal(t, "FanTray1", fan.Name)
	assert.Equal(t, "FAN111222", fan.SerialNum)

	// Verify transceiver slots
	require.Len(t, inv.XcvrSlots, 2)
	xcvr1 := inv.XcvrSlots["1"]
	assert.Equal(t, "Xcvr1", xcvr1.Name)
	assert.Equal(t, "SFP+ 10GBASE-SR", xcvr1.Description)
	assert.Equal(t, "XCV001002", xcvr1.SerialNum)

	// Verify card slots (empty)
	assert.Empty(t, inv.CardSlots)
}

func TestInventoryMinimal(t *testing.T) {
	// CloudEOS-like response with minimal data
	jsonData := `{
		"systemInformation": {
			"name": "CloudEOS",
			"description": "EOS in a virtual machine",
			"hardwareRev": "",
			"serialNum": "ABCDEF123456",
			"mfgDate": "",
			"hwEpoch": ""
		},
		"powerSupplySlots": {},
		"fanTraySlots": {},
		"xcvrSlots": {},
		"cardSlots": {}
	}`
	var inv showInventory
	err := json.Unmarshal([]byte(jsonData), &inv)
	require.NoError(t, err)
	assert.Equal(t, "CloudEOS", inv.SystemInformation.Name)
	assert.Equal(t, "ABCDEF123456", inv.SystemInformation.SerialNum)
	assert.Empty(t, inv.PowerSupplySlots)
	assert.Empty(t, inv.FanTraySlots)
	assert.Empty(t, inv.XcvrSlots)
	assert.Empty(t, inv.CardSlots)
}

func TestEnvironmentCoolingGetCmd(t *testing.T) {
	s := &showEnvironmentCooling{}
	assert.Equal(t, "show system environment cooling", s.GetCmd())
}

func TestInventoryGetCmd(t *testing.T) {
	s := &showInventory{}
	assert.Equal(t, "show inventory", s.GetCmd())
}
