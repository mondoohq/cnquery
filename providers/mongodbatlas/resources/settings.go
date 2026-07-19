// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"net/http"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
)

// isAccessDenied reports whether the API response indicates the credential is
// not authorized to read the resource (a feature-gated or elevated-privilege
// endpoint). Such resources degrade to null rather than failing the scan.
func isAccessDenied(resp *http.Response) bool {
	return resp != nil && (resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden)
}

func (r *mqlMongodbatlas) projectSettings() (*mqlMongodbatlasProjectConfig, error) {
	pid, err := projectID(r.MqlRuntime)
	if err != nil {
		return nil, err
	}
	s, _, err := atlasClient(r.MqlRuntime).ProjectsApi.GetProjectSettings(context.Background(), pid).Execute()
	if err != nil {
		return nil, err
	}
	res, err := CreateResource(r.MqlRuntime, "mongodbatlas.projectConfig", map[string]*llx.RawData{
		"__id":                                        llx.StringData("mongodbatlas.projectConfig/" + pid),
		"isDataExplorerEnabled":                       llx.BoolData(s.GetIsDataExplorerEnabled()),
		"isDataExplorerGenAIFeaturesEnabled":          llx.BoolData(s.GetIsDataExplorerGenAIFeaturesEnabled()),
		"isExtendedStorageSizesEnabled":               llx.BoolData(s.GetIsExtendedStorageSizesEnabled()),
		"isPerformanceAdvisorEnabled":                 llx.BoolData(s.GetIsPerformanceAdvisorEnabled()),
		"isRealtimePerformancePanelEnabled":           llx.BoolData(s.GetIsRealtimePerformancePanelEnabled()),
		"isSchemaAdvisorEnabled":                      llx.BoolData(s.GetIsSchemaAdvisorEnabled()),
		"isCollectDatabaseSpecificsStatisticsEnabled": llx.BoolData(s.GetIsCollectDatabaseSpecificsStatisticsEnabled()),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlMongodbatlasProjectConfig), nil
}

func (r *mqlMongodbatlas) auditing() (*mqlMongodbatlasAuditConfig, error) {
	pid, err := projectID(r.MqlRuntime)
	if err != nil {
		return nil, err
	}
	a, httpResp, err := atlasClient(r.MqlRuntime).AuditingApi.GetAuditingConfiguration(context.Background(), pid).Execute()
	if err != nil {
		if isAccessDenied(httpResp) || (httpResp != nil && httpResp.StatusCode == http.StatusNotFound) {
			r.Auditing.State = plugin.StateIsSet | plugin.StateIsNull
			return nil, nil
		}
		return nil, err
	}
	res, err := CreateResource(r.MqlRuntime, "mongodbatlas.auditConfig", map[string]*llx.RawData{
		"__id":                      llx.StringData("mongodbatlas.auditConfig/" + pid),
		"enabled":                   llx.BoolData(a.GetEnabled()),
		"auditAuthorizationSuccess": llx.BoolData(a.GetAuditAuthorizationSuccess()),
		"auditFilter":               llx.StringData(a.GetAuditFilter()),
		"configurationType":         llx.StringData(a.GetConfigurationType()),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlMongodbatlasAuditConfig), nil
}

func (r *mqlMongodbatlas) encryptionAtRest() (*mqlMongodbatlasEncryptionConfig, error) {
	pid, err := projectID(r.MqlRuntime)
	if err != nil {
		return nil, err
	}
	e, httpResp, err := atlasClient(r.MqlRuntime).EncryptionAtRestUsingCustomerKeyManagementApi.GetEncryptionAtRest(context.Background(), pid).Execute()
	if err != nil {
		if isAccessDenied(httpResp) || (httpResp != nil && httpResp.StatusCode == http.StatusNotFound) {
			r.EncryptionAtRest.State = plugin.StateIsSet | plugin.StateIsNull
			return nil, nil
		}
		return nil, err
	}
	aws := e.GetAwsKms()
	azure := e.GetAzureKeyVault()
	gcp := e.GetGoogleCloudKms()
	res, err := CreateResource(r.MqlRuntime, "mongodbatlas.encryptionConfig", map[string]*llx.RawData{
		"__id":                  llx.StringData("mongodbatlas.encryptionConfig/" + pid),
		"awsKmsEnabled":         llx.BoolData(aws.GetEnabled()),
		"awsKmsValid":           llx.BoolData(aws.GetValid()),
		"azureKeyVaultEnabled":  llx.BoolData(azure.GetEnabled()),
		"azureKeyVaultValid":    llx.BoolData(azure.GetValid()),
		"googleCloudKmsEnabled": llx.BoolData(gcp.GetEnabled()),
		"googleCloudKmsValid":   llx.BoolData(gcp.GetValid()),
		"enabledForSearchNodes": llx.BoolData(e.GetEnabledForSearchNodes()),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlMongodbatlasEncryptionConfig), nil
}
