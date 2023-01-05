package azure

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/cloud"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
	security "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/security/armsecurity"
	"go.mondoo.com/cnquery/resources/packs/core"
)

func (a *mqlAzureCloudDefender) id() (string, error) {
	return "azure.cloudDefender", nil
}

func (a *mqlAzureCloudDefender) GetMonitoringAgentAutoProvision() (interface{}, error) {
	at, err := azureTransport(a.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	token, err := at.GetTokenCredential()
	if err != nil {
		return nil, err
	}
	client, err := security.NewAutoProvisioningSettingsClient(at.SubscriptionID(), token, &arm.ClientOptions{})
	if err != nil {
		return nil, err
	}

	setting, err := client.Get(ctx, "default", &security.AutoProvisioningSettingsClientGetOptions{})
	if err != nil {
		return nil, err
	}
	autoProvision := *setting.Properties.AutoProvision
	return autoProvision == security.AutoProvisionOn, nil
}

func (a *mqlAzureCloudDefender) GetSecurityContacts() (interface{}, error) {
	at, err := azureTransport(a.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	token, err := at.GetTokenCredential()
	if err != nil {
		return nil, err
	}
	rawToken, err := token.GetToken(ctx, policy.TokenRequestOptions{
		Scopes: []string{"https://management.core.windows.net//.default"},
	})
	if err != nil {
		return nil, err
	}
	ep := cloud.AzurePublic.Services[cloud.ResourceManager].Endpoint
	list, err := getSecurityContacts(ctx, at.SubscriptionID(), ep, rawToken.Token)
	if err != nil {
		return nil, err
	}
	res := []interface{}{}
	for _, contact := range list {
		alertNotifications, err := core.JsonToDict(contact.Properties.AlertNotifications)
		if err != nil {
			return nil, err
		}
		notificationsByRole, err := core.JsonToDict(contact.Properties.NotificationsByRole)
		if err != nil {
			return nil, err
		}
		var mails string
		if contact.Properties.Emails != nil {
			mails = *contact.Properties.Emails
		}
		mqlSecurityContact, err := a.MotorRuntime.CreateResource("azure.cloudDefender.securityContact",
			"id", core.ToString(contact.ID),
			"name", core.ToString(contact.Name),
			"emails", core.StrSliceToInterface(strings.Split(mails, ";")),
			"notificationsByRole", notificationsByRole,
			"alertNotifications", alertNotifications,
		)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlSecurityContact)
	}
	return res, nil
}

func (a *mqlAzureCloudDefenderSecurityContact) id() (string, error) {
	return a.Id()
}

// the armsecurity.NewListPager is broken, see https://github.com/Azure/azure-sdk-for-go/issues/19740.
// until it's fixed, we can fetch them manually
func getSecurityContacts(ctx context.Context, subscriptionId, host, token string) ([]security.Contact, error) {
	urlPath := "/subscriptions/{subscriptionId}/providers/Microsoft.Security/securityContacts"
	urlPath = strings.ReplaceAll(urlPath, "{subscriptionId}", url.PathEscape(subscriptionId))
	urlPath = runtime.JoinPaths(host, urlPath)
	client := http.Client{}
	req, err := http.NewRequest("GET", urlPath, nil)
	if err != nil {
		return []security.Contact{}, err
	}
	q := req.URL.Query()
	q.Set("api-version", "2020-01-01-preview")
	req.URL.RawQuery = q.Encode()
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	resp, err := client.Do(req)
	if err != nil {
		return []security.Contact{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return []security.Contact{}, errors.New("failed to fetch security contacts from " + urlPath + ": " + resp.Status)
	}

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return []security.Contact{}, err
	}
	result := []security.Contact{}
	err = json.Unmarshal(raw, &result)
	return result, err
}
