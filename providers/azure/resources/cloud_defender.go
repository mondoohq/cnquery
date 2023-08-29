// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/providers/azure/connection"
	"go.mondoo.com/cnquery/types"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/cloud"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
	security "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/security/armsecurity"
)

const (
	vaQualysPolicyDefinitionId string = "/providers/Microsoft.Authorization/policyDefinitions/13ce0167-8ca6-4048-8e6b-f996402e3c1b"

	// There are two policy per component: one for ARC clusters and one for k8s clusters
	arcClusterDefenderExtensionDefinitionId        string = "/providers/Microsoft.Authorization/policyDefinitions/708b60a6-d253-4fe0-9114-4be4c00f012c"
	kubernetesClusterDefenderExtensionDefinitionId string = "/providers/Microsoft.Authorization/policyDefinitions/64def556-fbad-4622-930e-72d1d5589bf5"

	arcClusterPolicyExtensionDefinitionId       string = "/providers/Microsoft.Authorization/policyDefinitions/0adc5395-9169-4b9b-8687-af838d69410a"
	kubernetesClusterPolicyExtensonDefinitionId string = "/providers/Microsoft.Authorization/policyDefinitions/0adc5395-9169-4b9b-8687-af838d69410a"
)

func (a *mqlAzureSubscriptionCloudDefender) id() (string, error) {
	return "azure.subscription.cloudDefender/" + a.SubscriptionId.Data, nil
}

func initAzureSubscriptionCloudDefender(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 0 {
		return args, nil, nil
	}

	conn := runtime.Connection.(*connection.AzureConnection)
	args["subscriptionId"] = llx.StringData(conn.SubId())

	return args, nil, nil
}

func (a *mqlAzureSubscriptionCloudDefenderSecurityContact) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionCloudDefender) defenderForServers() (interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	subId := a.SubscriptionId.Data
	rawToken, err := token.GetToken(ctx, policy.TokenRequestOptions{
		Scopes: []string{"https://management.core.windows.net//.default"},
	})
	if err != nil {
		return nil, err
	}
	ep := cloud.AzurePublic.Services[cloud.ResourceManager].Endpoint
	list, err := getPolicyAssignments(ctx, subId, ep, rawToken.Token)
	if err != nil {
		return nil, err
	}
	serverVASetings, err := getServerVulnAssessmentSettings(ctx, subId, ep, rawToken.Token)
	if err != nil {
		return nil, err
	}

	type defenderForServers struct {
		Enabled                         bool   `json:"enabled"`
		VulnerabilityManagementToolName string `json:"vulnerabilityManagementToolName"`
	}

	resp := defenderForServers{}
	for _, it := range list.PolicyAssignments {
		if it.Properties.PolicyDefinitionID == vaQualysPolicyDefinitionId {
			resp.Enabled = true
			resp.VulnerabilityManagementToolName = "Microsoft Defender for Cloud integrated Qualys scanner"
		}
	}
	for _, sett := range serverVASetings.Settings {
		if sett.Properties.SelectedProvider == "MdeTvm" && sett.Name == "AzureServersSetting" {
			resp.Enabled = true
			resp.VulnerabilityManagementToolName = "Microsoft Defender vulnerability management"

		}
	}
	return convert.JsonToDict(resp)
}

func (a *mqlAzureSubscriptionCloudDefender) monitoringAgentAutoProvision() (bool, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	subId := a.SubscriptionId.Data

	client, err := security.NewAutoProvisioningSettingsClient(subId, token, &arm.ClientOptions{})
	if err != nil {
		return false, err
	}

	setting, err := client.Get(ctx, "default", &security.AutoProvisioningSettingsClientGetOptions{})
	if err != nil {
		return false, err
	}
	autoProvision := *setting.Properties.AutoProvision
	return autoProvision == security.AutoProvisionOn, nil
}

func (a *mqlAzureSubscriptionCloudDefender) defenderForContainers() (interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	subId := a.SubscriptionId.Data

	rawToken, err := token.GetToken(ctx, policy.TokenRequestOptions{
		Scopes: []string{"https://management.core.windows.net//.default"},
	})
	if err != nil {
		return nil, err
	}
	ep := cloud.AzurePublic.Services[cloud.ResourceManager].Endpoint
	pas, err := getPolicyAssignments(ctx, subId, ep, rawToken.Token)
	if err != nil {
		return nil, err
	}

	type defenderForContainers struct {
		DefenderDaemonSet        bool `json:"defenderDaemonSet"`
		AzurePolicyForKubernetes bool `json:"azurePolicyForKubernetes"`
	}

	kubernetesDefender := false
	arcDefender := false
	kubernetesPolicyExt := false
	arcPoilcyExt := false
	for _, it := range pas.PolicyAssignments {
		if it.Properties.PolicyDefinitionID == arcClusterDefenderExtensionDefinitionId &&
			it.Properties.Scope == fmt.Sprintf("/subscriptions/%s", subId) {
			arcDefender = true
		}
		if it.Properties.PolicyDefinitionID == kubernetesClusterDefenderExtensionDefinitionId &&
			it.Properties.Scope == fmt.Sprintf("/subscriptions/%s", subId) {
			kubernetesDefender = true
		}
		if it.Properties.PolicyDefinitionID == arcClusterPolicyExtensionDefinitionId &&
			it.Properties.Scope == fmt.Sprintf("/subscriptions/%s", subId) {
			arcPoilcyExt = true
		}
		if it.Properties.PolicyDefinitionID == kubernetesClusterPolicyExtensonDefinitionId &&
			it.Properties.Scope == fmt.Sprintf("/subscriptions/%s", subId) {
			kubernetesPolicyExt = true
		}
	}

	def := defenderForContainers{
		DefenderDaemonSet:        arcDefender && kubernetesDefender,
		AzurePolicyForKubernetes: arcPoilcyExt && kubernetesPolicyExt,
	}
	return convert.JsonToDict(def)
}

func (a *mqlAzureSubscriptionCloudDefender) securityContacts() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	subId := a.SubscriptionId.Data
	rawToken, err := token.GetToken(ctx, policy.TokenRequestOptions{
		Scopes: []string{"https://management.core.windows.net//.default"},
	})
	ep := cloud.AzurePublic.Services[cloud.ResourceManager].Endpoint
	list, err := getSecurityContacts(ctx, subId, ep, rawToken.Token)
	if err != nil {
		return nil, err
	}
	res := []interface{}{}
	for _, contact := range list {
		alertNotifications, err := convert.JsonToDict(contact.Properties.AlertNotifications)
		if err != nil {
			return nil, err
		}
		notificationsByRole, err := convert.JsonToDict(contact.Properties.NotificationsByRole)
		if err != nil {
			return nil, err
		}
		mails := ""
		if contact.Properties.Emails != nil {
			mails = *contact.Properties.Emails
		}
		mailsArr := strings.Split(mails, ";")
		mqlSecurityContact, err := CreateResource(a.MqlRuntime, "azure.subscription.cloudDefender.securityContact",
			map[string]*llx.RawData{
				"id":                  llx.StringData(convert.ToString(contact.ID)),
				"name":                llx.StringData(convert.ToString(contact.Name)),
				"emails":              llx.ArrayData(convert.SliceAnyToInterface(mailsArr), types.String),
				"notificationsByRole": llx.DictData(notificationsByRole),
				"alertNotifications":  llx.DictData(alertNotifications),
			})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlSecurityContact)
	}
	return res, nil
}

func getPolicyAssignments(ctx context.Context, subscriptionId, host, token string) (PolicyAssignments, error) {
	urlPath := "/subscriptions/{subscriptionId}/providers/Microsoft.Authorization/policyAssignments"
	urlPath = strings.ReplaceAll(urlPath, "{subscriptionId}", url.PathEscape(subscriptionId))
	urlPath = runtime.JoinPaths(host, urlPath)
	client := http.Client{}
	req, err := http.NewRequest("GET", urlPath, nil)
	if err != nil {
		return PolicyAssignments{}, err
	}
	q := req.URL.Query()
	q.Set("api-version", "2022-06-01")
	req.URL.RawQuery = q.Encode()
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	resp, err := client.Do(req)
	if err != nil {
		return PolicyAssignments{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return PolicyAssignments{}, errors.New("failed to fetch security contacts from " + urlPath + ": " + resp.Status)
	}

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return PolicyAssignments{}, err
	}
	result := PolicyAssignments{}
	err = json.Unmarshal(raw, &result)
	return result, err
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
	if err != nil {
		// fallback, try to unmarshal to ContactList
		contactList := &security.ContactList{}
		err = json.Unmarshal(raw, contactList)
		if err != nil {
			return nil, err
		}
		for _, c := range contactList.Value {
			if c != nil {
				result = append(result, *c)
			}
		}
	}

	return result, err
}

func getServerVulnAssessmentSettings(ctx context.Context, subscriptionId, host, token string) (ServerVulnerabilityAssessmentsSettingsList, error) {
	urlPath := "/subscriptions/{subscriptionId}/providers/Microsoft.Security/serverVulnerabilityAssessmentsSettings"
	urlPath = strings.ReplaceAll(urlPath, "{subscriptionId}", url.PathEscape(subscriptionId))
	urlPath = runtime.JoinPaths(host, urlPath)
	client := http.Client{}
	req, err := http.NewRequest("GET", urlPath, nil)
	if err != nil {
		return ServerVulnerabilityAssessmentsSettingsList{}, err
	}
	q := req.URL.Query()
	q.Set("api-version", "2022-01-01-preview")
	req.URL.RawQuery = q.Encode()
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	resp, err := client.Do(req)
	if err != nil {
		return ServerVulnerabilityAssessmentsSettingsList{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return ServerVulnerabilityAssessmentsSettingsList{}, errors.New("failed to fetch security contacts from " + urlPath + ": " + resp.Status)
	}

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return ServerVulnerabilityAssessmentsSettingsList{}, err
	}
	result := ServerVulnerabilityAssessmentsSettingsList{}
	err = json.Unmarshal(raw, &result)
	return result, err
}

type PolicyAssignment struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Name     string `json:"name"`
	Location string `json:"location,omitempty"`
	Identity struct {
		Type        string `json:"type"`
		PrincipalID string `json:"principalId"`
		TenantID    string `json:"tenantId"`
	} `json:"identity,omitempty"`
	Properties struct {
		DisplayName string `json:"displayName"`
		Description string `json:"description"`
		Metadata    struct {
			Category string `json:"category"`
		} `json:"metadata"`
		PolicyDefinitionID string `json:"policyDefinitionId"`
		Parameters         struct {
			AllowedSkus struct {
				Value string `json:"value"`
			} `json:"allowedSkus"`
		} `json:"parameters"`
		Scope     string        `json:"scope"`
		NotScopes []interface{} `json:"notScopes"`
	} `json:"properties"`
}
type PolicyAssignments struct {
	PolicyAssignments []PolicyAssignment `json:"value"`
}

type ServerVulnerabilityAssessmentsSettings struct {
	Properties struct {
		SelectedProvider string `json:"selectedProvider"`
	} `json:"properties"`
	SystemData struct {
		CreatedBy          string    `json:"createdBy"`
		CreatedByType      string    `json:"createdByType"`
		CreatedAt          time.Time `json:"createdAt"`
		LastModifiedBy     string    `json:"lastModifiedBy"`
		LastModifiedByType string    `json:"lastModifiedByType"`
		LastModifiedAt     time.Time `json:"lastModifiedAt"`
	} `json:"systemData"`
	Kind string `json:"kind"`
	Name string `json:"name"`
	Type string `json:"type"`
	ID   string `json:"id"`
}

type ServerVulnerabilityAssessmentsSettingsList struct {
	Settings []ServerVulnerabilityAssessmentsSettings `json:"value"`
}
