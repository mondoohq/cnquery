package azure

import (
	"context"
	"encoding/json"
	"io"
	"io/ioutil"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/resources/mgmt/subscriptions"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/pkg/errors"
	"go.mondoo.io/mondoo/motor/transports"
	"go.mondoo.io/mondoo/motor/transports/local"
)

func isAzInstalled(t transports.Transport) (bool, error) {
	_, err := t.RunCommand("az")
	if err != nil {
		return false, errors.Wrap(err, "could not find az command")
	}
	return true, nil
}

// shells out to `az account show --output json` to determine the default account
// call `az account list` displays all subscriptions
func GetAccount() (*AzureAccount, error) {
	t, err := local.New()
	if err != nil {
		return nil, err
	}

	ok, err := isAzInstalled(t)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errors.New("az command not installed")
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
	// TODO: todo, support service accounts
	authorizer, err := auth.NewAuthorizerFromCLI()
	if err != nil {
		return subscriptions.Subscription{}, err
	}
	subscriptionsC := subscriptions.NewClient()
	subscriptionsC.Authorizer = authorizer

	ctx := context.Background()
	return subscriptionsC.Get(ctx, subscriptionId)
}
