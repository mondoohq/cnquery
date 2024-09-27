// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package windows

import (
	"encoding/json"
	"io"
)

// PSGetComputerInfoShort is a PowerShell script that retrieves computer information.
// This is inteded to be used only as fallback when PSGetComputerInfo fails because
// the command is too long.
const PSGetComputerInfoShort = `Get-ComputerInfo | ConvertTo-Json`

// PSGetComputerInfo is a PowerShell script that retrieves computer information. It also
// implements a fallback to work on systems with winrm disabled. See https://github.com/mondoohq/cnquery/pull/4520
// for more information.
const PSGetComputerInfo = `
function Get-CustomComputerInfo {
    $bios = Get-CimInstance -ClassName Win32_BIOS
    $computerSystem = Get-CimInstance -ClassName Win32_ComputerSystem
    $os = Get-CimInstance -ClassName Win32_OperatingSystem
    $timeZone = Get-CimInstance -ClassName Win32_TimeZone
    $windowsProduct = Get-ItemProperty "HKLM:\Software\Microsoft\Windows NT\CurrentVersion"
    $firmwareType = Get-CimInstance -Namespace root\cimv2 -ClassName Win32_ComputerSystem | Select-Object -ExpandProperty FirmwareType
    
    $result = [PSCustomObject]@{
        BiosBIOSVersion = $bios.SMBIOSBIOSVersion
        BiosCaption = $bios.Caption
        BiosCharacteristics = $bios.BiosCharacteristics
        BiosCurrentLanguage = $bios.CurrentLanguage
        BiosDescription = $bios.Description
        BiosEmbeddedControllerMajorVersion = $bios.EmbeddedControllerMajorVersion
        BiosEmbeddedControllerMinorVersion = $bios.EmbeddedControllerMinorVersion
        BiosFirmwareType = $firmwareType
        BiosIdentificationCode = $bios.IdentificationCode
        BiosInstallDate = $bios.InstallDate
        BiosInstallableLanguages = $bios.InstallableLanguages
        BiosLanguageEdition = $bios.LanguageEdition
        BiosListOfLanguages = $bios.ListOfLanguages
        BiosManufacturer = $bios.Manufacturer
        BiosName = $bios.Name
        BiosOtherTargetOS = $bios.OtherTargetOS
        BiosPrimaryBIOS = $bios.PrimaryBIOS
        BiosReleaseDate = $bios.ReleaseDate
        BiosSMBIOSBIOSVersion = $bios.SMBIOSBIOSVersion
        BiosSMBIOSMajorVersion = $bios.SMBIOSMajorVersion
        BiosSMBIOSMinorVersion = $bios.SMBIOSMinorVersion
        BiosSMBIOSPresent = $bios.SMBIOSPresent
        BiosSeralNumber = $bios.SerialNumber
        BiosSoftwareElementState = $bios.SoftwareElementState
        BiosStatus = $bios.Status
        BiosSystemBiosMajorVersion = $bios.SystemBiosMajorVersion
        BiosSystemBiosMinorVersion = $bios.SystemBiosMinorVersion
        BiosTargetOperatingSystem = $bios.TargetOperatingSystem
        BiosVersion = $bios.Version

        CsAdminPasswordStatus = $computerSystem.AdminPasswordStatus
        CsAutomaticManagedPagefile = $computerSystem.AutomaticManagedPagefile
        CsAutomaticResetBootOption = $computerSystem.AutomaticResetBootOption
        CsAutomaticResetCapability = $computerSystem.AutomaticResetCapability
        CsBootOptionOnLimit = $computerSystem.BootOptionOnLimit
        CsBootOptionOnWatchDog = $computerSystem.BootOptionOnWatchDog
        CsBootROMSupported = $computerSystem.BootROMSupported
        CsBootStatus = $computerSystem.BootStatus
        CsBootupState = $computerSystem.BootupState
        CsCaption = $computerSystem.Caption
        CsChassisBootupState = $computerSystem.ChassisBootupState
        CsChassisSKUNumber = $computerSystem.SKUNumber
        CsCurrentTimeZone = $timeZone.StandardName
        CsDNSHostName = $computerSystem.DNSHostName
        CsDaylightInEffect = $timeZone.DaylightInEffect
        CsDescription = $computerSystem.Description
        CsDomain = $computerSystem.Domain
        CsDomainRole = $computerSystem.DomainRole
        CsEnableDaylightSavingsTime = $computerSystem.EnableDaylightSavingsTime
        CsFrontPanelResetStatus = $computerSystem.FrontPanelResetStatus
        CsHypervisorPresent = $computerSystem.HypervisorPresent
        CsInfraredSupported = $computerSystem.InfraredSupported
        CsInitialLoadInfo = $computerSystem.InitialLoadInfo
        CsInstallDate = $computerSystem.InstallDate
        CsKeyboardPasswordStatus = $computerSystem.KeyboardPasswordStatus
        CsLastLoadInfo = $computerSystem.LastLoadInfo
        CsManufacturer = $computerSystem.Manufacturer
        CsModel = $computerSystem.Model
        CsName = $computerSystem.Name
        CsNetworkServerModeEnabled = $computerSystem.NetworkServerModeEnabled
        CsNumberOfLogicalProcessors = $computerSystem.NumberOfLogicalProcessors
        CsNumberOfProcessors = $computerSystem.NumberOfProcessors
        CsOEMStringArray = $computerSystem.OEMStringArray
        CsPCSystemType = $computerSystem.PCSystemType
        CsPCSystemTypeEx = $computerSystem.PCSystemTypeEx
        CsPartOfDomain = $computerSystem.PartOfDomain
        CsPauseAfterReset = $computerSystem.PauseAfterReset
        CsPhyicallyInstalledMemory = $computerSystem.TotalPhysicalMemory
        CsPowerManagementCapabilities = $computerSystem.PowerManagementCapabilities
        CsPowerManagementSupported = $computerSystem.PowerManagementSupported
        CsPowerOnPasswordStatus = $computerSystem.PowerOnPasswordStatus
        CsPowerState = $computerSystem.PowerState
        CsPowerSupplyState = $computerSystem.PowerSupplyState
        CsPrimaryOwnerContact = $computerSystem.PrimaryOwnerContact
        CsPrimaryOwnerName = $computerSystem.PrimaryOwnerName
        CsProcessors = $computerSystem.Processor
        CsResetCapability = $computerSystem.ResetCapability
        CsResetCount = $computerSystem.ResetCount
        CsResetLimit = $computerSystem.ResetLimit
        CsRoles = $computerSystem.Roles
        CsStatus = $computerSystem.Status
        CsSupportContactDescription = $computerSystem.SupportContactDescription
        CsSystemFamily = $computerSystem.SystemFamily
        CsSystemSKUNumber = $computerSystem.SystemSKUNumber
        CsSystemType = $computerSystem.SystemType
        CsThermalState = $computerSystem.ThermalState
        CsTotalPhysicalMemory = $computerSystem.TotalPhysicalMemory
        CsUserName = $computerSystem.UserName
        CsWakeUpType = $computerSystem.WakeUpType
        CsWorkgroup = $computerSystem.Workgroup

        OsArchitecture = $os.OSArchitecture
        OsBootDevice = $os.BootDevice
        OsBuildNumber = $os.BuildNumber
        OsBuildType = $os.BuildType
        OsCSDVersion = $os.CSDVersion
        OsCodeSet = $os.CodeSet
        OsCountryCode = $os.CountryCode
        OsCurrentTimeZone = $os.CurrentTimeZone
        OsDataExecutionPrevention32BitApplications = $os.DataExecutionPrevention_32BitApplications
        OsDataExecutionPreventionAvailable = $os.DataExecutionPrevention_Available
        OsDataExecutionPreventionDrivers = $os.DataExecutionPrevention_Drivers
        OsDataExecutionPreventionSupportPolicy = $os.DataExecutionPrevention_SupportPolicy
        OsDebug = $os.Debug
        OsDistributed = $os.Distributed
        OsEncryptionLevel = $os.EncryptionLevel
        OsForegroundApplicationBoost = $os.ForegroundApplicationBoost
        OsFreePhysicalMemory = $os.FreePhysicalMemory
        OsFreeSpaceInPagingFiles = $os.FreeSpaceInPagingFiles
        OsFreeVirtualMemory = $os.FreeVirtualMemory
        OsHardwareAbstractionLayer = $os.Version
        OsHotFixes = $os.HotFixes
        OsInUseVirtualMemory = $os.InUseVirtualMemory
        OsInstallDate = $os.InstallDate
        OsLanguage = $os.OSLanguage
        OsLastBootUpTime = $os.LastBootUpTime
        OsLocalDateTime = $os.LocalDateTime
        OsLocale = $os.Locale
        OsLocaleID = $os.LocaleID
        OsManufacturer = $os.Manufacturer
        OsMaxNumberOfProcesses = $os.MaxNumberOfProcesses
        OsMaxProcessMemorySize = $os.MaxProcessMemorySize
        OsMuiLanguages = $os.MUILanguages
        OsName = $os.Name
        OsNumberOfLicensedUsers = $os.NumberOfLicensedUsers
        OsNumberOfProcesses = $os.NumberOfProcesses
        OsNumberOfUsers = $os.NumberOfUsers
        OsOperatingSystemSKU = $os.OperatingSystemSKU
        OsOrganization = $os.Organization
        OsOtherTypeDescription = $os.OtherTypeDescription
        OsPAEEnabled = $os.PAEEnabled
        OsPagingFiles = $os.PagingFiles
        OsPortableOperatingSystem = $os.PortableOperatingSystem
        OsPrimary = $os.Primary
        OsProductSuites = $os.ProductSuites
        OsProductType = $os.ProductType
        OsRegisteredUser = $os.RegisteredUser
        OsSerialNumber = $os.SerialNumber
        OsServerLevel = $os.ServerLevel
        OsServicePackMajorVersion = $os.ServicePackMajorVersion
        OsServicePackMinorVersion = $os.ServicePackMinorVersion
        OsSizeStoredInPagingFiles = $os.SizeStoredInPagingFiles
        OsStatus = $os.Status
        OsSuites = $os.Suites
        OsSystemDevice = $os.SystemDevice
        OsSystemDirectory = $os.SystemDirectory
        OsSystemDrive = $os.SystemDrive
        OsTotalSwapSpaceSize = $os.TotalSwapSpaceSize
        OsTotalVirtualMemorySize = $os.TotalVirtualMemorySize
        OsTotalVisibleMemorySize = $os.TotalVisibleMemorySize
        OsType = $os.OSType
        OsUptime = $os.LastBootUpTime
        OsVersion = $os.Version
        OsWindowsDirectory = $os.WindowsDirectory

        TimeZone = $timeZone.StandardName
        WindowsBuildLabEx = $windowsProduct.BuildLabEx
        WindowsCurrentVersion = $windowsProduct.CurrentVersion
        WindowsEditionId = $windowsProduct.EditionID
        WindowsInstallDateFromRegistry = $windowsProduct.InstallDate
        WindowsInstallationType = $windowsProduct.InstallationType
        WindowsProductId = $windowsProduct.ProductId
        WindowsProductName = $windowsProduct.ProductName
        WindowsRegisteredOrganization = $windowsProduct.RegisteredOrganization
        WindowsRegisteredOwner = $windowsProduct.RegisteredOwner
        WindowsSystemRoot = $windowsProduct.SystemRoot
    }

    return $result
}

function Get-ComputerInfoWithFallback {
    $computerInfo = Get-ComputerInfo

    if ($computerInfo.OsProductType -eq $null) {
        return Get-CustomComputerInfo
    } else {
        return $computerInfo
    }
}

Get-ComputerInfoWithFallback | ConvertTo-Json
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
