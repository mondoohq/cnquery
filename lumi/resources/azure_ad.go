package resources

import "errors"

func (a *lumiAzuread) id() (string, error) {
	return "azuread", nil
}

func (a *lumiAzuread) GetUsers() ([]interface{}, error) {
	return nil, errors.New("not implemented")
}

func (a *lumiAzuread) GetGroups() ([]interface{}, error) {
	return nil, errors.New("not implemented")
}

func (a *lumiAzuread) GetDomains() ([]interface{}, error) {
	return nil, errors.New("not implemented")
}

func (a *lumiAzuread) GetApplications() ([]interface{}, error) {
	return nil, errors.New("not implemented")
}

func (a *lumiAzuread) GetServicePrincipals() ([]interface{}, error) {
	return nil, errors.New("not implemented")
}

func (a *lumiAzureadUser) id() (string, error) {
	return a.Id()
}

func (a *lumiAzureadGroup) id() (string, error) {
	return a.Id()
}

func (a *lumiAzureadGroup) GetMembers() ([]interface{}, error) {
	return nil, errors.New("not implemented")
}

func (a *lumiAzureadDomain) id() (string, error) {
	return a.Name()
}

func (a *lumiAzureadApplication) id() (string, error) {
	return a.Id()
}

func (a *lumiAzureadServiceprincipal) id() (string, error) {
	return a.Id()
}

func (a *lumiAzurermStorageBlob) id() (string, error) {
	return a.Id()
}

func (a *lumiAzurermMssqlServer) id() (string, error) {
	return a.Id()
}

func (a *lumiAzurermMssqlDatabase) id() (string, error) {
	return a.Id()
}

func (a *lumiAzurermPostgresqlServer) id() (string, error) {
	return a.Id()
}

func (a *lumiAzurermPostgresqlDatabase) id() (string, error) {
	return a.Id()
}
