package resources

import (
	"fmt"

	"github.com/cockroachdb/errors"
	"go.mondoo.io/mondoo/lumi"
	"go.mondoo.io/mondoo/lumi/resources/smbios"
)

// we use lumi machine to cache the queries to the os
func (m *lumiMachine) id() (string, error) {
	return "machine", nil
}

func getbiosinfo(runtime *lumi.Runtime) (*smbios.SmBiosInfo, error) {
	obj, err := runtime.CreateResource("machine")
	if err != nil {
		return nil, err
	}
	machine := obj.(Machine)

	// check if we already have simething in the cache
	var biosInfo *smbios.SmBiosInfo
	c, ok := machine.LumiResource().Cache.Load("_biosInfo")
	if ok {
		biosInfo = c.Data.(*smbios.SmBiosInfo)
	} else {
		// find suitable package manager
		t := runtime.Motor.Transport
		p, err := runtime.Motor.Platform()
		if err != nil {
			return nil, errors.Wrap(err, "could not detect suiteable smbios manager for platform")
		}
		pm, err := smbios.ResolveManager(t, p)
		if pm == nil || err != nil {
			return nil, fmt.Errorf("could not detect suiteable smbios manager for platform")
		}

		// retrieve smbios info
		biosInfo, err = pm.Info()
		if err != nil {
			return nil, errors.Wrap(err, "could not retrieve smbios info for platform")
		}

		machine.LumiResource().Cache.Store("_biosInfo", &lumi.CacheEntry{Data: biosInfo})
	}

	return biosInfo, nil
}

func (m *lumiMachineBios) id() (string, error) {
	return "machine.bios", nil
}

func (p *lumiMachineBios) init(args *lumi.Args) (*lumi.Args, MachineBios, error) {
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

func (m *lumiMachineSystem) id() (string, error) {
	return "machine.system", nil
}

func (p *lumiMachineSystem) init(args *lumi.Args) (*lumi.Args, MachineSystem, error) {
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

func (m *lumiMachineBaseboard) id() (string, error) {
	return "machine.baseboard", nil
}

func (p *lumiMachineBaseboard) init(args *lumi.Args) (*lumi.Args, MachineBaseboard, error) {
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

func (m *lumiMachineChassis) id() (string, error) {
	return "machine.chassis", nil
}

func (p *lumiMachineChassis) init(args *lumi.Args) (*lumi.Args, MachineChassis, error) {
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
