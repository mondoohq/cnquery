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

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/cloud"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
	security "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/security/armsecurity"
	"go.mondoo.com/cnquery/v11/providers/azure/connection"
)

type armSecurityConn struct {
	subscriptionId string
	host           string
	token          azcore.TokenCredential
}

func (a armSecurityConn) GetToken() (azcore.AccessToken, error) {
	return a.token.GetToken(context.Background(), policy.TokenRequestOptions{
		Scopes: []string{"https://management.core.windows.net//.default"},
	})
}

func getArmSecurityConnection(ctx context.Context, conn *connection.AzureConnection, subId string) (armSecurityConn, error) {
	token := conn.Token()

	ep := cloud.AzurePublic.Services[cloud.ResourceManager].Endpoint
	return armSecurityConn{subId, ep, token}, nil
}

func getPolicyAssignments(ctx context.Context, conn armSecurityConn) (PolicyAssignments, error) {
	token, err := conn.GetToken()
	if err != nil {
		return PolicyAssignments{}, err
	}
	urlPath := "/subscriptions/{subscriptionId}/providers/Microsoft.Authorization/policyAssignments"
	urlPath = strings.ReplaceAll(urlPath, "{subscriptionId}", url.PathEscape(conn.subscriptionId))
	urlPath = runtime.JoinPaths(conn.host, urlPath)
	client := http.Client{}
	req, err := http.NewRequest("GET", urlPath, nil)
	if err != nil {
		return PolicyAssignments{}, err
	}
	q := req.URL.Query()
	q.Set("api-version", "2022-06-01")
	req.URL.RawQuery = q.Encode()
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token.Token))

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
func getSecurityContacts(ctx context.Context, conn armSecurityConn) ([]security.Contact, error) {
	token, err := conn.GetToken()
	if err != nil {
		return []security.Contact{}, err
	}
	urlPath := "/subscriptions/{subscriptionId}/providers/Microsoft.Security/securityContacts"
	urlPath = strings.ReplaceAll(urlPath, "{subscriptionId}", url.PathEscape(conn.subscriptionId))
	urlPath = runtime.JoinPaths(conn.host, urlPath)
	client := http.Client{}
	req, err := http.NewRequest("GET", urlPath, nil)
	if err != nil {
		return []security.Contact{}, err
	}
	q := req.URL.Query()
	q.Set("api-version", "2020-01-01-preview")
	req.URL.RawQuery = q.Encode()
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token.Token))

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

func getServerVulnAssessmentSettings(ctx context.Context, conn armSecurityConn) (ServerVulnerabilityAssessmentsSettingsList, error) {
	token, err := conn.GetToken()
	if err != nil {
		return ServerVulnerabilityAssessmentsSettingsList{}, err
	}
	urlPath := "/subscriptions/{subscriptionId}/providers/Microsoft.Security/serverVulnerabilityAssessmentsSettings"
	urlPath = strings.ReplaceAll(urlPath, "{subscriptionId}", url.PathEscape(conn.subscriptionId))
	urlPath = runtime.JoinPaths(conn.host, urlPath)
	client := http.Client{}
	req, err := http.NewRequest("GET", urlPath, nil)
	if err != nil {
		return ServerVulnerabilityAssessmentsSettingsList{}, err
	}
	q := req.URL.Query()
	q.Set("api-version", "2022-01-01-preview")
	req.URL.RawQuery = q.Encode()
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token.Token))

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

// https://learn.microsoft.com/en-us/azure/templates/microsoft.authorization/policyassignments?pivots=deployment-language-bicep#property-values
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
		DisplayName     string `json:"displayName"`
		Description     string `json:"description"`
		AssignmentType  string `json:"assignmentType"`
		EnforcementMode string `json:"enforcementMode"`
		Metadata        struct {
			Category string `json:"category"`
		} `json:"metadata"`
		PolicyDefinitionID string `json:"policyDefinitionId"`
		Parameters         struct {
			AllowedSkus struct {
				Value string `json:"value"`
			} `json:"allowedSkus"`
			Effect struct {
				Value string `json:"value"`
			} `json:"effect"`
			ApprovedExtensions struct {
				Value []string `json:"value"`
			} `json:"approvedExtensions"`
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
