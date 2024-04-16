// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v11/providers/os/connection/shared"
	"go.mondoo.com/cnquery/v11/providers/os/resources/windows"
)

func (s *mqlWindowsBitlocker) volumes() ([]interface{}, error) {
	conn := s.MqlRuntime.Connection.(shared.Connection)

	volumes, err := windows.GetBitLockerVolumes(conn)
	if err != nil {
		return nil, err
	}

	res := []interface{}{}
	for i := range volumes {
		v := volumes[i]

		cs, _ := convert.JsonToDict(v.ConversionStatus)
		em, _ := convert.JsonToDict(v.EncryptionMethod)
		version, _ := convert.JsonToDict(v.Version)
		ps, _ := convert.JsonToDict(v.ProtectionStatus)

		volume, err := CreateResource(s.MqlRuntime, "windows.bitlocker.volume", map[string]*llx.RawData{
			"deviceID":           llx.StringData(v.DeviceID),
			"driveLetter":        llx.StringData(v.DriveLetter),
			"conversionStatus":   llx.DictData(cs),
			"encryptionMethod":   llx.DictData(em),
			"lockStatus":         llx.IntData(v.LockStatus),
			"persistentVolumeID": llx.StringData(v.PersistentVolumeID),
			"protectionStatus":   llx.DictData(ps),
			"version":            llx.DictData(version),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, volume)
	}
	return res, nil
}

func (s *mqlWindowsBitlockerVolume) id() (string, error) {
	return "bitlocker.volume/" + s.DeviceID.Data, nil
}
