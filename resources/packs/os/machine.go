package os

import (
	"fmt"

	"errors"
	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/resources/packs/os/smbios"
)

// we use MQL machine to cache the queries to the os
func (m *mqlMachine) id() (string, error) {
	return "machine", nil
}

func getbiosinfo(runtime *resources.Runtime) (*smbios.SmBiosInfo, error) {
	obj, err := runtime.CreateResource("machine")
	if err != nil {
		return nil, err
	}
	machine := obj.(Machine)

	// check if we already have simething in the cache
	var biosInfo *smbios.SmBiosInfo
	c, ok := machine.MqlResource().Cache.Load("_biosInfo")
	if ok {
		biosInfo = c.Data.(*smbios.SmBiosInfo)
	} else {
		// find suitable package manager
		pf, err := runtime.Motor.Platform()
		if err != nil {
			return nil, errors.Join(err, errors.New("could not detect suitable smbios manager for platform"))
		}

		osProvider, err := osProvider(runtime.Motor)
		if err != nil {
			return nil, err
		}

		pm, err := smbios.ResolveManager(osProvider, pf)
		if pm == nil || err != nil {
			return nil, fmt.Errorf("could not detect suitable smbios manager for platform")
		}

		// retrieve smbios info
		biosInfo, err = pm.Info()
		if err != nil {
			return nil, errors.Join(err, errors.New("could not retrieve smbios info for platform"))
		}

		machine.MqlResource().Cache.Store("_biosInfo", &resources.CacheEntry{Data: biosInfo})
	}

	return biosInfo, nil
}

func (m *mqlMachineBios) id() (string, error) {
	return "machine.bios", nil
}

func (p *mqlMachineBios) init(args *resources.Args) (*resources.Args, MachineBios, error) {
	if len(*args) > 2 {
		return args, nil, nil
	}

	biosInfo, err := getbiosinfo(p.MotorRuntime)
	if err != nil {
		return nil, nil, err
	}

	if biosInfo == nil {
		return nil, nil, errors.New("could not retrieve smbios info")
	}

	(*args)["vendor"] = biosInfo.BIOS.Vendor
	(*args)["version"] = biosInfo.BIOS.Version
	(*args)["releaseDate"] = biosInfo.BIOS.ReleaseDate

	return args, nil, nil
}

func (m *mqlMachineSystem) id() (string, error) {
	return "machine.system", nil
}

func (p *mqlMachineSystem) init(args *resources.Args) (*resources.Args, MachineSystem, error) {
	if len(*args) > 2 {
		return args, nil, nil
	}

	biosInfo, err := getbiosinfo(p.MotorRuntime)
	if err != nil {
		return nil, nil, err
	}

	if biosInfo == nil {
		return nil, nil, errors.New("could not retrieve smbios info")
	}

	(*args)["manufacturer"] = biosInfo.SysInfo.Vendor
	(*args)["product"] = biosInfo.SysInfo.Model
	(*args)["version"] = biosInfo.SysInfo.Version
	(*args)["serial"] = biosInfo.SysInfo.SerialNumber
	(*args)["uuid"] = biosInfo.SysInfo.UUID
	(*args)["sku"] = biosInfo.SysInfo.SKU
	(*args)["family"] = biosInfo.SysInfo.Familiy

	return args, nil, nil
}

func (m *mqlMachineBaseboard) id() (string, error) {
	return "machine.baseboard", nil
}

func (p *mqlMachineBaseboard) init(args *resources.Args) (*resources.Args, MachineBaseboard, error) {
	if len(*args) > 2 {
		return args, nil, nil
	}

	biosInfo, err := getbiosinfo(p.MotorRuntime)
	if err != nil {
		return nil, nil, err
	}

	if biosInfo == nil {
		return nil, nil, errors.New("could not retrieve smbios info")
	}

	(*args)["manufacturer"] = biosInfo.BaseBoardInfo.Vendor
	(*args)["product"] = biosInfo.BaseBoardInfo.Model
	(*args)["version"] = biosInfo.BaseBoardInfo.Version
	(*args)["serial"] = biosInfo.BaseBoardInfo.SerialNumber
	(*args)["assetTag"] = biosInfo.BaseBoardInfo.AssetTag

	return args, nil, nil
}

func (m *mqlMachineChassis) id() (string, error) {
	return "machine.chassis", nil
}

func (p *mqlMachineChassis) init(args *resources.Args) (*resources.Args, MachineChassis, error) {
	if len(*args) > 2 {
		return args, nil, nil
	}

	biosInfo, err := getbiosinfo(p.MotorRuntime)
	if err != nil {
		return nil, nil, err
	}

	if biosInfo == nil {
		return nil, nil, errors.New("could not retrieve smbios info")
	}

	(*args)["manufacturer"] = biosInfo.ChassisInfo.Vendor
	(*args)["version"] = biosInfo.ChassisInfo.Version
	(*args)["serial"] = biosInfo.ChassisInfo.SerialNumber
	(*args)["assetTag"] = biosInfo.ChassisInfo.AssetTag

	return args, nil, nil
}
