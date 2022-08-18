package windows

import (
	"encoding/json"
	"io"
	"io/ioutil"

	"go.mondoo.io/mondoo/motor/providers/os"
	"go.mondoo.io/mondoo/resources/packs/os/powershell"
)

// https://docs.microsoft.com/en-us/windows/win32/secprov/getconversionstatus-win32-encryptablevolume
var conversionStatusValues = map[int64]string{
	0: "FullyDecrypted",
	1: "FullyEncrypted",
	2: "EncryptionInProgress",
	3: "DecryptionInProgress",
	4: "EncryptionPaused",
	5: "DecryptionPaused",
}

// https://docs.microsoft.com/en-us/windows/win32/secprov/getconversionstatus-win32-encryptablevolume
var wipingStatusValues = map[int64]string{
	0: "FreeSpaceNotWiped",
	1: "FreeSpaceWiped",
	2: "FreeSpaceWipingInProgress",
	3: "FreeSpaceWipingPaused",
}

// https://docs.microsoft.com/en-us/windows/win32/secprov/getencryptionmethod-win32-encryptablevolume
var encryptionMethodValues = map[int64]string{
	0: "NONE",
	1: "AES_128_WITH_DIFFUSER",
	2: "AES_256_WITH_DIFFUSER",
	3: "AES_128",
	4: "AES_256",
	5: "HARDWARE_ENCRYPTION",
	6: "XTS_AES_128",
	7: "XTS_AES_256",
}

var fveVersionValues = map[int64]string{
	0: "Unknown",
	1: "Vista",
	2: "Win7",
}

// https://docs.microsoft.com/en-us/windows/win32/secprov/getprotectionstatus-win32-encryptablevolume
var protectionStatusValues = map[int64]string{
	0: "Unprotected",
	1: "Protected",
	2: "Unknown",
}

const bitlockerStatusScript = `
$encryptedVolumes = Get-WmiObject -namespace "Root\cimv2\security\MicrosoftVolumeEncryption" -ClassName "Win32_Encryptablevolume" 

$bitlockerStatus = @()

foreach ($volume in $encryptedVolumes) {
	
	$wmiVersion = $volume.GetVersion()
	$version = New-Object psobject -Property @{
	  "Version" =  $wmiVersion.Version;
	}
	
	$wmiConversionStatus = $volume.GetConversionStatus()
	$conversionStatus = New-Object psobject -Property @{
	  "ConversionStatus" =  $wmiConversionStatus.ConversionStatus;
	  "EncryptionFlags" =  $wmiConversionStatus.EncryptionFlags;
	  "EncryptionPercentage" =  $wmiConversionStatus.EncryptionPercentage;
	  "WipingPercentage"  = $wmiConversionStatus.WipingPercentage;
	  "WipingStatus"  = $wmiConversionStatus.WipingStatus;
	}
	
	$wmilockStatus = $volume.GetLockStatus()
	$lockStatus = New-Object psobject -Property @{
	  "LockStatus" =  $wmilockStatus.LockStatus;
	}
	
	$volumeStatus = New-Object PSObject
	Add-Member -InputObject $volumeStatus -MemberType NoteProperty -Name volume -Value $volume
	Add-Member -InputObject $volumeStatus -MemberType NoteProperty -Name version -Value $version
	Add-Member -InputObject $volumeStatus -MemberType NoteProperty -Name conversionStatus -Value $conversionStatus
	Add-Member -InputObject $volumeStatus -MemberType NoteProperty -Name lockStatus -Value $lockStatus
	$bitlockerStatus = $bitlockerStatus + $volumeStatus
}
ConvertTo-Json -Depth 3 -Compress $bitlockerStatus
`

// powershellBitlockerVolumeStatus is the struct to parse the powershell result
type powershellBitlockerVolumeStatus struct {
	Volume struct {
		ConversionStatus                 int64
		DeviceID                         string
		DriveLetter                      string
		EncryptionMethod                 int64
		IsVolumeInitializedForProtection bool
		PersistentVolumeID               string
		ProtectionStatus                 int64
		VolumeType                       int64
	}
	Version struct {
		Version int64
	}
	ConversionStatus struct {
		ConversionStatus     int64
		WipingStatus         int64
		WipingPercentage     int64
		EncryptionFlags      int64
		EncryptionPercentage int64
	}
	LockStatus struct {
		LockStatus int64
	}
}

// bitlockerVolumeStatus returns the status for one individual volume
type bitlockerVolumeStatus struct {
	DeviceID           string
	DriveLetter        string
	ConversionStatus   conversionStatus
	EncryptionMethod   statusCode
	LockStatus         int64
	PersistentVolumeID string
	ProtectionStatus   statusCode
	Version            statusCode
}

type conversionStatus struct {
	ConversionStatus     statusCode
	WipingStatus         statusCode
	WipingPercentage     int64
	EncryptionPercentage int64
}

type statusCode struct {
	Code int64  `json:"code"`
	Text string `json:"text"`
}

func GetBitLockerVolumes(p os.OperatingSystemProvider) ([]bitlockerVolumeStatus, error) {
	c, err := p.RunCommand(powershell.Encode(bitlockerStatusScript))
	if err != nil {
		return nil, err
	}

	return ParseWindowsBitlockerStatus(c.Stdout)
}

func ParseWindowsBitlockerStatus(r io.Reader) ([]bitlockerVolumeStatus, error) {
	var volumeStatus []powershellBitlockerVolumeStatus
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(data, &volumeStatus)
	if err != nil {
		return nil, err
	}

	res := []bitlockerVolumeStatus{}
	for i := range volumeStatus {
		v := volumeStatus[i]

		bvs := bitlockerVolumeStatus{
			DeviceID:    v.Volume.DeviceID,
			DriveLetter: v.Volume.DriveLetter,
			ConversionStatus: conversionStatus{
				ConversionStatus: statusCode{
					Code: v.ConversionStatus.ConversionStatus,
					Text: conversionStatusValues[v.ConversionStatus.ConversionStatus],
				},
				EncryptionPercentage: v.ConversionStatus.EncryptionPercentage,
				WipingStatus: statusCode{
					Code: v.ConversionStatus.WipingStatus,
					Text: wipingStatusValues[v.ConversionStatus.WipingStatus],
				},
				WipingPercentage: v.ConversionStatus.WipingPercentage,
			},
			EncryptionMethod: statusCode{
				Code: v.Volume.EncryptionMethod,
				Text: encryptionMethodValues[v.Volume.EncryptionMethod],
			},
			LockStatus:         v.LockStatus.LockStatus,
			PersistentVolumeID: v.Volume.PersistentVolumeID,
			ProtectionStatus: statusCode{
				Code: v.Volume.ProtectionStatus,
				Text: protectionStatusValues[v.Volume.ProtectionStatus],
			},
			Version: statusCode{
				Code: v.Version.Version,
				Text: fveVersionValues[v.Version.Version],
			},
		}
		res = append(res, bvs)
	}
	return res, nil
}
