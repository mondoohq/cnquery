package smbios

import (
	"encoding/json"
	"io"
	"io/ioutil"

	"go.mondoo.io/mondoo/lumi/resources/powershell"
	"go.mondoo.io/mondoo/motor"
)

const smbiosWindowsScript = `
$bios = Get-WmiObject -class Win32_Bios
$baseboard = Get-WmiObject Win32_BaseBoard
$chassis = Get-WmiObject Win32_SystemEnclosure
$sys = Get-WmiObject Win32_ComputerSystem
$sysProduct = Get-WmiObject Win32_ComputerSystemProduct

$smbios = New-Object -TypeName PSObject
$smbios | Add-Member -MemberType NoteProperty -Value $bios -Name Bios
$smbios | Add-Member -MemberType NoteProperty -Value $baseboard -Name BaseBoard
$smbios | Add-Member -MemberType NoteProperty -Value $chassis -Name Chassis
$smbios | Add-Member -MemberType NoteProperty -Value $sys -Name System
$smbios | Add-Member -MemberType NoteProperty -Value $sysProduct -Name SystemProduct

$smbios | ConvertTo-Json
`

type smbiosWindows struct {
	Bios          smbiosWinBios       `json:"Bios"`
	BaseBoard     smbiosBaseBoard     `json:"BaseBoard"`
	Chassis       smbiosChassis       `json:"Chassis"`
	System        smbiosSystem        `json:"System"`
	SystemProduct smbiosSystemProduct `json:"SystemProduct"`
}

type smbiosWinBios struct {
	Manufacturer      string `json:"Manufacturer"`
	SMBIOSBIOSVersion string `json:"SMBIOSBIOSVersion"`
	ReleaseDate       string `json:"ReleaseDate"`
	SerialNumber      string `json:"SerialNumber"`
}

type smbiosBaseBoard struct {
	Manufacturer string `json:"Manufacturer"`
	Product      string `json:"Product"`
	Version      string `json:"Version"`
	SerialNumber string `json:"SerialNumber"`
}

type smbiosChassis struct {
	Manufacturer string  `json:"Manufacturer"`
	Model        *string `json:"Model"`

	ChassisTypes []uint `json:"ChassisTypes"`

	Version        string `json:"Version"`
	SerialNumber   string `json:"SerialNumber"`
	SMBIOSAssetTag string `json:"SMBIOSAssetTag"`
}

type smbiosSystem struct{}

type smbiosSystemProduct struct {
	Vendor            string `json:"Vendor"`
	Name              string `json:"Name"`
	Version           string `json:"Version"`
	SKUNumber         string `json:"SKUNumber"`
	UUID              string `json:"UUID"`
	IdentifyingNumber string `json:"IdentifyingNumber"`
}

// see https://docs.microsoft.com/en-us/windows-hardware/drivers/bringup/sample-powershell-script-to-query-smbios-locally
// https://docs.microsoft.com/en-us/windows-hardware/drivers/bringup/smbios
type WindowsSmbiosManager struct {
	motor *motor.Motor
}

func (s *WindowsSmbiosManager) Name() string {
	return "Windows Smbios Manager"
}

func (s *WindowsSmbiosManager) Info() (*SmBiosInfo, error) {
	c, err := s.motor.Transport.RunCommand(powershell.Encode(smbiosWindowsScript))
	if err != nil {
		return nil, err
	}

	winBios, err := ParseWindowsSmbiosInfo(c.Stdout)
	if err != nil {
		return nil, err
	}

	smInfo := SmBiosInfo{
		BIOS: BiosInfo{
			Vendor:      winBios.Bios.Manufacturer,
			Version:     winBios.Bios.SMBIOSBIOSVersion,
			ReleaseDate: winBios.Bios.ReleaseDate,
		},
		SysInfo: SysInfo{
			Vendor:  winBios.SystemProduct.Vendor,
			Model:   winBios.SystemProduct.Name,
			Version: winBios.SystemProduct.Version,
			SKU:     winBios.SystemProduct.SKUNumber,
			UUID:    winBios.SystemProduct.UUID,
		},
		BaseBoardInfo: BaseBoardInfo{
			Vendor:       winBios.BaseBoard.Manufacturer,
			Model:        winBios.BaseBoard.Product,
			SerialNumber: winBios.BaseBoard.SerialNumber,
			Version:      winBios.BaseBoard.Version,
		},
		ChassisInfo: ChassisInfo{
			Vendor:       winBios.Chassis.Manufacturer,
			Model:        toString(winBios.Chassis.Model),
			Version:      winBios.Chassis.Version,
			SerialNumber: winBios.Chassis.SerialNumber,
		},
	}

	return &smInfo, nil
}

func toString(i *string) string {
	if i == nil {
		return ""
	}
	return *i
}

func ParseWindowsSmbiosInfo(r io.Reader) (smbiosWindows, error) {
	var smbios smbiosWindows
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return smbios, err
	}

	err = json.Unmarshal(data, &smbios)
	if err != nil {
		return smbios, err
	}

	return smbios, nil
}
