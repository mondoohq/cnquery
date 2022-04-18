package resources

import (
	"go.mondoo.io/mondoo/lumi"
	"go.mondoo.io/mondoo/lumi/resources/windows"
)

func (w *lumiWindowsSecurityHealth) id() (string, error) {
	return "windows.security.health", nil
}

func (p *lumiWindowsSecurityHealth) init(args *lumi.Args) (*lumi.Args, WindowsSecurityHealth, error) {
	if len(*args) > 2 {
		return args, nil, nil
	}

	health, err := windows.GetSecurityProviderHealth(p.Runtime.Motor.Transport)
	if err != nil {
		return nil, nil, err
	}

	filewall, _ := jsonToDict(health.Firewall)
	autoupdate, _ := jsonToDict(health.AutoUpdate)
	antivirus, _ := jsonToDict(health.AntiVirus)
	antispyware, _ := jsonToDict(health.AntiSpyware)
	internetsettings, _ := jsonToDict(health.InternetSettings)
	uac, _ := jsonToDict(health.Uac)
	securitycenterservice, _ := jsonToDict(health.SecurityCenterService)

	(*args)["firewall"] = filewall
	(*args)["autoUpdate"] = autoupdate
	(*args)["antiVirus"] = antivirus
	(*args)["antiSpyware"] = antispyware
	(*args)["internetSettings"] = internetsettings
	(*args)["uac"] = uac
	(*args)["securityCenterService"] = securitycenterservice

	return args, nil, nil
}
