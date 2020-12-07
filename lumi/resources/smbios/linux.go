package smbios

import (
	"io/ioutil"
	"os"

	"github.com/spf13/afero"
	"go.mondoo.io/mondoo/motor"
)

type LinuxSmbiosManager struct {
	motor *motor.Motor
}

func (s *LinuxSmbiosManager) Name() string {
	return "Linux Smbios Manager"
}

func (s *LinuxSmbiosManager) Info() (*SmBiosInfo, error) {
	smInfo := SmBiosInfo{}

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
