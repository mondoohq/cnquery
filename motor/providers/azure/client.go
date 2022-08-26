package azure

import (
	"context"
	"encoding/json"
	"io"
	"io/ioutil"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/resources/mgmt/subscriptions"
	"github.com/cockroachdb/errors"
	"go.mondoo.com/cnquery/motor/providers/local"
)

func IsAzInstalled() error {
	t, err := local.New()
	if err != nil {
		return err
	}

	_, err = t.RunCommand("az")
	if err != nil {
		return errors.New("could not find az command")
	}
	return nil
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

func VerifySubscription(subscriptionId string) (subscriptions.Subscription, error) {
	authorizer, err := GetAuthorizer()
	if err != nil {
		return subscriptions.Subscription{}, err
	}

	subscriptionsC := subscriptions.NewClient()
	subscriptionsC.Authorizer = authorizer

	ctx := context.Background()
	return subscriptionsC.Get(ctx, subscriptionId)
}
