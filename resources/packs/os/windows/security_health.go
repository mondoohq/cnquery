package windows

import (
	"encoding/json"
	"io"
	"io/ioutil"

	"go.mondoo.io/mondoo/motor/providers/os"
	"go.mondoo.io/mondoo/motor/providers/os/powershell"
)

// This implements the Windows Security Center API
// Powershell does not offer a native method to gather this information, therefore we need
// to write a C# script that is encapsulated in powershell
//
// https://docs.microsoft.com/en-us/windows/win32/api/Wscapi/ne-wscapi-wsc_security_provider
// https://github.com/microsoft/Windows-classic-samples/tree/main/Samples/WebSecurityCenter

// https://docs.microsoft.com/en-us/windows/win32/api/wscapi/ne-wscapi-wsc_security_provider_health
var securityHealthStatusValues = map[int64]string{
	0: "GOOD",
	1: "NOT_MONITORED",
	2: "POOR",
	3: "SNOOZE",
}

// The available security provider are documented in
// https://docs.microsoft.com/en-us/windows/win32/api/wscapi/ne-wscapi-wsc_security_provider
const windowsSecurityHealthScript = `
$MethodDefinition = @"
[DllImport("wscapi.dll",CharSet = CharSet.Unicode, SetLastError = true)]
private static extern int WscGetSecurityProviderHealth(int inValue, ref int outValue);

public static int GetSecurityProviderHealth(int inValue)
{
  int outValue = -1;
  int result = WscGetSecurityProviderHealth(inValue, ref outValue);
  return outValue;
}
"@
 
$mondoo_wscapi = Add-Type -MemberDefinition $MethodDefinition -Name ‘mondoo_wscapi’ -Namespace ‘Win32’ -PassThru

$WSC_SECURITY_PROVIDER_FIREWALL = 1
$WSC_SECURITY_PROVIDER_AUTOUPDATE_SETTINGS = 2
$WSC_SECURITY_PROVIDER_ANTIVIRUS = 4
$WSC_SECURITY_PROVIDER_ANTISPYWARE = 8
$WSC_SECURITY_PROVIDER_INTERNET_SETTINGS = 16
$WSC_SECURITY_PROVIDER_USER_ACCOUNT_CONTROL = 32
$WSC_SECURITY_PROVIDER_SERVICE = 64

$securityProviderHealth = New-Object PSObject
Add-Member -InputObject $securityProviderHealth -MemberType NoteProperty -Name firewall -Value $mondoo_wscapi::GetSecurityProviderHealth($WSC_SECURITY_PROVIDER_FIREWALL)
Add-Member -InputObject $securityProviderHealth -MemberType NoteProperty -Name autoUpdate -Value $mondoo_wscapi::GetSecurityProviderHealth($WSC_SECURITY_PROVIDER_AUTOUPDATE_SETTINGS)
Add-Member -InputObject $securityProviderHealth -MemberType NoteProperty -Name antiVirus -Value $mondoo_wscapi::GetSecurityProviderHealth($WSC_SECURITY_PROVIDER_ANTIVIRUS)
Add-Member -InputObject $securityProviderHealth -MemberType NoteProperty -Name antiSpyware -Value $mondoo_wscapi::GetSecurityProviderHealth($WSC_SECURITY_PROVIDER_ANTISPYWARE)
Add-Member -InputObject $securityProviderHealth -MemberType NoteProperty -Name internetSettings -Value $mondoo_wscapi::GetSecurityProviderHealth($WSC_SECURITY_PROVIDER_INTERNET_SETTINGS)
Add-Member -InputObject $securityProviderHealth -MemberType NoteProperty -Name uac -Value $mondoo_wscapi::GetSecurityProviderHealth($WSC_SECURITY_PROVIDER_USER_ACCOUNT_CONTROL)
Add-Member -InputObject $securityProviderHealth -MemberType NoteProperty -Name securityCenterService -Value $mondoo_wscapi::GetSecurityProviderHealth($WSC_SECURITY_PROVIDER_SERVICE)

ConvertTo-Json -Depth 3 -Compress $securityProviderHealth
`

type powershellSecurityHealthStatus struct {
	Firewall              int64
	AutoUpdate            int64
	AntiVirus             int64
	AntiSpyware           int64
	InternetSettings      int64
	Uac                   int64
	SecurityCenterService int64
}

type windowsSecurityHealth struct {
	Firewall              statusCode
	AutoUpdate            statusCode
	AntiVirus             statusCode
	AntiSpyware           statusCode
	InternetSettings      statusCode
	Uac                   statusCode
	SecurityCenterService statusCode
}

func GetSecurityProviderHealth(p os.OperatingSystemProvider) (*windowsSecurityHealth, error) {
	c, err := p.RunCommand(powershell.Encode(windowsSecurityHealthScript))
	if err != nil {
		return nil, err
	}

	return ParseSecurityProviderHealth(c.Stdout)
}

func ParseSecurityProviderHealth(r io.Reader) (*windowsSecurityHealth, error) {
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	var status powershellSecurityHealthStatus
	err = json.Unmarshal(data, &status)
	if err != nil {
		return nil, err
	}

	return &windowsSecurityHealth{
		Firewall: statusCode{
			Code: status.Firewall,
			Text: securityHealthStatusValues[status.Firewall],
		},
		AutoUpdate: statusCode{
			Code: status.AutoUpdate,
			Text: securityHealthStatusValues[status.AutoUpdate],
		},
		Uac: statusCode{
			Code: status.Uac,
			Text: securityHealthStatusValues[status.Uac],
		},
		AntiSpyware: statusCode{
			Code: status.AntiSpyware,
			Text: securityHealthStatusValues[status.AntiSpyware],
		},
		AntiVirus: statusCode{
			Code: status.AntiVirus,
			Text: securityHealthStatusValues[status.AntiVirus],
		},
		InternetSettings: statusCode{
			Code: status.InternetSettings,
			Text: securityHealthStatusValues[status.InternetSettings],
		},
		SecurityCenterService: statusCode{
			Code: status.SecurityCenterService,
			Text: securityHealthStatusValues[status.SecurityCenterService],
		},
	}, nil
}
