package smbios

import (
	"errors"

	"go.mondoo.io/mondoo/motor"
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

func ResolveManager(motor *motor.Motor) (SmBiosManager, error) {
	var biosM SmBiosManager

	platform, err := motor.Platform()
	if err != nil {
		return nil, err
	}

	// check darwin before unix since darwin is also a unix
	if platform.IsFamily("darwin") {
		biosM = &OSXSmbiosManager{motor: motor}
	} else if platform.IsFamily("linux") {
		biosM = &LinuxSmbiosManager{motor: motor}
	} else if platform.IsFamily("windows") {
		biosM = &WindowsSmbiosManager{motor: motor}
	}

	if biosM == nil {
		return nil, errors.New("could not detect suitable smbios manager for platform: " + platform.Name)
	}

	return biosM, nil
}
