// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package windows

import (
	"encoding/json"
	"io"
)

// PSGetComputerInfo is a PowerShell script that retrieves computer information.
const PSGetComputerInfo = `Get-ComputerInfo | ConvertTo-Json`

// PSGetComputerInfoCustom is a PowerShell script that retrieves computer information. It
// implements a fallback to work on systems with winrm disabled. See
// https://github.com/mondoohq/cnquery/pull/4520 for more information.
const PSGetComputerInfoCustom = `
function Get-CustomComputerInfo {
    $bios = Get-CimInstance -ClassName Win32_BIOS
    $computerSystem = Get-CimInstance -ClassName Win32_ComputerSystem
    $os = Get-CimInstance -ClassName Win32_OperatingSystem
    $timeZone = Get-CimInstance -ClassName Win32_TimeZone
    $windowsProduct = Get-ItemProperty "HKLM:\Software\Microsoft\Windows NT\CurrentVersion"
    $firmwareType = Get-CimInstance -Namespace root\cimv2 -ClassName Win32_ComputerSystem | Select-Object -ExpandProperty FirmwareType
    $result = [PSCustomObject]@{
        Bios = $bios
        ComputerSystem = $computerSystem
        Os = $os
        TimeZone = $timeZone
        WindowsProduct = $windowsProduct
        FirmwareType = $firmwareType
    }
    return $result
}
Get-CustomComputerInfo | ConvertTo-Json
`

func ParseComputerInfo(r io.Reader) (map[string]interface{}, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	var properties map[string]interface{}
	err = json.Unmarshal(data, &properties)
	if err != nil {
		return nil, err
	}

	return properties, nil
}

type CustomComputerInfo struct {
	Bios           map[string]interface{} `json:"Bios"`
	ComputerSystem map[string]interface{} `json:"ComputerSystem"`
	Os             map[string]interface{} `json:"Os"`
	TimeZone       map[string]interface{} `json:"TimeZone"`
	WindowsProduct map[string]interface{} `json:"WindowsProduct"`
}

func ParseCustomComputerInfo(r io.Reader) (map[string]interface{}, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	customComputerInfo := &CustomComputerInfo{}
	err = json.Unmarshal(data, customComputerInfo)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"BiosBIOSVersion":                    customComputerInfo.Bios["SMBIOSBIOSVersion"],
		"BiosCaption":                        customComputerInfo.Bios["Caption"],
		"BiosCharacteristics":                customComputerInfo.Bios["BiosCharacteristics"],
		"BiosCurrentLanguage":                customComputerInfo.Bios["CurrentLanguage"],
		"BiosDescription":                    customComputerInfo.Bios["Description"],
		"BiosEmbeddedControllerMajorVersion": customComputerInfo.Bios["EmbeddedControllerMajorVersion"],
		"BiosEmbeddedControllerMinorVersion": customComputerInfo.Bios["EmbeddedControllerMinorVersion"],
		"BiosFirmwareType":                   customComputerInfo.WindowsProduct["FirmwareType"],
		"BiosIdentificationCode":             customComputerInfo.Bios["IdentificationCode"],
		"BiosInstallDate":                    customComputerInfo.Bios["InstallDate"],
		"BiosInstallableLanguages":           customComputerInfo.Bios["InstallableLanguages"],
		"BiosLanguageEdition":                customComputerInfo.Bios["LanguageEdition"],
		"BiosListOfLanguages":                customComputerInfo.Bios["ListOfLanguages"],
		"BiosManufacturer":                   customComputerInfo.Bios["Manufacturer"],
		"BiosName":                           customComputerInfo.Bios["Name"],
		"BiosOtherTargetOS":                  customComputerInfo.Bios["OtherTargetOS"],
		"BiosPrimaryBIOS":                    customComputerInfo.Bios["PrimaryBIOS"],
		"BiosReleaseDate":                    customComputerInfo.Bios["ReleaseDate"],
		"BiosSMBIOSBIOSVersion":              customComputerInfo.Bios["SMBIOSBIOSVersion"],
		"BiosSMBIOSMajorVersion":             customComputerInfo.Bios["SMBIOSMajorVersion"],
		"BiosSMBIOSMinorVersion":             customComputerInfo.Bios["SMBIOSMinorVersion"],
		"BiosSMBIOSPresent":                  customComputerInfo.Bios["SMBIOSPresent"],
		"BiosSerialNumber":                   customComputerInfo.Bios["SerialNumber"],
		"BiosSoftwareElementState":           customComputerInfo.Bios["SoftwareElementState"],
		"BiosStatus":                         customComputerInfo.Bios["Status"],
		"BiosSystemBiosMajorVersion":         customComputerInfo.Bios["SystemBiosMajorVersion"],
		"BiosSystemBiosMinorVersion":         customComputerInfo.Bios["SystemBiosMinorVersion"],
		"BiosTargetOperatingSystem":          customComputerInfo.Bios["TargetOperatingSystem"],
		"BiosVersion":                        customComputerInfo.Bios["Version"],

		"CsAdminPasswordStatus":         customComputerInfo.ComputerSystem["AdminPasswordStatus"],
		"CsAutomaticManagedPagefile":    customComputerInfo.ComputerSystem["AutomaticManagedPagefile"],
		"CsAutomaticResetBootOption":    customComputerInfo.ComputerSystem["AutomaticResetBootOption"],
		"CsAutomaticResetCapability":    customComputerInfo.ComputerSystem["AutomaticResetCapability"],
		"CsBootOptionOnLimit":           customComputerInfo.ComputerSystem["BootOptionOnLimit"],
		"CsBootOptionOnWatchDog":        customComputerInfo.ComputerSystem["BootOptionOnWatchDog"],
		"CsBootROMSupported":            customComputerInfo.ComputerSystem["BootROMSupported"],
		"CsBootStatus":                  customComputerInfo.ComputerSystem["BootStatus"],
		"CsBootupState":                 customComputerInfo.ComputerSystem["BootupState"],
		"CsCaption":                     customComputerInfo.ComputerSystem["Caption"],
		"CsChassisBootupState":          customComputerInfo.ComputerSystem["ChassisBootupState"],
		"CsChassisSKUNumber":            customComputerInfo.ComputerSystem["SKUNumber"],
		"CsCurrentTimeZone":             customComputerInfo.TimeZone["StandardName"],
		"CsDNSHostName":                 customComputerInfo.ComputerSystem["DNSHostName"],
		"CsDaylightInEffect":            customComputerInfo.TimeZone["DaylightInEffect"],
		"CsDescription":                 customComputerInfo.ComputerSystem["Description"],
		"CsDomain":                      customComputerInfo.ComputerSystem["Domain"],
		"CsDomainRole":                  customComputerInfo.ComputerSystem["DomainRole"],
		"CsEnableDaylightSavingsTime":   customComputerInfo.ComputerSystem["EnableDaylightSavingsTime"],
		"CsFrontPanelResetStatus":       customComputerInfo.ComputerSystem["FrontPanelResetStatus"],
		"CsHypervisorPresent":           customComputerInfo.ComputerSystem["HypervisorPresent"],
		"CsInfraredSupported":           customComputerInfo.ComputerSystem["InfraredSupported"],
		"CsInitialLoadInfo":             customComputerInfo.ComputerSystem["InitialLoadInfo"],
		"CsInstallDate":                 customComputerInfo.ComputerSystem["InstallDate"],
		"CsKeyboardPasswordStatus":      customComputerInfo.ComputerSystem["KeyboardPasswordStatus"],
		"CsLastLoadInfo":                customComputerInfo.ComputerSystem["LastLoadInfo"],
		"CsManufacturer":                customComputerInfo.ComputerSystem["Manufacturer"],
		"CsModel":                       customComputerInfo.ComputerSystem["Model"],
		"CsName":                        customComputerInfo.ComputerSystem["Name"],
		"CsNetworkServerModeEnabled":    customComputerInfo.ComputerSystem["NetworkServerModeEnabled"],
		"CsNumberOfLogicalProcessors":   customComputerInfo.ComputerSystem["NumberOfLogicalProcessors"],
		"CsNumberOfProcessors":          customComputerInfo.ComputerSystem["NumberOfProcessors"],
		"CsOEMStringArray":              customComputerInfo.ComputerSystem["OEMStringArray"],
		"CsPCSystemType":                customComputerInfo.ComputerSystem["PCSystemType"],
		"CsPCSystemTypeEx":              customComputerInfo.ComputerSystem["PCSystemTypeEx"],
		"CsPartOfDomain":                customComputerInfo.ComputerSystem["PartOfDomain"],
		"CsPauseAfterReset":             customComputerInfo.ComputerSystem["PauseAfterReset"],
		"CsPhyicallyInstalledMemory":    customComputerInfo.ComputerSystem["TotalPhysicalMemory"],
		"CsPowerManagementCapabilities": customComputerInfo.ComputerSystem["PowerManagementCapabilities"],
		"CsPowerManagementSupported":    customComputerInfo.ComputerSystem["PowerManagementSupported"],
		"CsPowerOnPasswordStatus":       customComputerInfo.ComputerSystem["PowerOnPasswordStatus"],
		"CsPowerState":                  customComputerInfo.ComputerSystem["PowerState"],
		"CsPowerSupplyState":            customComputerInfo.ComputerSystem["PowerSupplyState"],
		"CsPrimaryOwnerContact":         customComputerInfo.ComputerSystem["PrimaryOwnerContact"],
		"CsPrimaryOwnerName":            customComputerInfo.ComputerSystem["PrimaryOwnerName"],
		"CsProcessors":                  customComputerInfo.ComputerSystem["Processor"],
		"CsResetCapability":             customComputerInfo.ComputerSystem["ResetCapability"],
		"CsResetCount":                  customComputerInfo.ComputerSystem["ResetCount"],
		"CsResetLimit":                  customComputerInfo.ComputerSystem["ResetLimit"],
		"CsRoles":                       customComputerInfo.ComputerSystem["Roles"],
		"CsStatus":                      customComputerInfo.ComputerSystem["Status"],
		"CsSupportContactDescription":   customComputerInfo.ComputerSystem["SupportContactDescription"],
		"CsSystemFamily":                customComputerInfo.ComputerSystem["SystemFamily"],
		"CsSystemSKUNumber":             customComputerInfo.ComputerSystem["SystemSKUNumber"],
		"CsSystemType":                  customComputerInfo.ComputerSystem["SystemType"],
		"CsThermalState":                customComputerInfo.ComputerSystem["ThermalState"],
		"CsTotalPhysicalMemory":         customComputerInfo.ComputerSystem["TotalPhysicalMemory"],
		"CsUserName":                    customComputerInfo.ComputerSystem["UserName"],
		"CsWakeUpType":                  customComputerInfo.ComputerSystem["WakeUpType"],
		"CsWorkgroup":                   customComputerInfo.ComputerSystem["Workgroup"],

		"OsArchitecture":    customComputerInfo.Os["OSArchitecture"],
		"OsBootDevice":      customComputerInfo.Os["BootDevice"],
		"OsBuildNumber":     customComputerInfo.Os["BuildNumber"],
		"OsBuildType":       customComputerInfo.Os["BuildType"],
		"OsCSDVersion":      customComputerInfo.Os["CSDVersion"],
		"OsCodeSet":         customComputerInfo.Os["CodeSet"],
		"OsCountryCode":     customComputerInfo.Os["CountryCode"],
		"OsCurrentTimeZone": customComputerInfo.Os["CurrentTimeZone"],
		"OsDataExecutionPrevention32BitApplications": customComputerInfo.Os["DataExecutionPrevention_32BitApplications"],
		"OsDataExecutionPreventionAvailable":         customComputerInfo.Os["DataExecutionPrevention_Available"],
		"OsDataExecutionPreventionDrivers":           customComputerInfo.Os["DataExecutionPrevention_Drivers"],
		"OsDataExecutionPreventionSupportPolicy":     customComputerInfo.Os["DataExecutionPrevention_SupportPolicy"],
		"OsDebug":                                    customComputerInfo.Os["Debug"],
		"OsDistributed":                              customComputerInfo.Os["Distributed"],
		"OsEncryptionLevel":                          customComputerInfo.Os["EncryptionLevel"],
		"OsForegroundApplicationBoost":               customComputerInfo.Os["ForegroundApplicationBoost"],
		"OsFreePhysicalMemory":                       customComputerInfo.Os["FreePhysicalMemory"],
		"OsFreeSpaceInPagingFiles":                   customComputerInfo.Os["FreeSpaceInPagingFiles"],
		"OsFreeVirtualMemory":                        customComputerInfo.Os["FreeVirtualMemory"],
		"OsHardwareAbstractionLayer":                 customComputerInfo.Os["Version"],
		"OsHotFixes":                                 customComputerInfo.Os["HotFixes"],
		"OsInUseVirtualMemory":                       customComputerInfo.Os["InUseVirtualMemory"],
		"OsInstallDate":                              customComputerInfo.Os["InstallDate"],
		"OsLanguage":                                 customComputerInfo.Os["OSLanguage"],
		"OsLastBootUpTime":                           customComputerInfo.Os["LastBootUpTime"],
		"OsLocalDateTime":                            customComputerInfo.Os["LocalDateTime"],
		"OsLocale":                                   customComputerInfo.Os["Locale"],
		"OsLocaleID":                                 customComputerInfo.Os["LocaleID"],
		"OsManufacturer":                             customComputerInfo.Os["Manufacturer"],
		"OsMaxNumberOfProcesses":                     customComputerInfo.Os["MaxNumberOfProcesses"],
		"OsMaxProcessMemorySize":                     customComputerInfo.Os["MaxProcessMemorySize"],
		"OsMuiLanguages":                             customComputerInfo.Os["MUILanguages"],
		"OsName":                                     customComputerInfo.Os["Name"],
		"OsNumberOfLicensedUsers":                    customComputerInfo.Os["NumberOfLicensedUsers"],
		"OsNumberOfProcesses":                        customComputerInfo.Os["NumberOfProcesses"],
		"OsNumberOfUsers":                            customComputerInfo.Os["NumberOfUsers"],
		"OsOperatingSystemSKU":                       customComputerInfo.Os["OperatingSystemSKU"],
		"OsOrganization":                             customComputerInfo.Os["Organization"],
		"OsOtherTypeDescription":                     customComputerInfo.Os["OtherTypeDescription"],
		"OsPAEEnabled":                               customComputerInfo.Os["PAEEnabled"],
		"OsPagingFiles":                              customComputerInfo.Os["PagingFiles"],
		"OsPortableOperatingSystem":                  customComputerInfo.Os["PortableOperatingSystem"],
		"OsPrimary":                                  customComputerInfo.Os["Primary"],
		"OsProductSuites":                            customComputerInfo.Os["ProductSuites"],
		"OsProductType":                              customComputerInfo.Os["ProductType"],
		"OsRegisteredUser":                           customComputerInfo.Os["RegisteredUser"],
		"OsSerialNumber":                             customComputerInfo.Os["SerialNumber"],
		"OsServerLevel":                              customComputerInfo.Os["ServerLevel"],
		"OsServicePackMajorVersion":                  customComputerInfo.Os["ServicePackMajorVersion"],
		"OsServicePackMinorVersion":                  customComputerInfo.Os["ServicePackMinorVersion"],
		"OsSizeStoredInPagingFiles":                  customComputerInfo.Os["SizeStoredInPagingFiles"],
		"OsStatus":                                   customComputerInfo.Os["Status"],
		"OsSuites":                                   customComputerInfo.Os["Suites"],
		"OsSystemDevice":                             customComputerInfo.Os["SystemDevice"],
		"OsSystemDirectory":                          customComputerInfo.Os["SystemDirectory"],
		"OsSystemDrive":                              customComputerInfo.Os["SystemDrive"],
		"OsTotalSwapSpaceSize":                       customComputerInfo.Os["TotalSwapSpaceSize"],
		"OsTotalVirtualMemorySize":                   customComputerInfo.Os["TotalVirtualMemorySize"],
		"OsTotalVisibleMemorySize":                   customComputerInfo.Os["TotalVisibleMemorySize"],
		"OsType":                                     customComputerInfo.Os["OSType"],
		"OsUptime":                                   customComputerInfo.Os["LastBootUpTime"],
		"OsVersion":                                  customComputerInfo.Os["Version"],
		"OsWindowsDirectory":                         customComputerInfo.Os["WindowsDirectory"],

		"TimeZone":                       customComputerInfo.TimeZone["StandardName"],
		"WindowsBuildLabEx":              customComputerInfo.WindowsProduct["BuildLabEx"],
		"WindowsCurrentVersion":          customComputerInfo.WindowsProduct["CurrentVersion"],
		"WindowsEditionId":               customComputerInfo.WindowsProduct["EditionID"],
		"WindowsInstallDateFromRegistry": customComputerInfo.WindowsProduct["InstallDate"],
		"WindowsInstallationType":        customComputerInfo.WindowsProduct["InstallationType"],
		"WindowsProductId":               customComputerInfo.WindowsProduct["ProductId"],
		"WindowsProductName":             customComputerInfo.WindowsProduct["ProductName"],
		"WindowsRegisteredOrganization":  customComputerInfo.WindowsProduct["RegisteredOrganization"],
		"WindowsRegisteredOwner":         customComputerInfo.WindowsProduct["RegisteredOwner"],
		"WindowsSystemRoot":              customComputerInfo.WindowsProduct["SystemRoot"],
	}, nil
}
