package gcp

import (
	"context"
	"errors"
	"time"

	"cloud.google.com/go/bigquery"
	"go.mondoo.com/cnquery/resources/packs/core"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

func (g *mqlGcpBigquery) id() (string, error) {
	return "gcp.bigquery", nil
}

func (g *mqlGcpBigquery) GetDatasets() ([]interface{}, error) {
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

		mqlInstance, err := g.MotorRuntime.CreateResource("gcp.bigquery.dataset",
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
		)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlInstance)
	}

	return res, nil
}

func (g *mqlGcpBigqueryDataset) id() (string, error) {
	projectId, err := g.ProjectId()
	if err != nil {
		return "", err
	}

	name, err := g.Id()
	if err != nil {
		return "", err
	}
	return "gcp.bigquery.dataset/" + projectId + "/" + name, nil
}

func (g *mqlGcpBigqueryDataset) GetTables() ([]interface{}, error) {
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

		mqlInstance, err := g.MotorRuntime.CreateResource("gcp.bigquery.table",
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

func (g *mqlGcpBigqueryTable) id() (string, error) {
	name, err := g.Name()
	if err != nil {
		return "", err
	}
	return "gcp.bigquery.table/" + name, nil
}

func (g *mqlGcpBigqueryDataset) GetModels() ([]interface{}, error) {
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

		mqlInstance, err := g.MotorRuntime.CreateResource("gcp.bigquery.model",
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

func (g *mqlGcpBigqueryModel) id() (string, error) {
	name, err := g.Name()
	if err != nil {
		return "", err
	}
	return "gcp.bigquery.model/" + name, nil
}

func (g *mqlGcpBigqueryDataset) GetRoutines() ([]interface{}, error) {
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

		mqlInstance, err := g.MotorRuntime.CreateResource("gcp.bigquery.routine",
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

func (g *mqlGcpBigqueryRoutine) id() (string, error) {
	name, err := g.Id()
	if err != nil {
		return "", err
	}
	return "gcp.bigquery.routine/" + name, nil
}
