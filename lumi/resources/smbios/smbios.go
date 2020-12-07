package smbios

import (
	"errors"
	"io/ioutil"
	"os"

	"github.com/spf13/afero"
	"go.mondoo.io/mondoo/motor"
)

type SmBIOSInfo struct {
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
type SmBIOSManager interface {
	Name() string
	Info() (*SmBIOSInfo, error)
}

func ResolveManager(motor *motor.Motor) (SmBIOSManager, error) {
	var biosM SmBIOSManager

	platform, err := motor.Platform()
	if err != nil {
		return nil, err
	}

	// check darwin before unix since darwin is also a unix
	if platform.IsFamily("darwin") {
		biosM = &OSXKernelManager{motor: motor}
	} else if platform.IsFamily("linux") {
		biosM = &LinuxSmbiosManager{motor: motor}
	}

	if biosM == nil {
		return nil, errors.New("could not detect suitable smbios manager for platform: " + platform.Name)
	}

	return biosM, nil
}

type LinuxSmbiosManager struct {
	motor *motor.Motor
}

func (s *LinuxSmbiosManager) Name() string {
	return "Linux Smbios Manager"
}

func (s *LinuxSmbiosManager) Info() (*SmBIOSInfo, error) {
	smInfo := SmBIOSInfo{}

	fs := s.motor.Transport.FS()
	afs := &afero.Afero{Fs: fs}
	root := "/sys/class/dmi/id"
	afs.Walk(root, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}

		f, err := fs.Open(path)
		if err != nil {
			return err
		}
		data, err := ioutil.ReadAll(f)
		if err != nil {
			return err
		}
		f.Close()
		value := string(data)

		switch info.Name() {
		case "bios_date":
			smInfo.BIOS.ReleaseDate = value
		case "bios_vendor":
			smInfo.BIOS.Vendor = value
		case "bios_version":
			smInfo.BIOS.Version = value
		case "board_asset_tag":
			smInfo.BaseBoardInfo.AssetTag = value
		case "board_name":
			smInfo.BaseBoardInfo.Model = value
		case "board_serial":
			smInfo.BaseBoardInfo.SerialNumber = value
		case "board_vendor":
			smInfo.BaseBoardInfo.Vendor = value
		case "board_version":
			smInfo.BaseBoardInfo.Version = value
		case "chassis_asset_tag":
			smInfo.ChassisInfo.AssetTag = value
		case "chassis_serial":
			smInfo.ChassisInfo.SerialNumber = value
		case "chassis_type":
			smInfo.ChassisInfo.Type = value
		case "chassis_vendor":
			smInfo.ChassisInfo.Vendor = value
		case "chassis_version":
			smInfo.ChassisInfo.Version = value
		case "product_family":
			smInfo.SysInfo.Familiy = value
		case "product_name":
			smInfo.SysInfo.Model = value
		case "product_serial":
			smInfo.SysInfo.SerialNumber = value
		case "product_sku":
			smInfo.SysInfo.SKU = value
		case "product_uuid":
			smInfo.SysInfo.UUID = value
		case "product_version":
			smInfo.SysInfo.Version = value
		case "sys_vendor":
			smInfo.SysInfo.Vendor = value
		}

		return nil
	})

	return &smInfo, nil
}

type OSXKernelManager struct {
	motor *motor.Motor
}

func (s *OSXKernelManager) Name() string {
	return "macOS Smbios Manager"
}

func (s *OSXKernelManager) Info() (*SmBIOSInfo, error) {
	return nil, errors.New("not implemented")
}
