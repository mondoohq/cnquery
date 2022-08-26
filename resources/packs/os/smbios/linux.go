package smbios

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/afero"
	os_provider "go.mondoo.com/cnquery/motor/providers/os"
)

type LinuxSmbiosManager struct {
	provider os_provider.OperatingSystemProvider
}

func (s *LinuxSmbiosManager) Name() string {
	return "Linux Smbios Manager"
}

func (s *LinuxSmbiosManager) Info() (*SmBiosInfo, error) {
	smInfo := SmBiosInfo{}

	fs := s.provider.FS()
	afs := &afero.Afero{Fs: fs}
	root := "/sys/class/dmi/id/"

	wErr := afs.Walk(root, func(path string, info os.FileInfo, fErr error) error {
		if info.IsDir() && path != root {
			return filepath.SkipDir
		}

		var dst *string
		switch info.Name() {
		case "bios_date":
			dst = &smInfo.BIOS.ReleaseDate
		case "bios_vendor":
			dst = &smInfo.BIOS.Vendor
		case "bios_version":
			dst = &smInfo.BIOS.Version
		case "board_asset_tag":
			dst = &smInfo.BaseBoardInfo.AssetTag
		case "board_name":
			dst = &smInfo.BaseBoardInfo.Model
		case "board_serial":
			dst = &smInfo.BaseBoardInfo.SerialNumber
		case "board_vendor":
			dst = &smInfo.BaseBoardInfo.Vendor
		case "board_version":
			dst = &smInfo.BaseBoardInfo.Version
		case "chassis_asset_tag":
			dst = &smInfo.ChassisInfo.AssetTag
		case "chassis_serial":
			dst = &smInfo.ChassisInfo.SerialNumber
		case "chassis_type":
			dst = &smInfo.ChassisInfo.Type
		case "chassis_vendor":
			dst = &smInfo.ChassisInfo.Vendor
		case "chassis_version":
			dst = &smInfo.ChassisInfo.Version
		case "product_family":
			dst = &smInfo.SysInfo.Familiy
		case "product_name":
			dst = &smInfo.SysInfo.Model
		case "product_serial":
			dst = &smInfo.SysInfo.SerialNumber
		case "product_sku":
			dst = &smInfo.SysInfo.SKU
		case "product_uuid":
			dst = &smInfo.SysInfo.UUID
		case "product_version":
			dst = &smInfo.SysInfo.Version
		case "sys_vendor":
			dst = &smInfo.SysInfo.Vendor
		}

		if dst != nil {
			f, err := fs.Open(path)
			if err != nil {
				return err
			}
			defer f.Close()
			data, err := ioutil.ReadAll(f)
			if err != nil {
				return err
			}
			*dst = strings.TrimSpace(string(data))
		}

		return nil
	})

	return &smInfo, wErr
}
