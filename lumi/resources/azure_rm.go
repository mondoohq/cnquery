package resources

import "errors"

func (a *lumiAzurerm) id() (string, error) {
	return "azurerm", nil
}

func (a *lumiAzurerm) GetVms() ([]interface{}, error) {
	return nil, errors.New("not implemented")
}

func (a *lumiAzurerm) GetSqlServers() ([]interface{}, error) {
	return nil, errors.New("not implemented")
}

func (a *lumiAzurerm) GetPostgresqlServers() ([]interface{}, error) {
	return nil, errors.New("not implemented")
}

func (a *lumiAzurermComputeVm) id() (string, error) {
	return a.Id()
}

func (a *lumiAzurermStorageAccount) id() (string, error) {
	return a.Id()
}
