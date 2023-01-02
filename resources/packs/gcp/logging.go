package gcp

import (
	"context"
	"fmt"

	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/resources/packs/core"
	"google.golang.org/api/logging/v2"
	"google.golang.org/api/option"
)

func (g *mqlGcpProjectLoggingservices) id() (string, error) {
	return "gcp.project.loggingservices", nil
}

func (g *mqlGcpProject) GetLogging() (interface{}, error) {
	projectId, err := g.Id()
	if err != nil {
		return nil, err
	}

	return g.MotorRuntime.CreateResource("gcp.project.loggingservices",
		"projectId", projectId,
	)
}

func (g *mqlGcpProjectLoggingservices) GetBuckets() ([]interface{}, error) {
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

		var mqlCmekSettings resources.ResourceType
		if bucket.CmekSettings != nil {
			mqlCmekSettings, err = g.MotorRuntime.CreateResource("gcp.project.loggingservices.bucket.cmekSettings",
				"id", fmt.Sprintf("%s/cmekSettings", bucket.Name),
				"kmsKeyName", bucket.CmekSettings.KmsKeyName,
				"kmsKeyVersionName", bucket.CmekSettings.KmsKeyVersionName,
				"name", bucket.CmekSettings.Name,
				"serviceAccountId", bucket.CmekSettings.ServiceAccountId,
			)
			if err != nil {
				return nil, err
			}
		}

		indexConfigs := make([]interface{}, 0, len(bucket.IndexConfigs))
		for i, cfg := range bucket.IndexConfigs {
			mqlIndexConfig, err := g.MotorRuntime.CreateResource("gcp.project.loggingservices.bucket.indexConfigs",
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

		mqlBucket, err := g.MotorRuntime.CreateResource("gcp.project.loggingservices.bucket",
			"id", bucket.Name,
			"cmekSettings", mqlCmekSettings,
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

func (g *mqlGcpProjectLoggingservicesBucket) id() (string, error) {
	return g.Id()
}

func (g *mqlGcpProjectLoggingservicesBucketCmekSettings) id() (string, error) {
	return g.Id()
}

func (g *mqlGcpProjectLoggingservicesBucketIndexConfig) id() (string, error) {
	return g.Id()
}
