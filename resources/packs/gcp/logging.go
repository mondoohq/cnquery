package gcp

import (
	"context"
	"fmt"

	"cloud.google.com/go/logging/logadmin"
	"go.mondoo.com/cnquery/resources/packs/core"
	"google.golang.org/api/iterator"
	"google.golang.org/api/logging/v2"
	"google.golang.org/api/option"
)

func (g *mqlGcpProjectLoggingservice) id() (string, error) {
	projectId, err := g.ProjectId()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/gcp.project.loggingservice", projectId), nil
}

func (g *mqlGcpProject) GetLogging() (interface{}, error) {
	projectId, err := g.Id()
	if err != nil {
		return nil, err
	}

	return g.MotorRuntime.CreateResource("gcp.project.loggingservice",
		"projectId", projectId,
	)
}

func (g *mqlGcpProjectLoggingservice) GetBuckets() ([]interface{}, error) {
	provider, err := gcpProvider(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	projectId, err := g.ProjectId()
	if err != nil {
		return nil, err
	}

	client, err := provider.Client(logging.CloudPlatformReadOnlyScope, logging.LoggingReadScope)
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	loggingSvc, err := logging.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}

	buckets, err := loggingSvc.Projects.Locations.Buckets.List(fmt.Sprintf("projects/%s/locations/-", projectId)).Do()
	if err != nil {
		return nil, err
	}

	mqlBuckets := make([]interface{}, 0, len(buckets.Buckets))
	for _, bucket := range buckets.Buckets {

		var mqlCmekSettingsDict map[string]interface{}
		if bucket.CmekSettings != nil {
			type mqlCmekSettings struct {
				KmsKeyName        string `json:"kmsKeyName"`
				KmsKeyVersionName string `json:"kmsKeyVersionName"`
				Name              string `json:"name"`
				ServiceAccountId  string `json:"serviceAccountId"`
			}
			mqlCmekSettingsDict, err = core.JsonToDict(mqlCmekSettings{
				KmsKeyName:        bucket.CmekSettings.KmsKeyName,
				KmsKeyVersionName: bucket.CmekSettings.KmsKeyVersionName,
				Name:              bucket.CmekSettings.Name,
				ServiceAccountId:  bucket.CmekSettings.ServiceAccountId,
			})
			if err != nil {
				return nil, err
			}
		}

		indexConfigs := make([]interface{}, 0, len(bucket.IndexConfigs))
		for i, cfg := range bucket.IndexConfigs {
			mqlIndexConfig, err := g.MotorRuntime.CreateResource("gcp.project.loggingservice.bucket.indexConfigs",
				"id", fmt.Sprintf("%s/indexConfigs/%d", bucket.Name, i),
				"created", parseTime(cfg.CreateTime),
				"fieldPath", cfg.FieldPath,
				"type", cfg.Type,
			)
			if err != nil {
				return nil, err
			}
			indexConfigs = append(indexConfigs, mqlIndexConfig)
		}

		mqlBucket, err := g.MotorRuntime.CreateResource("gcp.project.loggingservice.bucket",
			"projectId", projectId,
			"cmekSettings", mqlCmekSettingsDict,
			"created", parseTime(bucket.CreateTime),
			"description", bucket.Description,
			"indexConfigs", indexConfigs,
			"lifecycleState", bucket.LifecycleState,
			"locked", bucket.Locked,
			"name", bucket.Name,
			"restrictedFields", core.StrSliceToInterface(bucket.RestrictedFields),
			"retentionDays", bucket.RetentionDays,
			"updated", parseTime(bucket.UpdateTime),
		)
		if err != nil {
			return nil, err
		}
		mqlBuckets = append(mqlBuckets, mqlBucket)
	}
	return mqlBuckets, nil
}

func (g *mqlGcpProjectLoggingservice) GetMetrics() ([]interface{}, error) {
	provider, err := gcpProvider(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	projectId, err := g.ProjectId()
	if err != nil {
		return nil, err
	}

	creds, err := provider.Credentials(logging.CloudPlatformReadOnlyScope, logging.LoggingReadScope)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	logadminClient, err := logadmin.NewClient(ctx, projectId, option.WithCredentials(creds))
	if err != nil {
		return nil, err
	}

	var metrics []interface{}
	it := logadminClient.Metrics(ctx)
	for {
		m, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		metric, err := g.MotorRuntime.CreateResource("gcp.project.loggingservice.metric",
			"id", m.ID,
			"projectId", projectId,
			"description", m.Description,
			"filter", m.Filter,
		)
		if err != nil {
			return nil, err
		}
		metrics = append(metrics, metric)
	}
	return metrics, nil
}

func (g *mqlGcpProjectLoggingservice) GetSinks() ([]interface{}, error) {
	provider, err := gcpProvider(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	projectId, err := g.ProjectId()
	if err != nil {
		return nil, err
	}

	creds, err := provider.Credentials(logging.CloudPlatformReadOnlyScope, logging.LoggingReadScope)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	logadminClient, err := logadmin.NewClient(ctx, projectId, option.WithCredentials(creds))
	if err != nil {
		return nil, err
	}

	var sinks []interface{}
	it := logadminClient.Sinks(ctx)
	for {
		s, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		sink, err := g.MotorRuntime.CreateResource("gcp.project.loggingservice.sink",
			"id", s.ID,
			"projectId", projectId,
			"destination", s.Destination,
			"filter", s.Filter,
			"writerIdentity", s.WriterIdentity,
			"includeChildren", s.IncludeChildren,
		)
		if err != nil {
			return nil, err
		}
		sinks = append(sinks, sink)
	}
	return sinks, nil
}

func (g *mqlGcpProjectLoggingserviceMetric) id() (string, error) {
	projectId, err := g.ProjectId()
	if err != nil {
		return "", err
	}
	id, err := g.Id()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/gcp.project.loggingservice.metric/%s", projectId, id), nil
}

func (g *mqlGcpProjectLoggingserviceSink) id() (string, error) {
	projectId, err := g.ProjectId()
	if err != nil {
		return "", err
	}
	id, err := g.Id()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/gcp.project.loggingservice.sink/%s", projectId, id), nil
}

func (g *mqlGcpProjectLoggingserviceBucket) id() (string, error) {
	projectId, err := g.ProjectId()
	if err != nil {
		return "", err
	}
	name, err := g.Name()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/%s", projectId, name), nil
}

func (g *mqlGcpProjectLoggingserviceBucketIndexConfig) id() (string, error) {
	return g.Id()
}
