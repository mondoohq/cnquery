// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"fmt"
	"strings"

	"go.mondoo.com/cnquery/v10/llx"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v10/providers/gcp/connection"
	"go.mondoo.com/cnquery/v10/types"

	"cloud.google.com/go/logging/logadmin"
	"google.golang.org/api/iterator"
	"google.golang.org/api/logging/v2"
	"google.golang.org/api/option"
)

func (g *mqlGcpProjectLoggingservice) id() (string, error) {
	if g.ProjectId.Error != nil {
		return "", g.ProjectId.Error
	}
	projectId := g.ProjectId.Data
	return fmt.Sprintf("%s/gcp.project.loggingservice", projectId), nil
}

func (g *mqlGcpProject) logging() (*mqlGcpProjectLoggingservice, error) {
	if g.Id.Error != nil {
		return nil, g.Id.Error
	}
	projectId := g.Id.Data

	res, err := CreateResource(g.MqlRuntime, "gcp.project.loggingservice", map[string]*llx.RawData{
		"projectId": llx.StringData(projectId),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlGcpProjectLoggingservice), nil
}

func (g *mqlGcpProjectLoggingservice) buckets() ([]interface{}, error) {
	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)

	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	client, err := conn.Client(logging.CloudPlatformReadOnlyScope, logging.LoggingReadScope)
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
			mqlCmekSettingsDict, err = convert.JsonToDict(mqlCmekSettings{
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
			mqlIndexConfig, err := CreateResource(g.MqlRuntime, "gcp.project.loggingservice.bucket.indexConfigs", map[string]*llx.RawData{
				"id":        llx.StringData(fmt.Sprintf("%s/indexConfigs/%d", bucket.Name, i)),
				"created":   llx.TimeDataPtr(parseTime(cfg.CreateTime)),
				"fieldPath": llx.StringData(cfg.FieldPath),
				"type":      llx.StringData(cfg.Type),
			})
			if err != nil {
				return nil, err
			}
			indexConfigs = append(indexConfigs, mqlIndexConfig)
		}

		mqlBucket, err := CreateResource(g.MqlRuntime, "gcp.project.loggingservice.bucket", map[string]*llx.RawData{
			"projectId":        llx.StringData(projectId),
			"cmekSettings":     llx.DictData(mqlCmekSettingsDict),
			"created":          llx.TimeDataPtr(parseTime(bucket.CreateTime)),
			"description":      llx.StringData(bucket.Description),
			"indexConfigs":     llx.ArrayData(indexConfigs, types.Resource("gcp.project.loggingservice.bucket.indexConfig")),
			"lifecycleState":   llx.StringData(bucket.LifecycleState),
			"locked":           llx.BoolData(bucket.Locked),
			"name":             llx.StringData(bucket.Name),
			"restrictedFields": llx.ArrayData(convert.SliceAnyToInterface(bucket.RestrictedFields), types.String),
			"retentionDays":    llx.IntData(bucket.RetentionDays),
			"updated":          llx.TimeDataPtr(parseTime(bucket.UpdateTime)),
		})
		if err != nil {
			return nil, err
		}
		mqlBuckets = append(mqlBuckets, mqlBucket)
	}
	return mqlBuckets, nil
}

func (g *mqlGcpProjectLoggingservice) metrics() ([]interface{}, error) {
	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)

	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	creds, err := conn.Credentials(logging.CloudPlatformReadOnlyScope, logging.LoggingReadScope)
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
		metric, err := CreateResource(g.MqlRuntime, "gcp.project.loggingservice.metric", map[string]*llx.RawData{
			"id":          llx.StringData(m.ID),
			"projectId":   llx.StringData(projectId),
			"description": llx.StringData(m.Description),
			"filter":      llx.StringData(m.Filter),
		})
		if err != nil {
			return nil, err
		}
		metrics = append(metrics, metric)
	}
	return metrics, nil
}

func (g *mqlGcpProjectLoggingserviceMetric) alertPolicies() ([]interface{}, error) {
	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	if g.Id.Error != nil {
		return nil, g.Id.Error
	}
	id := g.Id.Data

	// Find alert policies for projectId
	obj, err := CreateResource(g.MqlRuntime, "gcp.project.monitoringService", map[string]*llx.RawData{
		"projectId": llx.StringData(projectId),
	})
	if err != nil {
		return nil, err
	}
	gcpMonitoring := obj.(*mqlGcpProjectMonitoringService)
	alertPolicies := gcpMonitoring.GetAlertPolicies()
	if alertPolicies.Error != nil {
		return nil, alertPolicies.Error
	}

	var res []interface{}
	for _, alertPolicy := range alertPolicies.Data {
		mqlAP := alertPolicy.(*mqlGcpProjectMonitoringServiceAlertPolicy)
		conditions := mqlAP.GetConditions()
		if conditions.Error != nil {
			return nil, conditions.Error
		}
		for _, c := range conditions.Data {
			mqlC := c.(map[string]interface{})
			var cond map[string]interface{}
			if mqlC["threshold"] != nil {
				cond = mqlC["threshold"].(map[string]interface{})
			} else if mqlC["absent"] != nil {
				cond = mqlC["absent"].(map[string]interface{})
			} else if mqlC["matchedLog"] != nil {
				cond = mqlC["matchedLog"].(map[string]interface{})
			} else if mqlC["monitoringQueryLanguage"] != nil {
				cond = mqlC["monitoringQueryLanguage"].(map[string]interface{})
			} else {
				continue
			}

			if parseAlertPolicyConditionFilterMetricName(cond) == id {
				res = append(res, alertPolicy)
			}
		}
	}
	return res, nil
}

func parseAlertPolicyConditionFilterMetricName(condition map[string]interface{}) string {
	filter := condition["filter"].(string)
	// The filter is composed of multiple statements split by AND or OR and spaces in between
	parts := strings.Split(filter, " ")
	for _, p := range parts {
		// If the statement starts with metric.type="logging.googleapis.com/user/ then we are interested in it
		if strings.HasPrefix(p, "metric.type=\"logging.googleapis.com/user/") {
			// The filter looks like this: metric.type=\"logging.googleapis.com/user/log-metric-filter-and-alerts-exist-for-project-ownership-assignments-changes\"
			// We are interested in the user part of that string
			return strings.TrimSuffix(strings.TrimPrefix(p, "metric.type=\"logging.googleapis.com/user/"), "\"")
		}
	}
	return ""
}

func (g *mqlGcpProjectLoggingservice) sinks() ([]interface{}, error) {
	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)

	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	creds, err := conn.Credentials(logging.CloudPlatformReadOnlyScope, logging.LoggingReadScope)
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
		args := map[string]*llx.RawData{
			"id":              llx.StringData(s.ID),
			"projectId":       llx.StringData(projectId),
			"destination":     llx.StringData(s.Destination),
			"filter":          llx.StringData(s.Filter),
			"writerIdentity":  llx.StringData(s.WriterIdentity),
			"includeChildren": llx.BoolData(s.IncludeChildren),
		}
		if !strings.HasPrefix(s.Destination, "storage.googleapis.com/") {
			args["storageBucket"] = llx.NilData
		}
		sink, err := CreateResource(g.MqlRuntime, "gcp.project.loggingservice.sink", args)
		if err != nil {
			return nil, err
		}

		sinks = append(sinks, sink)
	}
	return sinks, nil
}

func (g *mqlGcpProjectLoggingserviceSink) storageBucket() (*mqlGcpProjectStorageServiceBucket, error) {
	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	dest := g.GetDestination()
	if dest.Error != nil {
		return nil, dest.Error
	}
	if strings.HasPrefix(dest.Data, "storage.googleapis.com/") {
		obj, err := CreateResource(g.MqlRuntime, "gcp.project.storageService", map[string]*llx.RawData{
			"projectId": llx.StringData(projectId),
		})
		if err != nil {
			return nil, err
		}
		gcpStorage := obj.(*mqlGcpProjectStorageService)
		buckets := gcpStorage.GetBuckets()
		if buckets.Error != nil {
			return nil, buckets.Error
		}

		targetBucketName := strings.TrimPrefix(dest.Data, "storage.googleapis.com/")
		for _, bucket := range buckets.Data {
			bucketName := bucket.(*mqlGcpProjectStorageServiceBucket).GetName()
			if bucketName.Error != nil {
				return nil, bucketName.Error
			}

			if bucketName.Data == targetBucketName {
				return bucket.(*mqlGcpProjectStorageServiceBucket), nil
			}
		}
	}

	g.StorageBucket.State = plugin.StateIsSet | plugin.StateIsNull
	return nil, nil
}

func (g *mqlGcpProjectLoggingserviceMetric) id() (string, error) {
	if g.ProjectId.Error != nil {
		return "", g.ProjectId.Error
	}
	projectId := g.ProjectId.Data
	if g.Id.Error != nil {
		return "", g.Id.Error
	}
	id := g.Id.Data
	return fmt.Sprintf("%s/gcp.project.loggingservice.metric/%s", projectId, id), nil
}

func (g *mqlGcpProjectLoggingserviceSink) id() (string, error) {
	if g.ProjectId.Error != nil {
		return "", g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	if g.Id.Error != nil {
		return "", g.Id.Error
	}
	id := g.Id.Data
	return fmt.Sprintf("%s/gcp.project.loggingservice.sink/%s", projectId, id), nil
}

func (g *mqlGcpProjectLoggingserviceBucket) id() (string, error) {
	if g.ProjectId.Error != nil {
		return "", g.ProjectId.Error
	}
	projectId := g.ProjectId.Data
	if g.Name.Error != nil {
		return "", g.Name.Error
	}
	name := g.Name.Data
	return fmt.Sprintf("%s/%s", projectId, name), nil
}

func (g *mqlGcpProjectLoggingserviceBucketIndexConfig) id() (string, error) {
	return g.Id.Data, g.Id.Error
}
