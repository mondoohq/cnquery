// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"encoding/json"
	"errors"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
)

func initMacosHardware(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if args == nil {
		args = map[string]*llx.RawData{}
	}

	// MQL equivalent in CLI
	// parse.json(content: command('system_profiler SPHardwareDataType -json').stdout).params['SPHardwareDataType'].first['chip_type']
	res, err := NewResource(runtime, "command", map[string]*llx.RawData{
		"command": llx.StringData("system_profiler SPHardwareDataType -json"),
	})
	if err != nil {
		return nil, nil, err
	}

	cmd, ok := res.(*mqlCommand)
	if !ok {
		return nil, nil, errors.New("could not run command")
	}

	jsonData := cmd.GetStdout().Data

	type machineJsonData struct {
		SPHardwareDataType []struct {
			ActivationLockStatus string `json:"activation_lock_status"`
			BootRomVersion       string `json:"boot_rom_version"`
			ChipType             string `json:"chip_type"`
			MachineModel         string `json:"machine_model"`
			MachineName          string `json:"machine_name"`
			ModelNumber          string `json:"model_number"`
			NumberProcessors     string `json:"number_processors"`
			OsLoaderVersion      string `json:"os_loader_version"`
			PhysicalMemory       string `json:"physical_memory"`
			PlatformUUID         string `json:"platform_uuid"`
			ProvisioningUUID     string `json:"provisioning_uuid"`
			SerialNumber         string `json:"serial_number"`
		} `json:"SPHardwareDataType"`
	}

	var resp machineJsonData
	err = json.Unmarshal([]byte(jsonData), &resp)
	if err != nil {
		return nil, nil, errors.New("could not gather hardware information")
	}
	if len(resp.SPHardwareDataType) == 0 {
		return nil, nil, errors.New("could not gather hardware information")
	}

	hardware := resp.SPHardwareDataType[0]
	args["activationLockStatus"] = llx.StringData(hardware.ActivationLockStatus)
	args["bootRomVersion"] = llx.StringData(hardware.BootRomVersion)
	args["chipType"] = llx.StringData(hardware.ChipType)
	args["machineModel"] = llx.StringData(hardware.MachineModel)
	args["machineName"] = llx.StringData(hardware.MachineName)
	args["modelNumber"] = llx.StringData(hardware.ModelNumber)
	args["numberProcessors"] = llx.StringData(hardware.NumberProcessors)
	args["osLoaderVersion"] = llx.StringData(hardware.OsLoaderVersion)
	args["physicalMemory"] = llx.StringData(hardware.PhysicalMemory)
	args["platformUUID"] = llx.StringData(hardware.PlatformUUID)
	args["provisioningUDID"] = llx.StringData(hardware.ProvisioningUUID)
	args["serialNumber"] = llx.StringData(hardware.SerialNumber)

	return args, nil, nil
}
