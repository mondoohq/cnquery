package smbios

import (
	"errors"

	"go.mondoo.io/mondoo/motor/providers/os"

	"go.mondoo.io/mondoo/motor/platform"
)

type SmBiosInfo struct {
	BIOS          BiosInfo
	SysInfo       SysInfo
	BaseBoardInfo BaseBoardInfo
	ChassisInfo   ChassisInfo
}

type BiosInfo struct {
	Vendor      string
	Version     string
	ReleaseDate string
}

type SysInfo struct {
	Vendor       string
	Model        string
	Version      string
	SerialNumber string
	UUID         string
	Familiy      string
	SKU          string
}

type BaseBoardInfo struct {
	Vendor       string
	Model        string
	Version      string
	SerialNumber string
	AssetTag     string
}

type ChassisInfo struct {
	Vendor       string
	Model        string
	Version      string
	SerialNumber string
	AssetTag     string
	Type         string
}

// https://en.wikipedia.org/wiki/System_Management_BIOS
// https://www.dmtf.org/sites/default/files/standards/documents/DSP0134_3.4.0.pdf
// There are also tools (https://github.com/digitalocean/go-smbios) out there to parse
// the memory dump directly, but this would require to transfer large amout of data \
// for remove access, therefore we restrict the data to what is exposed in /sys/class/dmi/id/
type SmBiosManager interface {
	Name() string
	Info() (*SmBiosInfo, error)
}

func ResolveManager(provider os.OperatingSystemProvider, pf *platform.Platform) (SmBiosManager, error) {
	var biosM SmBiosManager

	// check darwin before unix since darwin is also a unix
	if pf.IsFamily("darwin") {
		biosM = &OSXSmbiosManager{provider: provider, platform: pf}
	} else if pf.IsFamily("linux") {
		biosM = &LinuxSmbiosManager{provider: provider}
	} else if pf.IsFamily("windows") {
		biosM = &WindowsSmbiosManager{provider: provider}
	}

	if biosM == nil {
		return nil, errors.New("could not detect suitable smbios manager for platform: " + pf.Name)
	}

	return biosM, nil
}
