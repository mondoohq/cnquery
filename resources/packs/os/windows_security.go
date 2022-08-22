package os

import (
	"go.mondoo.io/mondoo/resources"
	"go.mondoo.io/mondoo/resources/packs/core"
	"go.mondoo.io/mondoo/resources/packs/os/windows"
)

func (w *mqlWindowsSecurity) id() (string, error) {
	return "windows.security", nil
}

func (w *mqlWindowsSecurity) GetProducts() ([]interface{}, error) {
	osProvider, err := osProvider(w.MotorRuntime.Motor)
	if err != nil {
		return nil, err
	}

	products, err := windows.GetSecurityProducts(osProvider)
	if err != nil {
		return nil, err
	}

	res := []interface{}{}
	for i := range products {
		p := products[i]

		mqlProduct, err := w.MotorRuntime.CreateResource("windows.security.product",
			"type", p.Type,
			"guid", p.Guid,
			"name", p.Name,
			"state", p.State,
			"productState", p.ProductStatus,
			"signatureState", p.SignatureStatus,
			"timestamp", &p.Timestamp,
		)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlProduct)
	}

	return res, nil
}

func (w *mqlWindowsSecurityProduct) id() (string, error) {
	guid, _ := w.Guid()
	return "windows.security.product/" + guid, nil
}

func (w *mqlWindowsSecurityHealth) id() (string, error) {
	return "windows.security.health", nil
}

func (p *mqlWindowsSecurityHealth) init(args *resources.Args) (*resources.Args, WindowsSecurityHealth, error) {
	if len(*args) > 2 {
		return args, nil, nil
	}

	osProvider, err := osProvider(p.MotorRuntime.Motor)
	if err != nil {
		return nil, nil, err
	}

	health, err := windows.GetSecurityProviderHealth(osProvider)
	if err != nil {
		return nil, nil, err
	}

	filewall, _ := core.JsonToDict(health.Firewall)
	autoupdate, _ := core.JsonToDict(health.AutoUpdate)
	antivirus, _ := core.JsonToDict(health.AntiVirus)
	antispyware, _ := core.JsonToDict(health.AntiSpyware)
	internetsettings, _ := core.JsonToDict(health.InternetSettings)
	uac, _ := core.JsonToDict(health.Uac)
	securitycenterservice, _ := core.JsonToDict(health.SecurityCenterService)

	(*args)["firewall"] = filewall
	(*args)["autoUpdate"] = autoupdate
	(*args)["antiVirus"] = antivirus
	(*args)["antiSpyware"] = antispyware
	(*args)["internetSettings"] = internetsettings
	(*args)["uac"] = uac
	(*args)["securityCenterService"] = securitycenterservice

	return args, nil, nil
}
