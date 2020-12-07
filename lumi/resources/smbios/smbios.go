package smbios

import (
	"bytes"
	"errors"
	"io"
	"io/ioutil"
	"os"
	"strings"

	"github.com/spf13/afero"
	"go.mondoo.io/mondoo/motor"
	plist "howett.net/plist"
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
		biosM = &OSXSmbiosManager{motor: motor}
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

// We use the ioreg implementation to get native access. Its preferred
// over system_profiler since its much faster
//
// hardware info
// ioreg -rw0 -d2 -c IOPlatformExpertDevice -a
//
// acpi info:
// ioreg -rw0 -d1 -c AppleACPIPlatformExpert
//
// get the rom version:
// ioreg -r -p IODeviceTree -n rom@0
//
// helpful mac commands:
// https://github.com/erikberglund/Scripts/blob/master/snippets/macos_hardware.md
//
// results can be compared with dmidecode
// http://cavaliercoder.com/blog/dmidecode-for-apple-osx.html
type OSXSmbiosManager struct {
	motor *motor.Motor
}

func (s *OSXSmbiosManager) Name() string {
	return "macOS Smbios Manager"
}

func (s *OSXSmbiosManager) Info() (*SmBIOSInfo, error) {
	smInfo := SmBIOSInfo{}

	cmd, err := s.motor.Transport.RunCommand("ioreg -rw0 -d2 -c IOPlatformExpertDevice -a")
	if err != nil {
		return nil, err
	}

	hw, err := ParseMacosIOPlatformExpertDevice(cmd.Stdout)
	if err != nil {
		return nil, err
	}

	smInfo.SysInfo.Vendor = plistData(hw.Manufacturer)
	smInfo.SysInfo.Model = plistData(hw.Model)
	smInfo.SysInfo.Version = plistData(hw.Version)
	smInfo.SysInfo.SerialNumber = hw.IOPlatformSerialNumber
	smInfo.SysInfo.UUID = hw.IOPlatformUUID

	smInfo.BaseBoardInfo.Vendor = plistData(hw.Manufacturer)
	smInfo.BaseBoardInfo.Model = plistData(hw.BoardID)
	smInfo.BaseBoardInfo.Version = ""
	smInfo.BaseBoardInfo.SerialNumber = hw.IOPlatformSerialNumber

	smInfo.ChassisInfo.Vendor = plistData(hw.Manufacturer)
	smInfo.ChassisInfo.Version = plistData(hw.BoardID)
	smInfo.ChassisInfo.SerialNumber = hw.IOPlatformSerialNumber
	smInfo.ChassisInfo.Type = "Laptop"

	cmd, err = s.motor.Transport.RunCommand("ioreg -r -p IODeviceTree -n rom@0 -a")
	if err != nil {
		return nil, err
	}

	bios, err := ParseMacosBios(cmd.Stdout)
	if err != nil {
		return nil, err
	}
	smInfo.BIOS.ReleaseDate = plistData(bios.ReleaseDate)
	smInfo.BIOS.Vendor = plistData(bios.Vendor)
	smInfo.BIOS.Version = plistData(bios.Version)

	return &smInfo, nil
}

func plistData(data []byte) string {
	return string(bytes.Trim(data, "\x00"))
}

type IOPlatformExpertDevice struct {
	IORegistryEntryName    string `plist:"IORegistryEntryName"`
	BoardID                []byte `plist:"board-id"`
	Manufacturer           []byte `plist:"manufacturer"`
	Model                  []byte `plist:"model"`
	SerialNumber           []byte `plist:"serial-number"`
	IOPlatformUUID         string `plist:"IOPlatformUUID"`
	IOPlatformSerialNumber string `plist:"IOPlatformSerialNumber"`
	ProductName            []byte `plist:"product-name"`
	Version                []byte `plist:"version"`
}

func ParseMacosIOPlatformExpertDevice(input io.Reader) (*IOPlatformExpertDevice, error) {
	var r io.ReadSeeker
	r, ok := input.(io.ReadSeeker)

	// if the read seaker is not implemented lets cache stdout in-memory
	if !ok {
		packageList, err := ioutil.ReadAll(input)
		if err != nil {
			return nil, err
		}
		r = strings.NewReader(string(packageList))
	}

	var data []IOPlatformExpertDevice
	decoder := plist.NewDecoder(r)
	err := decoder.Decode(&data)
	if err != nil {
		return nil, err
	}

	return &data[0], nil
}

type IODeviceTree struct {
	Vendor      []byte `plist:"vendor"`
	ReleaseDate []byte `plist:"release-date"`
	Version     []byte `plist:"version"`
}

func ParseMacosBios(input io.Reader) (*IODeviceTree, error) {
	var r io.ReadSeeker
	r, ok := input.(io.ReadSeeker)

	// if the read seaker is not implemented lets cache stdout in-memory
	if !ok {
		packageList, err := ioutil.ReadAll(input)
		if err != nil {
			return nil, err
		}
		r = strings.NewReader(string(packageList))
	}

	var data []IODeviceTree
	decoder := plist.NewDecoder(r)
	err := decoder.Decode(&data)
	if err != nil {
		return nil, err
	}

	return &data[0], nil
}
