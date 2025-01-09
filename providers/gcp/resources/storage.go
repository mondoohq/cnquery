// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v11/providers/gcp/connection"
	"go.mondoo.com/cnquery/v11/types"
	"google.golang.org/api/cloudresourcemanager/v3"
	"google.golang.org/api/iam/v1"
	"google.golang.org/api/option"
	"google.golang.org/api/storage/v1"
)

func (g *mqlGcpProjectStorageService) id() (string, error) {
	if g.ProjectId.Error != nil {
		return "", g.ProjectId.Error
	}
	projectId := g.ProjectId.Data
	return fmt.Sprintf("gcp.project.storageService/%s", projectId), nil
}

func initGcpProjectStorageService(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}

	conn, ok := runtime.Connection.(*connection.GcpConnection)
	if !ok {
		return nil, nil, errors.New("invalid connection provided, it is not a GCP connection")
	}
	projectId := conn.ResourceID()
	args["projectId"] = llx.StringData(projectId)

	return args, nil, nil
}

func (g *mqlGcpProject) storage() (*mqlGcpProjectStorageService, error) {
	if g.Id.Error != nil {
		return nil, g.Id.Error
	}
	projectId := g.Id.Data

	res, err := CreateResource(g.MqlRuntime, "gcp.project.storageService", map[string]*llx.RawData{
		"projectId": llx.StringData(projectId),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlGcpProjectStorageService), nil
}

func (g *mqlGcpProjectStorageService) buckets() ([]interface{}, error) {
	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)

	client, err := conn.Client(cloudresourcemanager.CloudPlatformReadOnlyScope, iam.CloudPlatformScope, storage.CloudPlatformScope)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	storageSvc, err := storage.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}

	projectID := conn.ResourceID()
	buckets, err := storageSvc.Buckets.List(projectID).Do()
	if err != nil {
		return nil, err
	}

	res := make([]interface{}, 0, len(buckets.Items))
	for i := range buckets.Items {
		bucket := buckets.Items[i]
		created := parseTime(bucket.TimeCreated)
		updated := parseTime(bucket.Updated)

		var iamConfigurationDict map[string]interface{}
		iamConfigurationDict, err = convert.JsonToDict(bucket.IamConfiguration)
		if err != nil {
			return nil, err
		}

		var retentionPolicy map[string]interface{}
		retentionPolicy, err = convert.JsonToDict(bucket.RetentionPolicy)
		if err != nil {
			return nil, err
		}
		enc, err := convert.JsonToDict(bucket.Encryption)
		if err != nil {
			return nil, err
		}

		mqlInstance, err := CreateResource(g.MqlRuntime, "gcp.project.storageService.bucket", map[string]*llx.RawData{
			"id":               llx.StringData(bucket.Id),
			"projectId":        llx.StringData(projectId),
			"name":             llx.StringData(bucket.Name),
			"labels":           llx.MapData(convert.MapToInterfaceMap(bucket.Labels), types.String),
			"location":         llx.StringData(bucket.Location),
			"locationType":     llx.StringData(bucket.LocationType),
			"projectNumber":    llx.StringData(strconv.FormatUint(bucket.ProjectNumber, 10)),
			"storageClass":     llx.StringData(bucket.StorageClass),
			"created":          llx.TimeDataPtr(created),
			"updated":          llx.TimeDataPtr(updated),
			"iamConfiguration": llx.DictData(iamConfigurationDict),
			"retentionPolicy":  llx.DictData(retentionPolicy),
			"encryption":       llx.DictData(enc),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlInstance)
	}
	return res, nil
}

func (g *mqlGcpProjectStorageServiceBucket) id() (string, error) {
	if g.Id.Error != nil {
		return "", g.Id.Error
	}
	id := g.Id.Data

	if g.ProjectId.Error != nil {
		return "", g.ProjectId.Error
	}
	projectId := g.ProjectId.Data
	return fmt.Sprintf("gcp.project.storageService.bucket/%s/%s", projectId, id), nil
}

func initGcpProjectStorageServiceBucket(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}

	// If no args are set, try reading them from the platform ID
	if len(args) == 0 {
		if ids := getAssetIdentifier(runtime); ids != nil {
			args["name"] = llx.StringData(ids.name)
			args["projectId"] = llx.StringData(ids.project)
			args["location"] = llx.StringData(ids.region)
		} else {
			return nil, nil, errors.New("no asset identifier found")
		}
	}

	obj, err := CreateResource(runtime, "gcp.project.storageService", map[string]*llx.RawData{
		"projectId": llx.StringData(args["projectId"].Value.(string)),
	})
	if err != nil {
		return nil, nil, err
	}
	storageSvc := obj.(*mqlGcpProjectStorageService)
	buckets := storageSvc.GetBuckets()
	if buckets.Error != nil {
		return nil, nil, buckets.Error
	}

	for _, b := range buckets.Data {
		bucket := b.(*mqlGcpProjectStorageServiceBucket)

		if bucket.Name.Error != nil {
			return nil, nil, bucket.Name.Error
		}
		name := bucket.Name.Data

		if bucket.ProjectId.Error != nil {
			return nil, nil, bucket.ProjectId.Error
		}
		projectId := bucket.ProjectId.Data

		if bucket.Location.Error != nil {
			return nil, nil, bucket.Location.Error
		}
		location := bucket.Location.Data

		if name == args["name"].Value.(string) && projectId == args["projectId"].Value.(string) && location == args["location"].Value.(string) {
			return args, bucket, nil
		}
	}
	return nil, nil, errors.New("bucket not found")
}

func (g *mqlGcpProjectStorageServiceBucket) iamPolicy() ([]interface{}, error) {
	if g.Name.Error != nil {
		return nil, g.Name.Error
	}
	bucketName := g.Name.Data

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)

	client, err := conn.Client(cloudresourcemanager.CloudPlatformReadOnlyScope, iam.CloudPlatformScope, storage.CloudPlatformScope)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	storeSvc, err := storage.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}

	policy, err := storeSvc.Buckets.GetIamPolicy(bucketName).Do()
	if err != nil {
		return nil, err
	}

	res := []interface{}{}
	for i := range policy.Bindings {
		b := policy.Bindings[i]

		mqlServiceaccount, err := CreateResource(g.MqlRuntime, "gcp.resourcemanager.binding", map[string]*llx.RawData{
			"id":      llx.StringData(bucketName + "-" + strconv.Itoa(i)),
			"role":    llx.StringData(b.Role),
			"members": llx.ArrayData(convert.SliceAnyToInterface(b.Members), types.String),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlServiceaccount)
	}

	return res, nil
}
