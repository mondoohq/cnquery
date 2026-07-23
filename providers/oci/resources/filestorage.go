// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/filestorage"
	"github.com/oracle/oci-go-sdk/v65/identity"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/mql/v13/providers/oci/connection"
	"go.mondoo.com/mql/v13/types"
)

func (o *mqlOciFileStorage) id() (string, error) {
	return "oci.fileStorage", nil
}

func (o *mqlOciFileStorage) fileSystems() ([]any, error) {
	conn := o.MqlRuntime.Connection.(*connection.OciConnection)

	ociResource, err := CreateResource(o.MqlRuntime, "oci", nil)
	if err != nil {
		return nil, err
	}
	oci := ociResource.(*mqlOci)
	list := oci.GetRegions()
	if list.Error != nil {
		return nil, list.Error
	}

	return ociRunRegionPool(o.getFileSystems(conn, list.Data))
}

func (o *mqlOciFileStorage) getFileSystemsForAD(ctx context.Context, fsClient *filestorage.FileStorageClient, compartmentID string, availabilityDomain string) ([]filestorage.FileSystemSummary, error) {
	fileSystems := []filestorage.FileSystemSummary{}
	var page *string
	for {
		request := filestorage.ListFileSystemsRequest{
			CompartmentId:      common.String(compartmentID),
			AvailabilityDomain: common.String(availabilityDomain),
			Page:               page,
		}

		response, err := fsClient.ListFileSystems(ctx, request)
		if err != nil {
			return nil, err
		}

		fileSystems = append(fileSystems, response.Items...)

		if response.OpcNextPage == nil {
			break
		}

		page = response.OpcNextPage
	}

	return fileSystems, nil
}

func (o *mqlOciFileStorage) getFileSystems(conn *connection.OciConnection, regions []any) []*jobpool.Job {
	ctx := context.Background()
	tasks := make([]*jobpool.Job, 0)
	for _, region := range regions {
		regionResource, ok := region.(*mqlOciRegion)
		if !ok {
			return jobErr(errors.New("invalid region type"))
		}
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("calling oci with region %s", regionResource.Id.Data)

			// Get availability domains for this region
			identityClient, err := conn.IdentityClientWithRegion(regionResource.Id.Data)
			if err != nil {
				return nil, err
			}

			adResponse, err := identityClient.ListAvailabilityDomains(ctx, identity.ListAvailabilityDomainsRequest{
				CompartmentId: common.String(conn.TenantID()),
			})
			if err != nil {
				return nil, err
			}

			fsClient, err := conn.FileStorageClient(regionResource.Id.Data)
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
						"systemTags":         llx.MapData(definedTagsToAny(fs.SystemTags), types.Dict),
					})
					if err != nil {
						return nil, err
					}
					mqlFs := mqlInstance.(*mqlOciFileStorageFileSystem)
					mqlFs.cacheKmsKeyId = stringValue(fs.KmsKeyId)
					mqlFs.cacheParentFileSystemId = parentFileSystemId
					res = append(res, mqlFs)
				}
			}

			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

type mqlOciFileStorageFileSystemInternal struct {
	cacheKmsKeyId           string
	cacheParentFileSystemId string
}

func (o *mqlOciFileStorageFileSystem) id() (string, error) {
	return "oci.fileStorage.fileSystem/" + o.Id.Data, nil
}

func (o *mqlOciFileStorageFileSystem) kmsKey() (*mqlOciKmsKey, error) {
	if o.cacheKmsKeyId == "" || !isOcid(o.cacheKmsKeyId) {
		o.KmsKey.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	mqlKey, err := NewResource(o.MqlRuntime, "oci.kms.key", map[string]*llx.RawData{
		"id": llx.StringData(o.cacheKmsKeyId),
	})
	if err != nil {
		return nil, err
	}
	return mqlKey.(*mqlOciKmsKey), nil
}
