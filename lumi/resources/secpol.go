package resources

import "errors"

func (p *lumiSecpol) id() (string, error) {
	return "secpol", nil
}

func (p *lumiSecpol) GetSystemaccess() ([]interface{}, error) {
	return nil, errors.New("not implemented")
}

func (p *lumiSecpol) GetEventaudit() ([]interface{}, error) {
	return nil, errors.New("not implemented")
}

func (p *lumiSecpol) GetRegistryvalues() ([]interface{}, error) {
	return nil, errors.New("not implemented")
}

func (p *lumiSecpol) GetPrivilegerights() ([]interface{}, error) {
	return nil, errors.New("not implemented")
}
