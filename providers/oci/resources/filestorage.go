// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/filestorage"
	"github.com/oracle/oci-go-sdk/v65/identity"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/oci/connection"
	"go.mondoo.com/mql/v13/types"
)

func (o *mqlOciFileStorage) id() (string, error) {
	return "oci.fileStorage", nil
}

func (o *mqlOciFileStorage) fileSystems() ([]any, error) {
	conn := o.MqlRuntime.Connection.(*connection.OciConnection)

	return ociCollect(o.MqlRuntime, ociScopeTenancyRoot,
		func(ctx context.Context, region string, compartmentID string) ([]any, error) {
			log.Debug().Msgf("calling oci with region %s", region)

			// Get availability domains for this region
			identityClient, err := conn.IdentityClientWithRegion(region)
			if err != nil {
				return nil, err
			}

			adResponse, err := identityClient.ListAvailabilityDomains(ctx, identity.ListAvailabilityDomainsRequest{
				CompartmentId: common.String(compartmentID),
			})
			if err != nil {
				return nil, err
			}

			fsClient, err := conn.FileStorageClient(region)
			if err != nil {
				return nil, err
			}

			var res []any
			for _, ad := range adResponse.Items {
				if ad.Name == nil {
					continue
				}

				fileSystems, err := o.getFileSystemsForAD(ctx, fsClient, conn.TenantID(), *ad.Name)
				if err != nil {
					return nil, err
				}

				for i := range fileSystems {
					fs := fileSystems[i]

					var created *time.Time
					if fs.TimeCreated != nil {
						created = &fs.TimeCreated.Time
					}

					var parentFileSystemId string
					if fs.SourceDetails != nil {
						parentFileSystemId = stringValue(fs.SourceDetails.ParentFileSystemId)
					}

					mqlInstance, err := CreateResource(o.MqlRuntime, "oci.fileStorage.fileSystem", map[string]*llx.RawData{
						"id":                 llx.StringDataPtr(fs.Id),
						"name":               llx.StringDataPtr(fs.DisplayName),
						"compartmentID":      llx.StringDataPtr(fs.CompartmentId),
						"availabilityDomain": llx.StringDataPtr(fs.AvailabilityDomain),
						"state":              llx.StringData(string(fs.LifecycleState)),
						"meteredBytes":       llx.IntDataPtr(fs.MeteredBytes),
						"isCloneParent":      llx.BoolDataPtr(fs.IsCloneParent),
						"created":            llx.TimeDataPtr(created),
						"freeformTags":       llx.MapData(strMapToAny(fs.FreeformTags), types.String),
						"definedTags":        llx.MapData(definedTagsToAny(fs.DefinedTags), types.Any),
						"systemTags":         llx.MapData(definedTagsToAny(fs.SystemTags), types.Dict),
					})
					if err != nil {
						return nil, err
					}
					mqlFs := mqlInstance.(*mqlOciFileStorageFileSystem)
					mqlFs.cacheKmsKeyID = stringValue(fs.KmsKeyId)
					mqlFs.cacheParentFileSystemID = parentFileSystemId
					res = append(res, mqlFs)
				}
			}

			return res, nil
		})
}

func (o *mqlOciFileStorage) getFileSystemsForAD(ctx context.Context, fsClient *filestorage.FileStorageClient, compartmentID string, availabilityDomain string) ([]filestorage.FileSystemSummary, error) {
	fileSystems, err := ociPaginate(ctx, func(ctx context.Context, page *string) ([]filestorage.FileSystemSummary, *string, error) {
		request := filestorage.ListFileSystemsRequest{
			CompartmentId:      common.String(compartmentID),
			AvailabilityDomain: common.String(availabilityDomain),
			Page:               page,
		}

		response, err := fsClient.ListFileSystems(ctx, request)
		if err != nil {
			return nil, nil, err
		}
		return response.Items, response.OpcNextPage, nil
	})
	if err != nil {
		return nil, err
	}

	return fileSystems, nil
}

type mqlOciFileStorageFileSystemInternal struct {
	cacheKmsKeyID           string
	cacheParentFileSystemID string
}

func (o *mqlOciFileStorageFileSystem) id() (string, error) {
	return "oci.fileStorage.fileSystem/" + o.Id.Data, nil
}

func (o *mqlOciFileStorageFileSystem) kmsKey() (*mqlOciKmsKey, error) {
	if o.cacheKmsKeyID == "" || !isOcid(o.cacheKmsKeyID) {
		o.KmsKey.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	mqlKey, err := NewResource(o.MqlRuntime, "oci.kms.key", map[string]*llx.RawData{
		"id": llx.StringData(o.cacheKmsKeyID),
	})
	if err != nil {
		return nil, err
	}
	return mqlKey.(*mqlOciKmsKey), nil
}
