package platform_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mondoo.io/mondoo/motor/platform"
)

func TestWindowsWmic(t *testing.T) {

	osVersion :=
		`Node,BootDevice,BuildNumber,BuildType,Caption,CodeSet,CountryCode,CreationClassName,CSCreationClassName,CSDVersion,CSName,CurrentTimeZone,DataExecutionPrevention_32BitApplications,DataExecutionPrevention_Available,DataExecutionPrevention_Drivers,DataExecutionPrevention_SupportPolicy,Debug,Description,Distributed,EncryptionLevel,ForegroundApplicationBoost,FreePhysicalMemory,FreeSpaceInPagingFiles,FreeVirtualMemory,InstallDate,LargeSystemCache,LastBootUpTime,LocalDateTime,Locale,Manufacturer,MaxNumberOfProcesses,MaxProcessMemorySize,MUILanguages,Name,NumberOfLicensedUsers,NumberOfProcesses,NumberOfUsers,OperatingSystemSKU,Organization,OSArchitecture,OSLanguage,OSProductSuite,OSType,OtherTypeDescription,PAEEnabled,PlusProductID,PlusVersionNumber,PortableOperatingSystem,Primary,ProductType,RegisteredUser,SerialNumber,ServicePackMajorVersion,ServicePackMinorVersion,SizeStoredInPagingFiles,Status,SuiteMask,SystemDevice,SystemDirectory,SystemDrive,TotalSwapSpaceSize,TotalVirtualMemorySize,TotalVisibleMemorySize,Version,WindowsDirectory
VAGRANT-2016,\\Device\\HarddiskVolume1,14393,Multiprocessor Free,Microsoft Windows Server 2016 Standard Evaluation,1252,1,Win32_OperatingSystem,Win32_ComputerSystem,,VAGRANT-2016,-420,TRUE,TRUE,TRUE,3,FALSE,,FALSE,256,2,1629000,1179648,2833804,20180313201557.000000-420,,20180630024418.280385-420,20180630024734.124000-420,0409,Microsoft Corporation,4294967295,137438953344,{en-US},Microsoft Windows Server 2016 Standard Evaluation|C:\\Windows|\\Device\\Harddisk0\\Partition2,0,35,1,79,Vagrant,64-bit,1033,272,18,,,,,FALSE,TRUE,3,,00378-00000-00000-AA739,0,0,1179648,OK,272,\\Device\\HarddiskVolume2,C:\\Windows\\system32,C:,,3276340,2096692,10.0.14393,C:\\Windows
`

	m, err := platform.ParseWinWmicOS(strings.NewReader(osVersion))
	assert.Nil(t, err)

	assert.Equal(t, "14393", m.BuildNumber, "buildnumber should be parsed properly")
	assert.Equal(t, "10.0.14393", m.Version, "version should be parsed properly")
	assert.Equal(t, "Microsoft Windows Server 2016 Standard Evaluation", m.Caption, "caption should be parsed properly")
	assert.Equal(t, "VAGRANT-2016", m.Node, "node should be parsed properly")
	assert.Equal(t, "", m.Description, "description should be parsed properly")
	assert.Equal(t, "Microsoft Corporation", m.Manufacturer, "manufacturer should be parsed properly")
	assert.Equal(t, "64-bit", m.OSArchitecture, "os architecture should be parsed properly")
	assert.Equal(t, "18", m.OSType, "os type should be parsed properly")
	assert.Equal(t, "3", m.ProductType, "product type should be parsed properly")
}
