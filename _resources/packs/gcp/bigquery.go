// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package gcp

import (
	"context"
	"errors"
	"fmt"
	"time"

	"cloud.google.com/go/bigquery"
	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/resources/packs/core"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

func (g *mqlGcpProjectBigqueryService) init(args *resources.Args) (*resources.Args, GcpProjectBigqueryService, error) {
	if len(*args) > 2 {
		return args, nil, nil
	}

	provider, err := gcpProvider(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, nil, err
	}

	projectId := provider.ResourceID()
	(*args)["projectId"] = projectId

	return args, nil, nil
}

func (g *mqlGcpProjectBigqueryService) id() (string, error) {
	projectId, err := g.ProjectId()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("gcp.project.bigqueryService/%s", projectId), nil
}

func (g *mqlGcpProject) GetBigquery() (interface{}, error) {
	projectId, err := g.Id()
	if err != nil {
		return nil, err
	}

	return g.MotorRuntime.CreateResource("gcp.project.bigqueryService",
		"projectId", projectId,
	)
}

func (g *mqlGcpProjectBigqueryService) GetDatasets() ([]interface{}, error) {
	provider, err := gcpProvider(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	projectId, err := g.ProjectId()
	if err != nil {
		return nil, err
	}

	client, err := provider.Client()
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
			mqlA, err := g.MotorRuntime.CreateResource("gcp.project.bigqueryService.dataset.accessEntry",
				"id", fmt.Sprintf("gcp.project.bigqueryService.dataset/%s/%s/accessEntry/%d", projectId, dataset.DatasetID, i),
				"datasetId", dataset.DatasetID,
				"role", string(a.Role),
				"entityType", entityTypeToString(a.EntityType),
				"entity", a.Entity,
				"viewRef", viewRef,
				"routineRef", routineRef,
				"datasetRef", datasetRef,
			)
			if err != nil {
				return nil, err
			}
			access = append(access, mqlA)
		}

		mqlInstance, err := g.MotorRuntime.CreateResource("gcp.project.bigqueryService.dataset",
			"id", dataset.DatasetID,
			"projectId", dataset.ProjectID,
			"name", metadata.Name,
			"description", metadata.Description,
			"location", metadata.Location,
			"labels", core.StrMapToInterface(metadata.Labels),
			"created", &metadata.CreationTime,
			"modified", &metadata.LastModifiedTime,
			"tags", core.StrMapToInterface(tags),
			"kmsName", kmsName,
			"access", access,
		)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlInstance)
	}

	return res, nil
}

func (g *mqlGcpProjectBigqueryServiceDataset) id() (string, error) {
	projectId, err := g.ProjectId()
	if err != nil {
		return "", err
	}

	name, err := g.Id()
	if err != nil {
		return "", err
	}
	return "gcp.project.bigqueryService.dataset/" + projectId + "/" + name, nil
}

func (g *mqlGcpProjectBigqueryServiceDataset) init(args *resources.Args) (*resources.Args, GcpProjectBigqueryServiceDataset, error) {
	if len(*args) > 2 {
		return args, nil, nil
	}

	// If no args are set, try reading them from the platform ID
	if len(*args) == 0 {
		if ids := getAssetIdentifier(g.MotorRuntime); ids != nil {
			(*args)["id"] = ids.name
			(*args)["location"] = ids.region
			(*args)["projectId"] = ids.project
		}
	}

	obj, err := g.MotorRuntime.CreateResource("gcp.project.bigqueryService", "projectId", (*args)["projectId"])
	if err != nil {
		return nil, nil, err
	}
	bigquerySvc := obj.(GcpProjectBigqueryService)
	datasets, err := bigquerySvc.Datasets()
	if err != nil {
		return nil, nil, err
	}

	for _, d := range datasets {
		dataset := d.(GcpProjectBigqueryServiceDataset)
		id, err := dataset.Id()
		if err != nil {
			return nil, nil, err
		}
		location, err := dataset.Location()
		if err != nil {
			return nil, nil, err
		}
		projectId, err := dataset.ProjectId()
		if err != nil {
			return nil, nil, err
		}

		if id == (*args)["id"] && projectId == (*args)["projectId"] && location == (*args)["location"] {
			return args, dataset, nil
		}
	}
	return nil, nil, &resources.ResourceNotFound{}
}

func (g *mqlGcpProjectBigqueryServiceDatasetAccessEntry) id() (string, error) {
	return g.Id()
}

func (g *mqlGcpProjectBigqueryServiceDataset) GetTables() ([]interface{}, error) {
	provider, err := gcpProvider(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	client, err := provider.Client()
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	projectID := provider.ResourceID()
	bigquerySvc, err := bigquery.NewClient(ctx, projectID, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}

	datasetID, err := g.Id()
	if err != nil {
		return nil, err
	}

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
			clusteringFields = core.StrSliceToInterface(metadata.Clustering.Fields)
		}

		externalDataConfig, err := core.JsonToDict(metadata.ExternalDataConfig)
		if err != nil {
			return nil, err
		}

		materializedView, err := core.JsonToDict(metadata.MaterializedView)
		if err != nil {
			return nil, err
		}

		rangePartitioning, err := core.JsonToDict(metadata.RangePartitioning)
		if err != nil {
			return nil, err
		}

		schema, err := core.JsonToDictSlice(metadata.Schema)
		if err != nil {
			return nil, err
		}

		timePartitioning, err := core.JsonToDict(metadata.TimePartitioning)
		if err != nil {
			return nil, err
		}

		var snapshotTime *time.Time
		if metadata.SnapshotDefinition != nil {
			snapshotTime = &metadata.SnapshotDefinition.SnapshotTime
		}

		mqlInstance, err := g.MotorRuntime.CreateResource("gcp.project.bigqueryService.table",
			"id", table.TableID,
			"projectId", table.ProjectID,
			"datasetId", table.DatasetID,
			"name", metadata.Name,
			"location", metadata.Location,
			"description", metadata.Description,
			"labels", core.StrMapToInterface(metadata.Labels),
			"useLegacySQL", metadata.UseLegacySQL,
			"requirePartitionFilter", metadata.RequirePartitionFilter,
			"created", &metadata.CreationTime,
			"modified", &metadata.LastModifiedTime,
			"numBytes", metadata.NumBytes,
			"numLongTermBytes", metadata.NumLongTermBytes,
			"numRows", int64(metadata.NumRows),
			"type", string(metadata.Type),
			"expirationTime", &metadata.ExpirationTime,
			"kmsName", kmsName,
			"snapshotTime", snapshotTime,
			"viewQuery", metadata.ViewQuery,
			"clusteringFields", clusteringFields,
			"externalDataConfig", externalDataConfig,
			"materializedView", materializedView,
			"rangePartitioning", rangePartitioning,
			"timePartitioning", timePartitioning,
			"schema", schema,
		)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlInstance)

	}
	return res, nil
}

func (g *mqlGcpProjectBigqueryServiceTable) id() (string, error) {
	projectId, err := g.ProjectId()
	if err != nil {
		return "", err
	}
	datasetId, err := g.DatasetId()
	if err != nil {
		return "", err
	}
	id, err := g.Id()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("gcp.project.bigqueryService.table/%s/%s/%s", projectId, datasetId, id), nil
}

func (g *mqlGcpProjectBigqueryServiceDataset) GetModels() ([]interface{}, error) {
	provider, err := gcpProvider(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	client, err := provider.Client()
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	projectID := provider.ResourceID()
	bigquerySvc, err := bigquery.NewClient(ctx, projectID, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}

	datasetID, err := g.Id()
	if err != nil {
		return nil, err
	}

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

		mqlInstance, err := g.MotorRuntime.CreateResource("gcp.project.bigqueryService.model",
			"id", model.ModelID,
			"datasetId", model.DatasetID,
			"projectId", model.ProjectID,
			"name", metadata.Name,
			"location", metadata.Location,
			"description", metadata.Description,
			"location", metadata.Location,
			"labels", core.StrMapToInterface(metadata.Labels),
			"created", &metadata.CreationTime,
			"modified", &metadata.LastModifiedTime,
			"type", string(metadata.Type),
			"expirationTime", &metadata.ExpirationTime,
			"kmsName", kmsName,
		)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlInstance)

	}
	return res, nil
}

func (g *mqlGcpProjectBigqueryServiceModel) id() (string, error) {
	projectId, err := g.ProjectId()
	if err != nil {
		return "", err
	}
	id, err := g.Id()
	if err != nil {
		return "", err
	}
	datasetId, err := g.DatasetId()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("gcp.project.bigqueryService.model/%s/%s/%s", projectId, datasetId, id), nil
}

func (g *mqlGcpProjectBigqueryServiceDataset) GetRoutines() ([]interface{}, error) {
	provider, err := gcpProvider(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	client, err := provider.Client()
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	projectID := provider.ResourceID()
	bigquerySvc, err := bigquery.NewClient(ctx, projectID, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}

	datasetID, err := g.Id()
	if err != nil {
		return nil, err
	}

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

		mqlInstance, err := g.MotorRuntime.CreateResource("gcp.project.bigqueryService.routine",
			"id", routine.RoutineID,
			"datasetId", routine.DatasetID,
			"projectId", routine.ProjectID,
			"language", metadata.Language,
			"description", metadata.Description,
			"created", &metadata.CreationTime,
			"modified", &metadata.LastModifiedTime,
			"type", string(metadata.Type),
		)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlInstance)

	}
	return res, nil
}

func (g *mqlGcpProjectBigqueryServiceRoutine) id() (string, error) {
	name, err := g.Id()
	if err != nil {
		return "", err
	}
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
