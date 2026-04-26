// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/gcp/connection"
	"go.mondoo.com/mql/v13/types"
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

type mqlGcpProjectStorageServiceInternal struct {
	serviceEnabled bool
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

	serviceEnabled, err := g.isServiceEnabled(service_storage)
	if err != nil {
		return nil, err
	}

	svc := res.(*mqlGcpProjectStorageService)
	svc.serviceEnabled = serviceEnabled
	if !serviceEnabled {
		log.Debug().Str("service", service_storage).Msg("gcp service is not enabled, skipping")
	}

	return svc, nil
}

func (g *mqlGcpProjectStorageService) buckets() ([]any, error) {
	if !g.serviceEnabled {
		return nil, nil
	}

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

	res := make([]any, 0, len(buckets.Items))
	for i := range buckets.Items {
		mqlBucket, err := mqlBucketFromAPI(g.MqlRuntime, projectId, buckets.Items[i])
		if err != nil {
			return nil, err
		}
		res = append(res, mqlBucket)
	}
	return res, nil
}

// mqlBucketFromAPI converts a *storage.Bucket into a fully-populated mql
// resource. Used by buckets() during a list and by init when resolving a
// single bucket by name.
func mqlBucketFromAPI(runtime *plugin.Runtime, projectId string, bucket *storage.Bucket) (*mqlGcpProjectStorageServiceBucket, error) {
	created := parseTime(bucket.TimeCreated)
	updated := parseTime(bucket.Updated)

	iamConfigurationDict, err := convert.JsonToDict(bucket.IamConfiguration)
	if err != nil {
		return nil, err
	}
	retentionPolicy, err := convert.JsonToDict(bucket.RetentionPolicy)
	if err != nil {
		return nil, err
	}
	enc, err := convert.JsonToDict(bucket.Encryption)
	if err != nil {
		return nil, err
	}

	publicAccessPrevention := ""
	var uniformBucketLevelAccess map[string]any
	if bucket.IamConfiguration != nil {
		publicAccessPrevention = bucket.IamConfiguration.PublicAccessPrevention
		uniformBucketLevelAccess, err = convert.JsonToDict(bucket.IamConfiguration.UniformBucketLevelAccess)
		if err != nil {
			return nil, err
		}
	}

	softDeletePolicy, err := convert.JsonToDict(bucket.SoftDeletePolicy)
	if err != nil {
		return nil, err
	}

	objectRetentionMode := ""
	if bucket.ObjectRetention != nil {
		objectRetentionMode = bucket.ObjectRetention.Mode
	}

	autoclass, err := convert.JsonToDict(bucket.Autoclass)
	if err != nil {
		return nil, err
	}

	mqlInstance, err := CreateResource(runtime, "gcp.project.storageService.bucket", map[string]*llx.RawData{
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
		"lifecycle": llx.ArrayData(
			storageLifecycleRulesToArrayInterface(runtime, bucket.Id, bucket.Lifecycle),
			types.Resource("gcp.project.storageService.bucket.lifecycleRule"),
		),
		"defaultEventBasedHold":    llx.BoolData(bucket.DefaultEventBasedHold),
		"rpo":                      llx.StringData(bucket.Rpo),
		"satisfiesPZS":             llx.BoolData(bucket.SatisfiesPZS),
		"versioningEnabled":        llx.BoolData(bucket.Versioning != nil && bucket.Versioning.Enabled),
		"publicAccessPrevention":   llx.StringData(publicAccessPrevention),
		"metageneration":           llx.IntData(bucket.Metageneration),
		"uniformBucketLevelAccess": llx.DictData(uniformBucketLevelAccess),
		"softDeletePolicy":         llx.DictData(softDeletePolicy),
		"objectRetentionMode":      llx.StringData(objectRetentionMode),
		"autoclass":                llx.DictData(autoclass),
	})
	if err != nil {
		return nil, err
	}
	mqlBucket := mqlInstance.(*mqlGcpProjectStorageServiceBucket)
	if bucket.Encryption != nil {
		mqlBucket.cacheDefaultKmsKeyName = bucket.Encryption.DefaultKmsKeyName
	}
	return mqlBucket, nil
}

type mqlGcpProjectStorageServiceBucketInternal struct {
	cacheDefaultKmsKeyName string
}

func (g *mqlGcpProjectStorageServiceBucket) defaultKmsKey() (*mqlGcpProjectKmsServiceKeyringCryptokey, error) {
	if g.cacheDefaultKmsKeyName == "" {
		g.DefaultKmsKey.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	res, err := NewResource(g.MqlRuntime, "gcp.project.kmsService.keyring.cryptokey",
		map[string]*llx.RawData{"resourcePath": llx.StringData(g.cacheDefaultKmsKeyName)})
	if err != nil {
		return nil, err
	}
	return res.(*mqlGcpProjectKmsServiceKeyringCryptokey), nil
}

func storageLifecycleRulesToArrayInterface(runtime *plugin.Runtime, bucketId string, lifecycle *storage.BucketLifecycle) (list []any) {
	if lifecycle == nil {
		return
	}
	for i, rule := range lifecycle.Rule {
		if rule == nil {
			continue
		}

		var (
			action      plugin.Resource
			condition   plugin.Resource
			err         error
			skip        = true
			ruleRawData = map[string]*llx.RawData{}
		)

		// create rule action resource
		if rule.Action != nil {
			action, err = CreateResource(runtime, "gcp.project.storageService.bucket.lifecycleRuleAction", map[string]*llx.RawData{
				"__id": llx.StringData(
					fmt.Sprintf("gcp.project.storageService.bucket.lifecycleRuleAction/%s/%d", bucketId, i),
				),
				"storageClass": llx.StringData(rule.Action.StorageClass),
				"type":         llx.StringData(rule.Action.Type),
			})
			if err != nil {
				continue
			}
			ruleRawData["action"] = llx.ResourceData(action, action.MqlName())
			skip = false
		}

		// create rule condition resource
		if rule.Condition != nil {
			condition, err = CreateResource(runtime, "gcp.project.storageService.bucket.lifecycleRuleCondition", map[string]*llx.RawData{
				"__id": llx.StringData(
					fmt.Sprintf("gcp.project.storageService.bucket.lifecycleRuleCondition/%s/%d", bucketId, i),
				),
				"age":                     llx.IntDataPtr(rule.Condition.Age),
				"daysSinceCustomTime":     llx.IntData(rule.Condition.DaysSinceCustomTime),
				"daysSinceNoncurrentTime": llx.IntData(rule.Condition.DaysSinceNoncurrentTime),
				"numNewerVersions":        llx.IntData(rule.Condition.NumNewerVersions),
				"isLive":                  llx.BoolDataPtr(rule.Condition.IsLive),
				"createdBefore":           llx.StringData(rule.Condition.CreatedBefore),
				"customTimeBefore":        llx.StringData(rule.Condition.CustomTimeBefore),
				"matchesPattern":          llx.StringData(rule.Condition.MatchesPattern),
				"noncurrentTimeBefore":    llx.StringData(rule.Condition.NoncurrentTimeBefore),
				"matchesPrefix":           llx.ArrayData(convert.SliceAnyToInterface(rule.Condition.MatchesPrefix), types.String),
				"matchesStorageClass":     llx.ArrayData(convert.SliceAnyToInterface(rule.Condition.MatchesStorageClass), types.String),
				"matchesSuffix":           llx.ArrayData(convert.SliceAnyToInterface(rule.Condition.MatchesSuffix), types.String),
			})
			if err != nil {
				continue
			}
			ruleRawData["condition"] = llx.ResourceData(condition, condition.MqlName())
			skip = false
		}

		// if the rule doesn't have an action or a condition, skip it
		if skip {
			continue
		}

		// add the rule id
		ruleRawData["__id"] = llx.StringData(
			fmt.Sprintf("gcp.project.storageService.bucket.lifecycleRule/%s/%d", bucketId, i),
		)

		r, err := CreateResource(runtime, "gcp.project.storageService.bucket.lifecycleRule", ruleRawData)
		if err != nil {
			continue
		}
		list = append(list, r)
	}

	return
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

	// If no args are set, try reading them from the platform ID. The asset
	// identifier path supplies all three (name + projectId + location).
	if len(args) == 0 {
		ids := getAssetIdentifier(runtime)
		if ids == nil {
			return nil, nil, errors.New("no asset identifier found")
		}
		args["name"] = llx.StringData(ids.name)
		args["projectId"] = llx.StringData(ids.project)
		args["location"] = llx.StringData(ids.region)
	}

	// Resolve the bucket directly via Buckets.Get(name). Bucket names are
	// globally unique in GCS, so a single Get returns everything we need —
	// including projectNumber and location — even when the caller only
	// supplied "name" (e.g. cross-resource references like a Datastream
	// connection profile pointing at a GCS bucket).
	nameRaw, ok := args["name"]
	if !ok || nameRaw == nil {
		return args, nil, nil
	}
	name, ok := nameRaw.Value.(string)
	if !ok || name == "" {
		return args, nil, nil
	}

	conn, ok := runtime.Connection.(*connection.GcpConnection)
	if !ok {
		return nil, nil, errors.New("invalid connection provided, it is not a GCP connection")
	}
	client, err := conn.Client(cloudresourcemanager.CloudPlatformReadOnlyScope, iam.CloudPlatformScope, storage.CloudPlatformScope)
	if err != nil {
		return nil, nil, err
	}
	ctx := context.Background()
	storageSvc, err := storage.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, nil, err
	}
	bucket, err := storageSvc.Buckets.Get(name).Do()
	if err != nil {
		return nil, nil, err
	}

	// Derive projectId from caller args when supplied; otherwise fall back to
	// the connection's project. Bucket.ProjectNumber is numeric (not the
	// human ID) so we can't use it directly as projectId.
	projectId := conn.ResourceID()
	if pidRaw, ok := args["projectId"]; ok && pidRaw != nil {
		if s, ok := pidRaw.Value.(string); ok && s != "" {
			projectId = s
		}
	}
	mqlBucket, err := mqlBucketFromAPI(runtime, projectId, bucket)
	if err != nil {
		return nil, nil, err
	}
	return args, mqlBucket, nil
}

func (g *mqlGcpProjectStorageServiceBucket) iamPolicy() ([]any, error) {
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

	res := []any{}
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
