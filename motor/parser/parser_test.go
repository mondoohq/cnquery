package parser

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOSReleaseParser(t *testing.T) {

	osRelease := `NAME="Ubuntu"
VERSION="16.04.3 LTS (Xenial Xerus)"
ID=ubuntu
ID_LIKE=debian
PRETTY_NAME="Ubuntu 16.04.3 LTS"
VERSION_ID="16.04"
HOME_URL="http://www.ubuntu.com/"
SUPPORT_URL="http://help.ubuntu.com/"
BUG_REPORT_URL="http://bugs.launchpad.net/ubuntu/"
VERSION_CODENAME=xenial
UBUNTU_CODENAME=xenial`

	m, err := ParseOsRelease(osRelease)
	assert.Nil(t, err)

	assert.Equal(t, "Ubuntu", m["NAME"], "NAME should be parsed properly")
	assert.Equal(t, "16.04.3 LTS (Xenial Xerus)", m["VERSION"], "VERSION should be parsed properly")
	assert.Equal(t, "ubuntu", m["ID"], "ID should be parsed properly")
	assert.Equal(t, "debian", m["ID_LIKE"], "ID_LIKE should be parsed properly")
	assert.Equal(t, "Ubuntu 16.04.3 LTS", m["PRETTY_NAME"], "PRETTY_NAME should be parsed properly")
	assert.Equal(t, "16.04", m["VERSION_ID"], "VERSION_ID should be parsed properly")
	assert.Equal(t, "http://www.ubuntu.com/", m["HOME_URL"], "HOME_URL should be parsed properly")
	assert.Equal(t, "http://help.ubuntu.com/", m["SUPPORT_URL"], "SUPPORT_URL should be parsed properly")
	assert.Equal(t, "http://bugs.launchpad.net/ubuntu/", m["BUG_REPORT_URL"], "BUG_REPORT_URL should be parsed properly")
	assert.Equal(t, "xenial", m["VERSION_CODENAME"], "VERSION_CODENAME should be parsed properly")
	assert.Equal(t, "xenial", m["UBUNTU_CODENAME"], "UBUNTU_CODENAME should be parsed properly")

	osRelease = `NAME="Oracle Linux Server"
VERSION="6.9"
ID="ol"
VERSION_ID="6.9"
PRETTY_NAME="Oracle Linux Server 6.9"
ANSI_COLOR="0;31"
CPE_NAME="cpe:/o:oracle:linux:6:9:server"
HOME_URL="https://linux.oracle.com/"
BUG_REPORT_URL="https://bugzilla.oracle.com/"

ORACLE_BUGZILLA_PRODUCT="Oracle Linux 6"
ORACLE_BUGZILLA_PRODUCT_VERSION=6.9
ORACLE_SUPPORT_PRODUCT="Oracle Linux"
ORACLE_SUPPORT_PRODUCT_VERSION=6.9`

	m, err = ParseOsRelease(osRelease)
	assert.Equal(t, "Oracle Linux Server", m["NAME"], "NAME should be parsed properly")
	assert.Equal(t, "ol", m["ID"], "ID should be parsed properly")
	assert.Equal(t, "6.9", m["VERSION"], "VERSION should be parsed properly")
}

func TestEtcLsbReleaseParser(t *testing.T) {

	lsbRelease := `DISTRIB_ID=Ubuntu
DISTRIB_RELEASE=16.04
DISTRIB_CODENAME=xenial
DISTRIB_DESCRIPTION="Ubuntu 16.04.3 LTS"`

	m, err := ParseLsbRelease(lsbRelease)
	assert.Nil(t, err)

	assert.Equal(t, "Ubuntu", m["DISTRIB_ID"], "DISTRIB_ID should be parsed properly")
	assert.Equal(t, "16.04", m["DISTRIB_RELEASE"], "DISTRIB_RELEASE should be parsed properly")
	assert.Equal(t, "xenial", m["DISTRIB_CODENAME"], "DISTRIB_CODENAME should be parsed properly")
	assert.Equal(t, "Ubuntu 16.04.3 LTS", m["DISTRIB_DESCRIPTION"], "DISTRIB_DESCRIPTION should be parsed properly")
}

func TestRedhatRelease(t *testing.T) {
	rhRelease := "CentOS Linux release 7.4.1708 (Core)"
	name, release, err := ParseRhelVersion(rhRelease)
	assert.Nil(t, err)
	assert.Equal(t, "CentOS Linux", name, "parse os name")
	assert.Equal(t, "7.4.1708", release, "parse release version")

	rhRelease = "CentOS release 6.9 (Final)"
	name, release, err = ParseRhelVersion(rhRelease)
	assert.Nil(t, err)
	assert.Equal(t, "CentOS", name, "parse os name")
	assert.Equal(t, "6.9", release, "parse release version")

	rhRelease = "Red Hat Enterprise Linux Server release 7.4 (Maipo)"
	name, release, err = ParseRhelVersion(rhRelease)
	assert.Nil(t, err)
	assert.Equal(t, "Red Hat Enterprise Linux Server", name, "parse os name")
	assert.Equal(t, "7.4", release, "parse release version")

	rhRelease = "Oracle Linux Server release 7.4 (Maipo)"
	name, release, err = ParseRhelVersion(rhRelease)
	assert.Nil(t, err)
	assert.Equal(t, "Oracle Linux Server", name, "parse os name")
	assert.Equal(t, "7.4", release, "parse release version")

}

func TestDarwinRelease(t *testing.T) {
	swVers := `ProductName:	Mac OS X
ProductVersion:	10.13.2
BuildVersion:	17C88
	`

	m, err := ParseDarwinRelease(swVers)
	assert.Nil(t, err)

	assert.Equal(t, "Mac OS X", m["ProductName"], "ProductName should be parsed properly")
	assert.Equal(t, "10.13.2", m["ProductVersion"], "ProductVersion should be parsed properly")
	assert.Equal(t, "17C88", m["BuildVersion"], "BuildVersion should be parsed properly")
}

func TestMacOsSystemVersion(t *testing.T) {

	systemVersion := `
	<?xml version="1.0" encoding="UTF-8"?>
	<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
	<plist version="1.0">
		<dict>
			<key>ProductBuildVersion</key>
			<string>17C88</string>
			<key>ProductCopyright</key>
			<string>1983-2017 Apple Inc.</string>
			<key>ProductName</key>
			<string>Mac OS X</string>
			<key>ProductUserVisibleVersion</key>
			<string>10.13.2</string>
			<key>ProductVersion</key>
			<string>10.13.2</string>
		</dict>
	</plist>
	`

	m, err := ParseMacOSSystemVersion(systemVersion)
	assert.Nil(t, err)

	assert.Equal(t, "17C88", m["ProductBuildVersion"], "ProductBuildVersion should be parsed properly")
	assert.Equal(t, "1983-2017 Apple Inc.", m["ProductCopyright"], "ProductCopyright should be parsed properly")
	assert.Equal(t, "Mac OS X", m["ProductName"], "ProductName should be parsed properly")
	assert.Equal(t, "10.13.2", m["ProductUserVisibleVersion"], "ProductUserVisibleVersion should be parsed properly")
	assert.Equal(t, "10.13.2", m["ProductVersion"], "ProductVersion should be parsed properly")
}

func TestWindowsWmic(t *testing.T) {

	osVersion :=
		`Node,BootDevice,BuildNumber,BuildType,Caption,CodeSet,CountryCode,CreationClassName,CSCreationClassName,CSDVersion,CSName,CurrentTimeZone,DataExecutionPrevention_32BitApplications,DataExecutionPrevention_Available,DataExecutionPrevention_Drivers,DataExecutionPrevention_SupportPolicy,Debug,Description,Distributed,EncryptionLevel,ForegroundApplicationBoost,FreePhysicalMemory,FreeSpaceInPagingFiles,FreeVirtualMemory,InstallDate,LargeSystemCache,LastBootUpTime,LocalDateTime,Locale,Manufacturer,MaxNumberOfProcesses,MaxProcessMemorySize,MUILanguages,Name,NumberOfLicensedUsers,NumberOfProcesses,NumberOfUsers,OperatingSystemSKU,Organization,OSArchitecture,OSLanguage,OSProductSuite,OSType,OtherTypeDescription,PAEEnabled,PlusProductID,PlusVersionNumber,PortableOperatingSystem,Primary,ProductType,RegisteredUser,SerialNumber,ServicePackMajorVersion,ServicePackMinorVersion,SizeStoredInPagingFiles,Status,SuiteMask,SystemDevice,SystemDirectory,SystemDrive,TotalSwapSpaceSize,TotalVirtualMemorySize,TotalVisibleMemorySize,Version,WindowsDirectory
VAGRANT-2016,\\Device\\HarddiskVolume1,14393,Multiprocessor Free,Microsoft Windows Server 2016 Standard Evaluation,1252,1,Win32_OperatingSystem,Win32_ComputerSystem,,VAGRANT-2016,-420,TRUE,TRUE,TRUE,3,FALSE,,FALSE,256,2,1629000,1179648,2833804,20180313201557.000000-420,,20180630024418.280385-420,20180630024734.124000-420,0409,Microsoft Corporation,4294967295,137438953344,{en-US},Microsoft Windows Server 2016 Standard Evaluation|C:\\Windows|\\Device\\Harddisk0\\Partition2,0,35,1,79,Vagrant,64-bit,1033,272,18,,,,,FALSE,TRUE,3,,00378-00000-00000-AA739,0,0,1179648,OK,272,\\Device\\HarddiskVolume2,C:\\Windows\\system32,C:,,3276340,2096692,10.0.14393,C:\\Windows
`

	m, err := ParseWinWmicOS(strings.NewReader(osVersion))
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
