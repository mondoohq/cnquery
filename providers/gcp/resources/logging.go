// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/gcp/connection"
	"go.mondoo.com/mql/v13/types"

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

type mqlGcpProjectLoggingserviceInternal struct {
	serviceEnabled bool
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

	serviceEnabled, err := g.isServiceEnabled(service_logging)
	if err != nil {
		return nil, err
	}

	svc := res.(*mqlGcpProjectLoggingservice)
	svc.serviceEnabled = serviceEnabled
	if !serviceEnabled {
		log.Debug().Str("service", service_logging).Msg("gcp service is not enabled, skipping")
	}

	return svc, nil
}

// Direct construction (e.g. `gcp.project.loggingservice.sinks`) bypasses
// gcp.project.logging(), leaving projectId empty and serviceEnabled false —
// accessors then short-circuit to empty silently. Delegate to the parent
// project accessor so the resulting instance is fully initialized.
func initGcpProjectLoggingservice(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if _, ok := args["projectId"]; ok {
		return args, nil, nil
	}
	conn, ok := runtime.Connection.(*connection.GcpConnection)
	if !ok {
		return nil, nil, errors.New("invalid connection provided, it is not a GCP connection")
	}
	proj, err := NewResource(runtime, "gcp.project", map[string]*llx.RawData{
		"id": llx.StringData(conn.ResourceID()),
	})
	if err != nil {
		return nil, nil, err
	}
	svc, err := proj.(*mqlGcpProject).logging()
	if err != nil {
		return nil, nil, err
	}
	return nil, svc, nil
}

func (g *mqlGcpProjectLoggingservice) buckets() ([]any, error) {
	if !g.serviceEnabled {
		return nil, nil
	}

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

	var mqlBuckets []any
	req := loggingSvc.Projects.Locations.Buckets.List(fmt.Sprintf("projects/%s/locations/-", projectId))
	if err := req.Pages(ctx, func(page *logging.ListBucketsResponse) error {
		for _, bucket := range page.Buckets {

			var mqlCmekSettingsDict map[string]any
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
					return err
				}
			}

			indexConfigs := make([]any, 0, len(bucket.IndexConfigs))
			for _, cfg := range bucket.IndexConfigs {
				mqlIndexConfig, err := CreateResource(g.MqlRuntime, "gcp.project.loggingservice.bucket.indexConfigs", map[string]*llx.RawData{
					"id":        llx.StringData(fmt.Sprintf("%s/indexConfigs/%s", bucket.Name, cfg.FieldPath)),
					"created":   llx.TimeDataPtr(parseTime(cfg.CreateTime)),
					"fieldPath": llx.StringData(cfg.FieldPath),
					"type":      llx.StringData(cfg.Type),
				})
				if err != nil {
					return err
				}
				indexConfigs = append(indexConfigs, mqlIndexConfig)
			}

			var bucketKmsKeyName string
			if bucket.CmekSettings != nil {
				bucketKmsKeyName = bucket.CmekSettings.KmsKeyName
			}
			mqlBucket, err := CreateResource(g.MqlRuntime, "gcp.project.loggingservice.bucket", map[string]*llx.RawData{
				"projectId":           llx.StringData(projectId),
				"location":            llx.StringData(parseLocationFromPath(bucket.Name)),
				"cmekSettings":        llx.DictData(mqlCmekSettingsDict),
				"created":             llx.TimeDataPtr(parseTime(bucket.CreateTime)),
				"description":         llx.StringData(bucket.Description),
				"indexConfigs":        llx.ArrayData(indexConfigs, types.Resource("gcp.project.loggingservice.bucket.indexConfig")),
				"lifecycleState":      llx.StringData(bucket.LifecycleState),
				"locked":              llx.BoolData(bucket.Locked),
				"name":                llx.StringData(bucket.Name),
				"restrictedFields":    llx.ArrayData(convert.SliceAnyToInterface(bucket.RestrictedFields), types.String),
				"retentionDays":       llx.IntData(bucket.RetentionDays),
				"updated":             llx.TimeDataPtr(parseTime(bucket.UpdateTime)),
				"logAnalyticsEnabled": llx.BoolData(bucket.AnalyticsEnabled),
			})
			if err != nil {
				return err
			}
			mqlBucket.(*mqlGcpProjectLoggingserviceBucket).cacheKmsKeyName = bucketKmsKeyName
			mqlBuckets = append(mqlBuckets, mqlBucket)
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return mqlBuckets, nil
}

func (g *mqlGcpProjectLoggingservice) metrics() ([]any, error) {
	if !g.serviceEnabled {
		return nil, nil
	}

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

	var metrics []any
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

func (g *mqlGcpProjectLoggingserviceMetric) alertPolicies() ([]any, error) {
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

	var res []any
	for _, alertPolicy := range alertPolicies.Data {
		mqlAP := alertPolicy.(*mqlGcpProjectMonitoringServiceAlertPolicy)
		conditions := mqlAP.GetConditions()
		if conditions.Error != nil {
			return nil, conditions.Error
		}
		for _, c := range conditions.Data {
			mqlC := c.(map[string]any)
			var cond map[string]any
			if mqlC["threshold"] != nil {
				cond = mqlC["threshold"].(map[string]any)
			} else if mqlC["absent"] != nil {
				cond = mqlC["absent"].(map[string]any)
			} else if mqlC["matchedLog"] != nil {
				cond = mqlC["matchedLog"].(map[string]any)
			} else if mqlC["monitoringQueryLanguage"] != nil {
				cond = mqlC["monitoringQueryLanguage"].(map[string]any)
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

func parseAlertPolicyConditionFilterMetricName(condition map[string]any) string {
	// Not every alert-policy condition carries a "filter" (e.g. monitoringQueryLanguage
	// conditions only have a "query"), so guard the assertion to avoid a panic.
	filter, ok := condition["filter"].(string)
	if !ok {
		return ""
	}
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

func (g *mqlGcpProjectLoggingservice) sinks() ([]any, error) {
	if !g.serviceEnabled {
		return nil, nil
	}

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

	var sinks []any
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

func (g *mqlGcpProjectLoggingservice) cmekKmsKeyName() (string, error) {
	if !g.serviceEnabled {
		return "", nil
	}

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)

	if g.ProjectId.Error != nil {
		return "", g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	client, err := conn.Client(logging.CloudPlatformReadOnlyScope, logging.LoggingReadScope)
	if err != nil {
		return "", err
	}
	ctx := context.Background()
	loggingSvc, err := logging.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return "", err
	}

	settings, err := loggingSvc.Projects.GetCmekSettings(fmt.Sprintf("projects/%s", projectId)).Context(ctx).Do()
	if err != nil {
		return "", err
	}
	return settings.KmsKeyName, nil
}

func (g *mqlGcpProjectLoggingserviceSink) capturesAllLogs() (bool, error) {
	if g.Filter.Error != nil {
		return false, g.Filter.Error
	}
	return strings.TrimSpace(g.Filter.Data) == "", nil
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

func (g *mqlGcpProjectLoggingserviceMetric) filtersIamChanges() (bool, error) {
	if g.Filter.Error != nil {
		return false, g.Filter.Error
	}
	return strings.Contains(g.Filter.Data, "SetIamPolicy"), nil
}

func (g *mqlGcpProjectLoggingserviceMetric) filtersAuditConfigChanges() (bool, error) {
	if g.Filter.Error != nil {
		return false, g.Filter.Error
	}
	f := g.Filter.Data
	return strings.Contains(f, "SetIamPolicy") && strings.Contains(f, "auditConfigDelta"), nil
}

func (g *mqlGcpProjectLoggingserviceMetric) filtersRouteChanges() (bool, error) {
	if g.Filter.Error != nil {
		return false, g.Filter.Error
	}
	return strings.Contains(g.Filter.Data, "compute.routes."), nil
}

func (g *mqlGcpProjectLoggingserviceMetric) filtersFirewallChanges() (bool, error) {
	if g.Filter.Error != nil {
		return false, g.Filter.Error
	}
	return strings.Contains(g.Filter.Data, "compute.firewalls."), nil
}

func (g *mqlGcpProjectLoggingserviceMetric) filtersSqlInstanceChanges() (bool, error) {
	if g.Filter.Error != nil {
		return false, g.Filter.Error
	}
	return strings.Contains(g.Filter.Data, "cloudsql.instances."), nil
}

func (g *mqlGcpProjectLoggingserviceMetric) filtersStorageIamChanges() (bool, error) {
	if g.Filter.Error != nil {
		return false, g.Filter.Error
	}
	return strings.Contains(g.Filter.Data, "storage.setIamPermissions"), nil
}

func (g *mqlGcpProjectLoggingserviceMetric) filtersProjectOwnershipChanges() (bool, error) {
	if g.Filter.Error != nil {
		return false, g.Filter.Error
	}
	return strings.Contains(g.Filter.Data, "roles/owner"), nil
}

func (g *mqlGcpProjectLoggingserviceMetric) filtersCustomRoleChanges() (bool, error) {
	if g.Filter.Error != nil {
		return false, g.Filter.Error
	}
	f := g.Filter.Data
	return strings.Contains(f, "iam_role") ||
		strings.Contains(f, "google.iam.admin.v1.CreateRole") ||
		strings.Contains(f, "google.iam.admin.v1.DeleteRole") ||
		strings.Contains(f, "google.iam.admin.v1.UpdateRole"), nil
}

func (g *mqlGcpProjectLoggingserviceMetric) filtersVpcNetworkChanges() (bool, error) {
	if g.Filter.Error != nil {
		return false, g.Filter.Error
	}
	return strings.Contains(g.Filter.Data, "compute.networks."), nil
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

type mqlGcpProjectLoggingserviceBucketInternal struct {
	cacheKmsKeyName string
}

func (g *mqlGcpProjectLoggingserviceBucket) kmsKey() (*mqlGcpProjectKmsServiceKeyringCryptokey, error) {
	if g.cacheKmsKeyName == "" {
		g.KmsKey.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	res, err := NewResource(g.MqlRuntime, "gcp.project.kmsService.keyring.cryptokey",
		map[string]*llx.RawData{"resourcePath": llx.StringData(g.cacheKmsKeyName)})
	if err != nil {
		return nil, err
	}
	return res.(*mqlGcpProjectKmsServiceKeyringCryptokey), nil
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

func initGcpProjectLoggingserviceBucket(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 3 {
		return args, nil, nil
	}

	if len(args) == 0 {
		if args == nil {
			args = make(map[string]*llx.RawData)
		}
		if ids := getAssetIdentifier(runtime); ids != nil {
			args["name"] = llx.StringData(ids.name)
			args["location"] = llx.StringData(ids.region)
			args["projectId"] = llx.StringData(ids.project)
		} else {
			return nil, nil, errors.New("no asset identifier found")
		}
	}

	obj, err := CreateResource(runtime, "gcp.project.loggingservice", map[string]*llx.RawData{
		"projectId": args["projectId"],
	})
	if err != nil {
		return nil, nil, err
	}
	svc := obj.(*mqlGcpProjectLoggingservice)
	buckets := svc.GetBuckets()
	if buckets.Error != nil {
		return nil, nil, buckets.Error
	}

	nameVal := args["name"].Value.(string)
	locationVal := ""
	if args["location"] != nil {
		locationVal = args["location"].Value.(string)
	}
	for _, b := range buckets.Data {
		bucket := b.(*mqlGcpProjectLoggingserviceBucket)
		if parseResourceName(bucket.Name.Data) == nameVal && (locationVal == "" || bucket.Location.Data == locationVal) {
			return args, bucket, nil
		}
	}

	return nil, nil, fmt.Errorf("logging bucket %q not found", nameVal)
}

func (g *mqlGcpProjectLoggingserviceBucketIndexConfig) id() (string, error) {
	return g.Id.Data, g.Id.Error
}

func (g *mqlGcpProjectLoggingservice) exclusions() ([]any, error) {
	if !g.serviceEnabled {
		return nil, nil
	}

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

	var mqlExclusions []any
	req := loggingSvc.Projects.Exclusions.List(fmt.Sprintf("projects/%s", projectId))
	if err := req.Pages(ctx, func(page *logging.ListExclusionsResponse) error {
		for _, exclusion := range page.Exclusions {
			mqlExclusion, err := CreateResource(g.MqlRuntime, "gcp.project.loggingservice.exclusion", map[string]*llx.RawData{
				"name":        llx.StringData(parseResourceName(exclusion.Name)),
				"description": llx.StringData(exclusion.Description),
				"filter":      llx.StringData(exclusion.Filter),
				"disabled":    llx.BoolData(exclusion.Disabled),
				"created":     llx.TimeDataPtr(parseTime(exclusion.CreateTime)),
				"updated":     llx.TimeDataPtr(parseTime(exclusion.UpdateTime)),
			})
			if err != nil {
				return err
			}
			mqlExclusions = append(mqlExclusions, mqlExclusion)
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return mqlExclusions, nil
}

func (g *mqlGcpProjectLoggingserviceExclusion) id() (string, error) {
	if g.Name.Error != nil {
		return "", g.Name.Error
	}
	name := g.Name.Data
	return fmt.Sprintf("gcp.project.loggingservice.exclusion/%s", name), nil
}

func (g *mqlGcpProjectLoggingserviceBucket) views() ([]any, error) {
	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)
	client, err := conn.Client(logging.CloudPlatformReadOnlyScope, logging.LoggingReadScope)
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	loggingSvc, err := logging.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}

	if g.Name.Error != nil {
		return nil, g.Name.Error
	}
	bucketName := g.Name.Data

	var res []any
	req := loggingSvc.Projects.Locations.Buckets.Views.List(bucketName)
	if err := req.Pages(ctx, func(page *logging.ListViewsResponse) error {
		for _, view := range page.Views {
			mqlView, err := CreateResource(g.MqlRuntime, "gcp.project.loggingservice.bucket.view", map[string]*llx.RawData{
				"id":          llx.StringData(view.Name),
				"name":        llx.StringData(view.Name),
				"description": llx.StringData(view.Description),
				"filter":      llx.StringData(view.Filter),
				"createTime":  llx.TimeDataPtr(parseTime(view.CreateTime)),
				"updateTime":  llx.TimeDataPtr(parseTime(view.UpdateTime)),
			})
			if err != nil {
				return err
			}
			res = append(res, mqlView)
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return res, nil
}

func (g *mqlGcpProjectLoggingserviceBucketView) id() (string, error) {
	if g.Id.Error != nil {
		return "", g.Id.Error
	}
	return fmt.Sprintf("gcp.project.loggingservice.bucket.view/%s", g.Id.Data), nil
}

// scopedSinks lists Cloud Logging sinks for an organization- or folder-level
// parent and converts them into MQL resources of the given type.
//
// parentResourceName is "organizations/{id}" or "folders/{id}". resourceType
// is the MQL resource name (e.g., "gcp.organization.loggingService.sink"),
// parentFieldName is the field on the MQL sink that holds the parent
// resource name (organizationName / folderName).
func scopedSinks(
	runtime *plugin.Runtime,
	conn *connection.GcpConnection,
	parentResourceName string,
	resourceType string,
	parentFieldName string,
) ([]any, error) {
	client, err := conn.Client(logging.CloudPlatformReadOnlyScope, logging.LoggingReadScope)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	loggingSvc, err := logging.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}

	var sinks []any
	var call interface {
		Pages(context.Context, func(*logging.ListSinksResponse) error) error
	}
	switch {
	case strings.HasPrefix(parentResourceName, "organizations/"):
		call = loggingSvc.Organizations.Sinks.List(parentResourceName)
	case strings.HasPrefix(parentResourceName, "folders/"):
		call = loggingSvc.Folders.Sinks.List(parentResourceName)
	default:
		return nil, fmt.Errorf("unsupported logging parent: %s", parentResourceName)
	}

	err = call.Pages(ctx, func(page *logging.ListSinksResponse) error {
		for _, s := range page.Sinks {
			args := map[string]*llx.RawData{
				parentFieldName:   llx.StringData(parentResourceName),
				"name":            llx.StringData(s.Name),
				"destination":     llx.StringData(s.Destination),
				"filter":          llx.StringData(s.Filter),
				"writerIdentity":  llx.StringData(s.WriterIdentity),
				"includeChildren": llx.BoolData(s.IncludeChildren),
				"disabled":        llx.BoolData(s.Disabled),
				"description":     llx.StringData(s.Description),
				"created":         llx.TimeDataPtr(parseTime(s.CreateTime)),
				"updated":         llx.TimeDataPtr(parseTime(s.UpdateTime)),
			}
			mqlSink, err := CreateResource(runtime, resourceType, args)
			if err != nil {
				return err
			}
			sinks = append(sinks, mqlSink)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return sinks, nil
}

func (g *mqlGcpOrganizationLoggingService) id() (string, error) {
	return g.OrganizationName.Data, g.OrganizationName.Error
}

func (g *mqlGcpOrganizationLoggingServiceSink) id() (string, error) {
	if g.OrganizationName.Error != nil {
		return "", g.OrganizationName.Error
	}
	if g.Name.Error != nil {
		return "", g.Name.Error
	}
	return g.OrganizationName.Data + "/sinks/" + g.Name.Data, nil
}

func (g *mqlGcpOrganization) logging() (*mqlGcpOrganizationLoggingService, error) {
	if g.Id.Error != nil {
		return nil, g.Id.Error
	}
	// g.Id.Data is "organizations/{id}" per initGcpOrganization
	res, err := CreateResource(g.MqlRuntime, "gcp.organization.loggingService", map[string]*llx.RawData{
		"organizationName": llx.StringData(g.Id.Data),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlGcpOrganizationLoggingService), nil
}

func (g *mqlGcpOrganizationLoggingService) sinks() ([]any, error) {
	if g.OrganizationName.Error != nil {
		return nil, g.OrganizationName.Error
	}
	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)
	return scopedSinks(g.MqlRuntime, conn, g.OrganizationName.Data,
		"gcp.organization.loggingService.sink", "organizationName")
}

func (g *mqlGcpFolderLoggingService) id() (string, error) {
	return g.FolderName.Data, g.FolderName.Error
}

func (g *mqlGcpFolderLoggingServiceSink) id() (string, error) {
	if g.FolderName.Error != nil {
		return "", g.FolderName.Error
	}
	if g.Name.Error != nil {
		return "", g.Name.Error
	}
	return g.FolderName.Data + "/sinks/" + g.Name.Data, nil
}

func (g *mqlGcpFolder) logging() (*mqlGcpFolderLoggingService, error) {
	if g.Id.Error != nil {
		return nil, g.Id.Error
	}
	res, err := CreateResource(g.MqlRuntime, "gcp.folder.loggingService", map[string]*llx.RawData{
		"folderName": llx.StringData(folderResourceName(g.Id.Data)),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlGcpFolderLoggingService), nil
}

func (g *mqlGcpFolderLoggingService) sinks() ([]any, error) {
	if g.FolderName.Error != nil {
		return nil, g.FolderName.Error
	}
	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)
	return scopedSinks(g.MqlRuntime, conn, g.FolderName.Data,
		"gcp.folder.loggingService.sink", "folderName")
}
