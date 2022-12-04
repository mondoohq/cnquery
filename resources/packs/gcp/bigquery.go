package gcp

import (
	"context"
	"errors"

	"cloud.google.com/go/bigquery"
	"go.mondoo.com/cnquery/resources/packs/core"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

func (g *mqlGcloudBigquery) id() (string, error) {
	return "gcloud.bigquery", nil
}

func (g *mqlGcloudBigquery) GetDatasets() ([]interface{}, error) {
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

		defaultEncryptionConfig, err := core.JsonToDict(metadata.DefaultEncryptionConfig)
		if err != nil {
			return nil, err
		}

		mqlInstance, err := g.MotorRuntime.CreateResource("gcloud.bigquery.dataset",
			"id", dataset.DatasetID,
			"projectId", dataset.ProjectID,
			"description", metadata.Description,
			"location", metadata.Location,
			"labels", core.StrMapToInterface(metadata.Labels),
			"created", &metadata.CreationTime,
			"modified", &metadata.LastModifiedTime,
			"tags", core.StrMapToInterface(tags),
			"defaultEncryptionConfig", defaultEncryptionConfig,
		)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlInstance)
	}

	return res, nil
}

func (g *mqlGcloudBigqueryDataset) id() (string, error) {
	name, err := g.Id()
	if err != nil {
		return "", err
	}
	return "gcloud.bigquery.dataset/" + name, nil
}

func (g *mqlGcloudBigqueryDataset) GetTables() ([]interface{}, error) {
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

		mqlInstance, err := g.MotorRuntime.CreateResource("gcloud.bigquery.table",
			"name", metadata.Name,
			"location", metadata.Location,
			"description", metadata.Description,
			"fullQualifiedName", table.FullyQualifiedName(),
			"labels", core.StrMapToInterface(metadata.Labels),
			"useLegacySQL", metadata.UseLegacySQL,
			"useStandardSQL", metadata.UseStandardSQL,
			"requirePartitionFilter", metadata.RequirePartitionFilter,
			"created", &metadata.CreationTime,
			"modified", &metadata.LastModifiedTime,
			"numBytes", metadata.NumBytes,
			"numLongTermBytes", metadata.NumLongTermBytes,
			"numRows", int64(metadata.NumRows),
			"type", string(metadata.Type),
			"expirationTime", &metadata.ExpirationTime,
		)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlInstance)

	}
	return res, nil
}

func (g *mqlGcloudBigqueryTable) id() (string, error) {
	name, err := g.Name()
	if err != nil {
		return "", err
	}
	return "gcloud.bigquery.table/" + name, nil
}

func (g *mqlGcloudBigqueryDataset) GetModels() ([]interface{}, error) {
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

		mqlInstance, err := g.MotorRuntime.CreateResource("gcloud.bigquery.model",
			"id", model.ModelID,
			"datasetId", model.DatasetID,
			"projectId", model.ProjectID,
			"name", metadata.Name,
			"location", metadata.Location,
			"description", metadata.Description,
			"fullQualifiedName", model.FullyQualifiedName(),
			"location", metadata.Location,
			"labels", core.StrMapToInterface(metadata.Labels),
			"created", &metadata.CreationTime,
			"modified", &metadata.LastModifiedTime,
			"type", string(metadata.Type),
			"expirationTime", &metadata.ExpirationTime,
		)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlInstance)

	}
	return res, nil
}

func (g *mqlGcloudBigqueryModel) id() (string, error) {
	name, err := g.Name()
	if err != nil {
		return "", err
	}
	return "gcloud.bigquery.model/" + name, nil
}

func (g *mqlGcloudBigqueryDataset) GetRoutines() ([]interface{}, error) {
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

		mqlInstance, err := g.MotorRuntime.CreateResource("gcloud.bigquery.routine",
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

func (g *mqlGcloudBigqueryRoutine) id() (string, error) {
	name, err := g.Id()
	if err != nil {
		return "", err
	}
	return "gcloud.bigquery.routine/" + name, nil
}
