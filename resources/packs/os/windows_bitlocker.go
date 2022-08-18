package os

import (
	"go.mondoo.io/mondoo/resources/packs/core"
	"go.mondoo.io/mondoo/resources/packs/os/windows"
)

func (s *lumiWindowsBitlocker) id() (string, error) {
	return "windows.bitlocker", nil
}

func (s *lumiWindowsBitlocker) GetVolumes() ([]interface{}, error) {
	osProvider, err := osProvider(s.MotorRuntime.Motor)
	if err != nil {
		return nil, err
	}

	volumes, err := windows.GetBitLockerVolumes(osProvider)
	if err != nil {
		return nil, err
	}

	res := []interface{}{}
	for i := range volumes {
		v := volumes[i]

		cs, _ := core.JsonToDict(v.ConversionStatus)
		em, _ := core.JsonToDict(v.EncryptionMethod)
		version, _ := core.JsonToDict(v.Version)
		ps, _ := core.JsonToDict(v.ProtectionStatus)

		volume, err := s.MotorRuntime.CreateResource("windows.bitlocker.volume",
			"deviceID", v.DeviceID,
			"driveLetter", v.DriveLetter,
			"conversionStatus", cs,
			"encryptionMethod", em,
			"lockStatus", v.LockStatus,
			"persistentVolumeID", v.PersistentVolumeID,
			"protectionStatus", ps,
			"version", version,
		)
		if err != nil {
			return nil, err
		}
		res = append(res, volume)
	}
	return res, nil
}

func (s *lumiWindowsBitlockerVolume) id() (string, error) {
	deviceId, err := s.DeviceID()
	if err != nil {
		return "", err
	}

	return "bitlocker.volume/" + deviceId, nil
}
