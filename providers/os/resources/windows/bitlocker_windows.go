// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

//go:build windows
// +build windows

package windows

import (
	"fmt"
	"runtime"

	"github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
	"go.mondoo.com/cnquery/v12/providers/os/connection/shared"
)

const bitlockerNamespace = `\\.\ROOT\CIMV2\Security\MicrosoftVolumeEncryption`
const bitlockerQuery = "SELECT * FROM Win32_EncryptableVolume"

// GetBitLockerVolumes retrieves BitLocker volume status.
// On local Windows, it uses native WMI API for better performance (~1-10ms).
// On remote connections, it falls back to PowerShell (~200-500ms).
func GetBitLockerVolumes(conn shared.Connection) ([]bitlockerVolumeStatus, error) {
	// Use native WMI API when running locally on Windows
	if conn.Type() == shared.Type_Local && runtime.GOOS == "windows" {
		return getNativeBitLockerVolumes()
	}
	return getPowershellBitLockerVolumes(conn)
}

// getNativeBitLockerVolumes uses native WMI API to query BitLocker status
func getNativeBitLockerVolumes() ([]bitlockerVolumeStatus, error) {
	// Initialize COM
	if err := ole.CoInitializeEx(0, ole.COINIT_MULTITHREADED); err != nil {
		oleErr, ok := err.(*ole.OleError)
		// S_FALSE (0x00000001) means COM was already initialized, which is fine
		if !ok || oleErr.Code() != 0x00000001 {
			return nil, fmt.Errorf("failed to initialize COM: %w", err)
		}
	}
	defer ole.CoUninitialize()

	// Create WMI locator
	unknown, err := oleutil.CreateObject("WbemScripting.SWbemLocator")
	if err != nil {
		return nil, fmt.Errorf("failed to create WMI locator: %w", err)
	}
	defer unknown.Release()

	wmi, err := unknown.QueryInterface(ole.IID_IDispatch)
	if err != nil {
		return nil, fmt.Errorf("failed to query WMI interface: %w", err)
	}
	defer wmi.Release()

	// Connect to BitLocker WMI namespace
	serviceRaw, err := oleutil.CallMethod(wmi, "ConnectServer", nil, bitlockerNamespace)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to WMI namespace: %w", err)
	}
	service := serviceRaw.ToIDispatch()
	defer service.Release()

	// Execute query for encryptable volumes
	resultRaw, err := oleutil.CallMethod(service, "ExecQuery", bitlockerQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to execute WMI query: %w", err)
	}
	result := resultRaw.ToIDispatch()
	defer result.Release()

	// Get the count of volumes
	countVar, err := oleutil.GetProperty(result, "Count")
	if err != nil {
		return nil, fmt.Errorf("failed to get volume count: %w", err)
	}
	count := int(countVar.Val)

	volumes := make([]bitlockerVolumeStatus, 0, count)

	// Iterate through each volume
	for i := 0; i < count; i++ {
		itemRaw, err := oleutil.CallMethod(result, "ItemIndex", i)
		if err != nil {
			return nil, fmt.Errorf("failed to get volume at index %d: %w", i, err)
		}
		item := itemRaw.ToIDispatch()

		vol, err := extractVolumeStatus(item)
		item.Release()
		if err != nil {
			return nil, fmt.Errorf("failed to extract volume status: %w", err)
		}

		volumes = append(volumes, vol)
	}

	return volumes, nil
}

// extractVolumeStatus extracts BitLocker status from a Win32_EncryptableVolume WMI object
func extractVolumeStatus(volume *ole.IDispatch) (bitlockerVolumeStatus, error) {
	var status bitlockerVolumeStatus

	// Get direct properties from the volume object
	deviceID, err := getStringProperty(volume, "DeviceID")
	if err != nil {
		return status, fmt.Errorf("failed to get DeviceID: %w", err)
	}
	status.DeviceID = deviceID

	driveLetter, err := getStringProperty(volume, "DriveLetter")
	if err != nil {
		return status, fmt.Errorf("failed to get DriveLetter: %w", err)
	}
	status.DriveLetter = driveLetter

	persistentVolumeID, err := getStringProperty(volume, "PersistentVolumeID")
	if err != nil {
		return status, fmt.Errorf("failed to get PersistentVolumeID: %w", err)
	}
	status.PersistentVolumeID = persistentVolumeID

	encryptionMethod, err := getInt64Property(volume, "EncryptionMethod")
	if err != nil {
		return status, fmt.Errorf("failed to get EncryptionMethod: %w", err)
	}
	status.EncryptionMethod = statusCode{
		Code: encryptionMethod,
		Text: encryptionMethodValues[encryptionMethod],
	}

	protectionStatus, err := getInt64Property(volume, "ProtectionStatus")
	if err != nil {
		return status, fmt.Errorf("failed to get ProtectionStatus: %w", err)
	}
	status.ProtectionStatus = statusCode{
		Code: protectionStatus,
		Text: protectionStatusValues[protectionStatus],
	}

	// Call GetVersion method - returns version info
	var versionOut ole.VARIANT
	ole.VariantInit(&versionOut)
	versionResult, err := oleutil.CallMethod(volume, "GetVersion", &versionOut)
	if err != nil {
		return status, fmt.Errorf("failed to call GetVersion: %w", err)
	}
	// Check return code (0 = success)
	if versionResult.Val == 0 {
		status.Version = statusCode{
			Code: versionOut.Val,
			Text: fveVersionValues[versionOut.Val],
		}
	}
	versionOut.Clear()

	// Call GetConversionStatus method - returns detailed encryption status
	var convStatusOut ole.VARIANT
	var encryptionPercentageOut ole.VARIANT
	var encryptionFlagsOut ole.VARIANT
	var wipingStatusOut ole.VARIANT
	var wipingPercentageOut ole.VARIANT
	ole.VariantInit(&convStatusOut)
	ole.VariantInit(&encryptionPercentageOut)
	ole.VariantInit(&encryptionFlagsOut)
	ole.VariantInit(&wipingStatusOut)
	ole.VariantInit(&wipingPercentageOut)

	conversionResult, err := oleutil.CallMethod(volume, "GetConversionStatus",
		&convStatusOut,
		&encryptionPercentageOut,
		&encryptionFlagsOut,
		&wipingStatusOut,
		&wipingPercentageOut,
	)
	if err != nil {
		return status, fmt.Errorf("failed to call GetConversionStatus: %w", err)
	}
	// Check return code (0 = success)
	if conversionResult.Val == 0 {
		status.ConversionStatus = conversionStatus{
			ConversionStatus: statusCode{
				Code: convStatusOut.Val,
				Text: conversionStatusValues[convStatusOut.Val],
			},
			WipingStatus: statusCode{
				Code: wipingStatusOut.Val,
				Text: wipingStatusValues[wipingStatusOut.Val],
			},
			WipingPercentage:     wipingPercentageOut.Val,
			EncryptionPercentage: encryptionPercentageOut.Val,
		}
	}
	convStatusOut.Clear()
	encryptionPercentageOut.Clear()
	encryptionFlagsOut.Clear()
	wipingStatusOut.Clear()
	wipingPercentageOut.Clear()

	// Call GetLockStatus method - returns whether volume is locked
	var lockStatusOut ole.VARIANT
	ole.VariantInit(&lockStatusOut)
	lockResult, err := oleutil.CallMethod(volume, "GetLockStatus", &lockStatusOut)
	if err != nil {
		return status, fmt.Errorf("failed to call GetLockStatus: %w", err)
	}
	// Check return code (0 = success)
	if lockResult.Val == 0 {
		status.LockStatus = lockStatusOut.Val
	}
	lockStatusOut.Clear()

	return status, nil
}

// getStringProperty safely gets a string property from a WMI object
func getStringProperty(obj *ole.IDispatch, name string) (string, error) {
	val, err := oleutil.GetProperty(obj, name)
	if err != nil {
		return "", err
	}
	if val.Val == 0 {
		return "", nil
	}
	return val.ToString(), nil
}

// getInt64Property safely gets an int64 property from a WMI object
func getInt64Property(obj *ole.IDispatch, name string) (int64, error) {
	val, err := oleutil.GetProperty(obj, name)
	if err != nil {
		return 0, err
	}
	return val.Val, nil
}
