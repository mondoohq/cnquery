package resources

import (
	"go.mondoo.io/mondoo/lumi"
	"go.mondoo.io/mondoo/lumi/resources/windows"
)

func (w *lumiWindowsSecurity) id() (string, error) {
	return "windows.security", nil
}

func (w *lumiWindowsSecurity) GetProducts() ([]interface{}, error) {
	products, err := windows.GetSecurityProducts(w.Runtime.Motor.Transport)
	if err != nil {
		return nil, err
	}

	res := []interface{}{}
	for i := range products {
		p := products[i]

		lumiProduct, err := w.Runtime.CreateResource("windows.security.product",
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
		res = append(res, lumiProduct)
	}

	return res, nil
}

func (w *lumiWindowsSecurityProduct) id() (string, error) {
	guid, _ := w.Guid()
	return "windows.security.product/" + guid, nil
}

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
