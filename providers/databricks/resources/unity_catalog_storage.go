// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"

	"github.com/databricks/databricks-sdk-go/service/catalog"
	"go.mondoo.com/mql/v13/llx"
)

// sseEncryption extracts the server-side encryption algorithm and KMS key ARN
// from a Unity Catalog securable's encryption details. Both are empty when no
// server-side encryption is recorded.
func sseEncryption(ed *catalog.EncryptionDetails) (algorithm string, kmsKeyArn string) {
	if ed == nil || ed.SseEncryptionDetails == nil {
		return "", ""
	}
	return string(ed.SseEncryptionDetails.Algorithm), ed.SseEncryptionDetails.AwsKmsKeyArn
}

func (r *mqlDatabricks) storageCredentials() ([]any, error) {
	ws, err := workspaceClient(r.MqlRuntime)
	if err != nil {
		return nil, err
	}

	creds, err := ws.StorageCredentials.ListAll(context.Background(), catalog.ListStorageCredentialsRequest{})
	if err != nil {
		return nil, err
	}

	out := []any{}
	for i := range creds {
		c := creds[i]

		var awsRoleArn, awsExternalId, azureConnectorId, azureAppId, gcpEmail string
		if c.AwsIamRole != nil {
			awsRoleArn = c.AwsIamRole.RoleArn
			awsExternalId = c.AwsIamRole.ExternalId
		}
		if c.AzureManagedIdentity != nil {
			azureConnectorId = c.AzureManagedIdentity.AccessConnectorId
		}
		if c.AzureServicePrincipal != nil {
			azureAppId = c.AzureServicePrincipal.ApplicationId
		}
		if c.DatabricksGcpServiceAccount != nil {
			gcpEmail = c.DatabricksGcpServiceAccount.Email
		}

		res, err := CreateResource(r.MqlRuntime, "databricks.storageCredential", map[string]*llx.RawData{
			"__id":                               llx.StringData("databricks.storageCredential/" + c.Name),
			"id":                                 llx.StringData(c.Id),
			"name":                               llx.StringData(c.Name),
			"fullName":                           llx.StringData(c.FullName),
			"owner":                              llx.StringData(c.Owner),
			"comment":                            llx.StringData(c.Comment),
			"metastoreId":                        llx.StringData(c.MetastoreId),
			"isolationMode":                      llx.StringData(string(c.IsolationMode)),
			"readOnly":                           llx.BoolData(c.ReadOnly),
			"usedForManagedStorage":              llx.BoolData(c.UsedForManagedStorage),
			"awsIamRoleArn":                      llx.StringData(awsRoleArn),
			"awsIamRoleExternalId":               llx.StringData(awsExternalId),
			"azureAccessConnectorId":             llx.StringData(azureConnectorId),
			"azureServicePrincipalApplicationId": llx.StringData(azureAppId),
			"gcpServiceAccountEmail":             llx.StringData(gcpEmail),
			"createdAt":                          llx.TimeDataPtr(epochMsTime(c.CreatedAt)),
			"createdBy":                          llx.StringData(c.CreatedBy),
			"updatedAt":                          llx.TimeDataPtr(epochMsTime(c.UpdatedAt)),
			"updatedBy":                          llx.StringData(c.UpdatedBy),
		})
		if err != nil {
			return nil, err
		}
		out = append(out, res)
	}
	return out, nil
}

func (r *mqlDatabricksStorageCredential) grants() ([]any, error) {
	ws, err := workspaceClient(r.MqlRuntime)
	if err != nil {
		return nil, err
	}
	return mqlDatabricksGrants(r.MqlRuntime, ws, catalog.SecurableTypeStorageCredential, r.Name.Data)
}

func (r *mqlDatabricks) externalLocations() ([]any, error) {
	ws, err := workspaceClient(r.MqlRuntime)
	if err != nil {
		return nil, err
	}

	locations, err := ws.ExternalLocations.ListAll(context.Background(), catalog.ListExternalLocationsRequest{})
	if err != nil {
		return nil, err
	}

	out := []any{}
	for i := range locations {
		l := locations[i]
		sseAlgorithm, sseKmsKeyArn := sseEncryption(l.EncryptionDetails)

		res, err := CreateResource(r.MqlRuntime, "databricks.externalLocation", map[string]*llx.RawData{
			"__id":                   llx.StringData("databricks.externalLocation/" + l.Name),
			"name":                   llx.StringData(l.Name),
			"url":                    llx.StringData(l.Url),
			"credentialName":         llx.StringData(l.CredentialName),
			"credentialId":           llx.StringData(l.CredentialId),
			"owner":                  llx.StringData(l.Owner),
			"comment":                llx.StringData(l.Comment),
			"metastoreId":            llx.StringData(l.MetastoreId),
			"isolationMode":          llx.StringData(string(l.IsolationMode)),
			"readOnly":               llx.BoolData(l.ReadOnly),
			"fallback":               llx.BoolData(l.Fallback),
			"browseOnly":             llx.BoolData(l.BrowseOnly),
			"sseEncryptionAlgorithm": llx.StringData(sseAlgorithm),
			"sseKmsKeyArn":           llx.StringData(sseKmsKeyArn),
			"createdAt":              llx.TimeDataPtr(epochMsTime(l.CreatedAt)),
			"createdBy":              llx.StringData(l.CreatedBy),
			"updatedAt":              llx.TimeDataPtr(epochMsTime(l.UpdatedAt)),
			"updatedBy":              llx.StringData(l.UpdatedBy),
		})
		if err != nil {
			return nil, err
		}
		out = append(out, res)
	}
	return out, nil
}

func (r *mqlDatabricksExternalLocation) grants() ([]any, error) {
	ws, err := workspaceClient(r.MqlRuntime)
	if err != nil {
		return nil, err
	}
	return mqlDatabricksGrants(r.MqlRuntime, ws, catalog.SecurableTypeExternalLocation, r.Name.Data)
}

func (r *mqlDatabricksSchema) volumes() ([]any, error) {
	ws, err := workspaceClient(r.MqlRuntime)
	if err != nil {
		return nil, err
	}

	volumes, err := ws.Volumes.ListAll(context.Background(), catalog.ListVolumesRequest{
		CatalogName: r.CatalogName.Data,
		SchemaName:  r.Name.Data,
	})
	if err != nil {
		return nil, err
	}

	out := []any{}
	for i := range volumes {
		v := volumes[i]
		sseAlgorithm, sseKmsKeyArn := sseEncryption(v.EncryptionDetails)

		res, err := CreateResource(r.MqlRuntime, "databricks.volume", map[string]*llx.RawData{
			"__id":                   llx.StringData("databricks.volume/" + v.FullName),
			"id":                     llx.StringData(v.VolumeId),
			"name":                   llx.StringData(v.Name),
			"fullName":               llx.StringData(v.FullName),
			"catalogName":            llx.StringData(v.CatalogName),
			"schemaName":             llx.StringData(v.SchemaName),
			"owner":                  llx.StringData(v.Owner),
			"comment":                llx.StringData(v.Comment),
			"metastoreId":            llx.StringData(v.MetastoreId),
			"volumeType":             llx.StringData(string(v.VolumeType)),
			"storageLocation":        llx.StringData(v.StorageLocation),
			"sseEncryptionAlgorithm": llx.StringData(sseAlgorithm),
			"sseKmsKeyArn":           llx.StringData(sseKmsKeyArn),
			"createdAt":              llx.TimeDataPtr(epochMsTime(v.CreatedAt)),
			"createdBy":              llx.StringData(v.CreatedBy),
			"updatedAt":              llx.TimeDataPtr(epochMsTime(v.UpdatedAt)),
			"updatedBy":              llx.StringData(v.UpdatedBy),
		})
		if err != nil {
			return nil, err
		}
		out = append(out, res)
	}
	return out, nil
}

func (r *mqlDatabricksVolume) grants() ([]any, error) {
	ws, err := workspaceClient(r.MqlRuntime)
	if err != nil {
		return nil, err
	}
	return mqlDatabricksGrants(r.MqlRuntime, ws, catalog.SecurableTypeVolume, r.FullName.Data)
}
