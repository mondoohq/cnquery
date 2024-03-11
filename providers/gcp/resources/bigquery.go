// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mondoo.com/cnquery/v10/llx"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v10/providers/gcp/connection"
	"go.mondoo.com/cnquery/v10/types"

	"cloud.google.com/go/bigquery"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

func initGcpProjectBigqueryService(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}

	conn := runtime.Connection.(*connection.GcpConnection)

	projectId := conn.ResourceID()
	args["projectId"] = llx.StringData(projectId)

	return args, nil, nil
}

func (g *mqlGcpProjectBigqueryService) id() (string, error) {
	if g.ProjectId.Error != nil {
		return "", g.ProjectId.Error
	}
	projectId := g.ProjectId.Data
	return fmt.Sprintf("gcp.project.bigqueryService/%s", projectId), nil
}

func (g *mqlGcpProject) bigquery() (*mqlGcpProjectBigqueryService, error) {
	if g.Id.Error != nil {
		return nil, g.Id.Error
	}
	projectId := g.Id.Data

	res, err := CreateResource(g.MqlRuntime, "gcp.project.bigqueryService", map[string]*llx.RawData{
		"projectId": llx.StringData(projectId),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlGcpProjectBigqueryService), nil
}

func (g *mqlGcpProjectBigqueryService) datasets() ([]interface{}, error) {
	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)

	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	client, err := conn.Client()
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	bigquerySvc, err := bigquery.NewClient(ctx, projectId, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}

	it := bigquerySvc.Datasets(ctx)
	res := []interface{}{}
	for {
		dataset, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		metadata, err := dataset.Metadata(ctx)
		if err != nil {
			return nil, err
		}

		tags := map[string]string{}
		for i := range metadata.Tags {
			tag := metadata.Tags[i]
			tags[tag.TagKey] = tag.TagValue
		}

		var kmsName string
		if metadata.DefaultEncryptionConfig != nil {
			kmsName = metadata.DefaultEncryptionConfig.KMSKeyName
		}

		access := make([]interface{}, 0, len(metadata.Access))
		for i, a := range metadata.Access {
			var viewRef interface{}
			if a.View != nil {
				viewRef = map[string]interface{}{
					"projectId": a.View.ProjectID,
					"datasetId": a.View.DatasetID,
					"tableId":   a.View.TableID,
				}
			}
			var routineRef interface{}
			if a.Routine != nil {
				routineRef = map[string]interface{}{
					"projectId": a.Routine.ProjectID,
					"datasetId": a.Routine.DatasetID,
					"tableId":   a.Routine.RoutineID,
				}
			}
			var datasetRef interface{}
			if a.Dataset != nil {
				datasetRef = map[string]interface{}{
					"projectId":   a.Dataset.Dataset.ProjectID,
					"datasetId":   a.Dataset.Dataset.DatasetID,
					"targetTypes": a.Dataset.TargetTypes,
				}
			}
			mqlA, err := CreateResource(g.MqlRuntime, "gcp.project.bigqueryService.dataset.accessEntry", map[string]*llx.RawData{
				"id":         llx.StringData(fmt.Sprintf("gcp.project.bigqueryService.dataset/%s/%s/accessEntry/%d", projectId, dataset.DatasetID, i)),
				"datasetId":  llx.StringData(dataset.DatasetID),
				"role":       llx.StringData(string(a.Role)),
				"entityType": llx.StringData(entityTypeToString(a.EntityType)),
				"entity":     llx.StringData(a.Entity),
				"viewRef":    llx.DictData(viewRef),
				"routineRef": llx.DictData(routineRef),
				"datasetRef": llx.DictData(datasetRef),
			})
			if err != nil {
				return nil, err
			}
			access = append(access, mqlA)
		}

		mqlInstance, err := CreateResource(g.MqlRuntime, "gcp.project.bigqueryService.dataset", map[string]*llx.RawData{
			"id":          llx.StringData(dataset.DatasetID),
			"projectId":   llx.StringData(dataset.ProjectID),
			"name":        llx.StringData(metadata.Name),
			"description": llx.StringData(metadata.Description),
			"location":    llx.StringData(metadata.Location),
			"labels":      llx.MapData(convert.MapToInterfaceMap(metadata.Labels), types.String),
			"created":     llx.TimeData(metadata.CreationTime),
			"modified":    llx.TimeData(metadata.LastModifiedTime),
			"tags":        llx.MapData(convert.MapToInterfaceMap(tags), types.String),
			"kmsName":     llx.StringData(kmsName),
			"access":      llx.ArrayData(access, types.Resource("gcp.project.bigqueryService.dataset.accessEntry")),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlInstance)
	}

	return res, nil
}

func (g *mqlGcpProjectBigqueryServiceDataset) id() (string, error) {
	if g.ProjectId.Error != nil {
		return "", g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	if g.Id.Error != nil {
		return "", g.Id.Error
	}
	name := g.Id.Data
	return "gcp.project.bigqueryService.dataset/" + projectId + "/" + name, nil
}

func initGcpProjectBigqueryServiceDataset(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}

	// If no args are set, try reading them from the platform ID
	if len(args) == 0 {
		if ids := getAssetIdentifier(runtime); ids != nil {
			args["id"] = llx.StringData(ids.name)
			args["location"] = llx.StringData(ids.region)
			args["projectId"] = llx.StringData(ids.project)
		} else {
			return nil, nil, errors.New("no asset identifier found")
		}
	}

	obj, err := CreateResource(runtime, "gcp.project.bigqueryService", map[string]*llx.RawData{
		"projectId": llx.StringData(args["projectId"].Value.(string)),
	})
	if err != nil {
		return nil, nil, err
	}
	bigquerySvc := obj.(*mqlGcpProjectBigqueryService)
	datasets := bigquerySvc.GetDatasets()
	if datasets.Error != nil {
		return nil, nil, datasets.Error
	}

	for _, d := range datasets.Data {
		dataset := d.(*mqlGcpProjectBigqueryServiceDataset)
		id := dataset.GetId()
		if id.Error != nil {
			return nil, nil, id.Error
		}
		location := dataset.GetLocation()
		if location.Error != nil {
			return nil, nil, location.Error
		}
		projectId := dataset.GetProjectId()
		if projectId.Error != nil {
			return nil, nil, projectId.Error
		}

		if id == args["id"].Value && projectId == args["projectId"].Value && location == args["location"].Value {
			return args, dataset, nil
		}
	}
	return nil, nil, nil
}

func (g *mqlGcpProjectBigqueryServiceDatasetAccessEntry) id() (string, error) {
	return g.Id.Data, g.Id.Error
}

func (g *mqlGcpProjectBigqueryServiceDataset) tables() ([]interface{}, error) {
	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)

	client, err := conn.Client()
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	projectID := conn.ResourceID()
	bigquerySvc, err := bigquery.NewClient(ctx, projectID, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}

	if g.Id.Error != nil {
		return nil, g.Id.Error
	}
	datasetID := g.Id.Data

	dataset := bigquerySvc.Dataset(datasetID)
	if dataset == nil {
		return nil, errors.New("could not find dataset:" + datasetID)
	}

	it := dataset.Tables(ctx)
	res := []interface{}{}
	for {
		table, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		metadata, err := table.Metadata(ctx)
		if err != nil {
			return nil, err
		}

		var kmsName string
		if metadata.EncryptionConfig != nil {
			kmsName = metadata.EncryptionConfig.KMSKeyName
		}

		var clusteringFields []interface{}
		if metadata.Clustering != nil {
			clusteringFields = convert.SliceAnyToInterface(metadata.Clustering.Fields)
		}

		externalDataConfig, err := convert.JsonToDict(metadata.ExternalDataConfig)
		if err != nil {
			return nil, err
		}

		materializedView, err := convert.JsonToDict(metadata.MaterializedView)
		if err != nil {
			return nil, err
		}

		rangePartitioning, err := convert.JsonToDict(metadata.RangePartitioning)
		if err != nil {
			return nil, err
		}

		schema, err := convert.JsonToDictSlice(metadata.Schema)
		if err != nil {
			return nil, err
		}

		timePartitioning, err := convert.JsonToDict(metadata.TimePartitioning)
		if err != nil {
			return nil, err
		}

		var snapshotTime *time.Time
		if metadata.SnapshotDefinition != nil {
			snapshotTime = &metadata.SnapshotDefinition.SnapshotTime
		}

		mqlInstance, err := CreateResource(g.MqlRuntime, "gcp.project.bigqueryService.table", map[string]*llx.RawData{
			"id":                     llx.StringData(table.TableID),
			"projectId":              llx.StringData(table.ProjectID),
			"datasetId":              llx.StringData(table.DatasetID),
			"name":                   llx.StringData(metadata.Name),
			"location":               llx.StringData(metadata.Location),
			"description":            llx.StringData(metadata.Description),
			"labels":                 llx.MapData(convert.MapToInterfaceMap(metadata.Labels), types.String),
			"useLegacySQL":           llx.BoolData(metadata.UseLegacySQL),
			"requirePartitionFilter": llx.BoolData(metadata.RequirePartitionFilter),
			"created":                llx.TimeData(metadata.CreationTime),
			"modified":               llx.TimeData(metadata.LastModifiedTime),
			"numBytes":               llx.IntData(metadata.NumBytes),
			"numLongTermBytes":       llx.IntData(metadata.NumLongTermBytes),
			"numRows":                llx.IntData(int64(metadata.NumRows)),
			"type":                   llx.StringData(string(metadata.Type)),
			"expirationTime":         llx.TimeData(metadata.ExpirationTime),
			"kmsName":                llx.StringData(kmsName),
			"snapshotTime":           llx.TimeDataPtr(snapshotTime),
			"viewQuery":              llx.StringData(metadata.ViewQuery),
			"clusteringFields":       llx.DictData(clusteringFields),
			"externalDataConfig":     llx.DictData(externalDataConfig),
			"materializedView":       llx.DictData(materializedView),
			"rangePartitioning":      llx.DictData(rangePartitioning),
			"timePartitioning":       llx.DictData(timePartitioning),
			"schema":                 llx.ArrayData(schema, types.Dict),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlInstance)

	}
	return res, nil
}

func (g *mqlGcpProjectBigqueryServiceTable) id() (string, error) {
	if g.ProjectId.Error != nil {
		return "", g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	if g.DatasetId.Error != nil {
		return "", g.DatasetId.Error
	}
	datasetId := g.DatasetId.Data

	if g.Id.Error != nil {
		return "", g.Id.Error
	}
	id := g.Id.Data
	return fmt.Sprintf("gcp.project.bigqueryService.table/%s/%s/%s", projectId, datasetId, id), nil
}

func (g *mqlGcpProjectBigqueryServiceDataset) models() ([]interface{}, error) {
	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)

	client, err := conn.Client()
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	projectID := conn.ResourceID()
	bigquerySvc, err := bigquery.NewClient(ctx, projectID, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}

	if g.Id.Error != nil {
		return nil, g.Id.Error
	}
	datasetID := g.Id.Data

	dataset := bigquerySvc.Dataset(datasetID)
	if dataset == nil {
		return nil, errors.New("could not find dataset:" + datasetID)
	}

	it := dataset.Models(ctx)
	res := []interface{}{}
	for {
		model, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		metadata, err := model.Metadata(ctx)
		if err != nil {
			return nil, err
		}

		var kmsName string
		if metadata.EncryptionConfig != nil {
			kmsName = metadata.EncryptionConfig.KMSKeyName
		}

		mqlInstance, err := CreateResource(g.MqlRuntime, "gcp.project.bigqueryService.model", map[string]*llx.RawData{
			"id":             llx.StringData(model.ModelID),
			"datasetId":      llx.StringData(model.DatasetID),
			"projectId":      llx.StringData(model.ProjectID),
			"name":           llx.StringData(metadata.Name),
			"description":    llx.StringData(metadata.Description),
			"location":       llx.StringData(metadata.Location),
			"labels":         llx.MapData(convert.MapToInterfaceMap(metadata.Labels), types.String),
			"created":        llx.TimeData(metadata.CreationTime),
			"modified":       llx.TimeData(metadata.LastModifiedTime),
			"type":           llx.StringData(string(metadata.Type)),
			"expirationTime": llx.TimeData(metadata.ExpirationTime),
			"kmsName":        llx.StringData(kmsName),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlInstance)

	}
	return res, nil
}

func (g *mqlGcpProjectBigqueryServiceModel) id() (string, error) {
	if g.ProjectId.Error != nil {
		return "", g.ProjectId.Error
	}
	projectId := g.ProjectId.Data
	if g.Id.Error != nil {
		return "", g.Id.Error
	}
	id := g.Id.Data

	if g.DatasetId.Error != nil {
		return "", g.DatasetId.Error
	}
	datasetId := g.DatasetId.Data
	return fmt.Sprintf("gcp.project.bigqueryService.model/%s/%s/%s", projectId, datasetId, id), nil
}

func (g *mqlGcpProjectBigqueryServiceDataset) routines() ([]interface{}, error) {
	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)

	client, err := conn.Client()
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	projectID := conn.ResourceID()
	bigquerySvc, err := bigquery.NewClient(ctx, projectID, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}

	if g.Id.Error != nil {
		return nil, g.Id.Error
	}
	datasetID := g.Id.Data

	dataset := bigquerySvc.Dataset(datasetID)
	if dataset == nil {
		return nil, errors.New("could not find dataset:" + datasetID)
	}

	it := dataset.Routines(ctx)
	res := []interface{}{}
	for {
		routine, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		metadata, err := routine.Metadata(ctx)
		if err != nil {
			return nil, err
		}

		mqlInstance, err := CreateResource(g.MqlRuntime, "gcp.project.bigqueryService.routine", map[string]*llx.RawData{
			"id":          llx.StringData(routine.RoutineID),
			"datasetId":   llx.StringData(routine.DatasetID),
			"projectId":   llx.StringData(routine.ProjectID),
			"language":    llx.StringData(metadata.Language),
			"description": llx.StringData(metadata.Description),
			"created":     llx.TimeData(metadata.CreationTime),
			"modified":    llx.TimeData(metadata.LastModifiedTime),
			"type":        llx.StringData(string(metadata.Type)),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlInstance)

	}
	return res, nil
}

func (g *mqlGcpProjectBigqueryServiceRoutine) id() (string, error) {
	if g.Id.Error != nil {
		return "", g.Id.Error
	}
	name := g.Id.Data
	return "gcp.project.bigqueryService.routine/" + name, nil
}

func entityTypeToString(entityType bigquery.EntityType) string {
	switch entityType {
	case bigquery.DomainEntity:
		return "DOMAIN"
	case bigquery.GroupEmailEntity:
		return "GROUP_EMAIL"
	case bigquery.UserEmailEntity:
		return "USER_EMAIL"
	case bigquery.SpecialGroupEntity:
		return "SPECIAL_GROUP"
	case bigquery.ViewEntity:
		return "VIEW"
	case bigquery.IAMMemberEntity:
		return "IAM_MEMBER"
	case bigquery.RoutineEntity:
		return "ROUTINE"
	case bigquery.DatasetEntity:
		return "DATASET"
	default:
		return "UNKNOWN"
	}
}
