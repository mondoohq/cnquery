// Copyright Mondoo, Inc. 2024, 2026
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
	"go.mondoo.com/mql/v13/providers/azure/connection"
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
	urlPath := "/subscriptions/{subscriptionId}/providers/Microsoft.Authorization/policyAssignments"
	urlPath = strings.ReplaceAll(urlPath, "{subscriptionId}", url.PathEscape(conn.subscriptionId))
	urlPath = runtime.JoinPaths(conn.host, urlPath)

	// Build the first request URL with the api-version query parameter. The
	// service returns a fully-formed absolute nextLink for subsequent pages, so
	// we only need to assemble the query on the initial request.
	firstURL, err := url.Parse(urlPath)
	if err != nil {
		return PolicyAssignments{}, err
	}
	q := firstURL.Query()
	q.Set("api-version", "2022-06-01")
	firstURL.RawQuery = q.Encode()

	client := http.Client{}
	result := PolicyAssignments{}
	nextURL := firstURL.String()
	for nextURL != "" {
		// Fetch the token per page so a long pagination run over many policy
		// assignments doesn't fail on an expired bearer token; the credential
		// caches and only refreshes when the token is near expiry.
		token, err := conn.GetToken()
		if err != nil {
			return PolicyAssignments{}, err
		}
		req, err := http.NewRequestWithContext(ctx, "GET", nextURL, nil)
		if err != nil {
			return PolicyAssignments{}, err
		}
		req.Header.Set("Accept", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token.Token))

		resp, err := client.Do(req)
		if err != nil {
			return PolicyAssignments{}, err
		}

		if resp.StatusCode != 200 {
			resp.Body.Close()
			return PolicyAssignments{}, errors.New("failed to fetch policy assignments from " + nextURL + ": " + resp.Status)
		}

		raw, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return PolicyAssignments{}, err
		}

		page := PolicyAssignments{}
		if err := json.Unmarshal(raw, &page); err != nil {
			return PolicyAssignments{}, err
		}
		result.PolicyAssignments = append(result.PolicyAssignments, page.PolicyAssignments...)

		if page.NextLink == nil || *page.NextLink == "" {
			break
		}
		nextURL = *page.NextLink
	}
	return result, nil
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
		return ServerVulnerabilityAssessmentsSettingsList{}, errors.New("failed to fetch server vulnerability assessment settings from " + urlPath + ": " + resp.Status)
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
	ID         string `json:"id"`
	Type       string `json:"type"`
	Name       string `json:"name"`
	Location   string `json:"location,omitempty"`
	SystemData any    `json:"systemData,omitempty"`
	Identity   struct {
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
		Scope     string `json:"scope"`
		NotScopes []any  `json:"notScopes"`
	} `json:"properties"`
}

type PolicyAssignments struct {
	PolicyAssignments []PolicyAssignment `json:"value"`
	NextLink          *string            `json:"nextLink"`
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
