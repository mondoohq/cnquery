package azure

import (
	"encoding/json"
	"io"
	"io/ioutil"

	"github.com/cockroachdb/errors"
	"go.mondoo.com/cnquery/motor/providers/local"
)

func IsAzInstalled() bool {
	t, err := local.New()
	if err != nil {
		return false
	}

	command, err := t.RunCommand("az")
	return command.ExitStatus == 0 && err == nil
}

// shells out to `az account show --output json` to determine the default account
// call `az account list` displays all subscriptions
func GetAccount() (*AzureAccount, error) {
	t, err := local.New()
	if err != nil {
		return nil, err
	}

	cmd, err := t.RunCommand("az account show --output json")
	if err != nil {
		return nil, errors.Wrap(err, "could not read az account show")
	}

	return ParseAzureAccount(cmd.Stdout)
}

func ParseAzureAccount(r io.Reader) (*AzureAccount, error) {
	var azureAccount AzureAccount

	data, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(data, &azureAccount)
	if err != nil {
		return nil, err
	}
	return &azureAccount, nil
}

type AzureAccount struct {
	EnvironmentName string `json:"environmentName"`
	HomeTenantID    string `json:"homeTenantId"`
	ID              string `json:"id"`
	TenantId        string `json:"tenantId"`
	State           string `json:"state"`
	Name            string `json:"name"`
	IsDefault       bool   `json:"isDefault"`
}
