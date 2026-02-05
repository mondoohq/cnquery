// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/aws/aws-sdk-go-v2/service/kms/types"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v12/llx"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/cnquery/v12/providers/aws/connection"
)

const (
	kmsKeyArnPattern = "arn:aws:kms:%s:%s:key/%s"
)

// NormalizeKmsKeyRef normalizes a KMS key reference to ARN format.
// KeyID supports multiple formats: Key ID (UUID), Key ARN, Alias Name, Alias ARN
func NormalizeKmsKeyRef(s, region, accountId string) (arn.ARN, error) {
	// Try ARN parse first (common case)
	parsed, arnErr := arn.Parse(s)
	if arnErr == nil {
		return parsed, nil
	}

	// Fallback: check if it's a key ID (UUID format: 36 chars with hyphens)
	// Example: 7a4eb143-c07b-4e24-b0b7-f3abfdbbb2c2
	// This is an edge case where Secrets Manager returns just the key ID
	if len(s) == 36 && s[8] == '-' && s[13] == '-' && s[18] == '-' && s[23] == '-' {
		return arn.ARN{
			Partition: "aws",
			Service:   "kms",
			Region:    region,
			AccountID: accountId,
			Resource:  "key/" + s,
		}, nil
	}

	// Todo add alias format handling here for Alias name and Alias ARN

	// If both checks fail, propagate the ARN parse error for better diagnostics
	return arn.ARN{}, fmt.Errorf("invalid KMS key reference %q: %w", s, arnErr)
}

func (a *mqlAwsKms) id() (string, error) {
	return "aws.kms", nil
}

func (a *mqlAwsKms) keys() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getKeys(conn), 5)
	poolOfJobs.Run()

	// check for errors
	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	// get all the results
	for i := range poolOfJobs.Jobs {
		res = append(res, poolOfJobs.Jobs[i].Result.([]any)...)
	}
	return res, nil
}

func (a *mqlAwsKms) getKeys(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("kms>getKeys>calling aws with region %s", region)

			svc := conn.Kms(region)
			res := []any{}

			keys := make([]types.KeyListEntry, 0)
			params := &kms.ListKeysInput{}
			paginator := kms.NewListKeysPaginator(svc, params, func(o *kms.ListKeysPaginatorOptions) {
				o.Limit = 100
			})
			for paginator.HasMorePages() {
				output, err := paginator.NextPage(context.TODO())
				if err != nil {
					return nil, err
				}
				keys = append(keys, output.Keys...)
			}

			for _, key := range keys {
				mqlKey, err := CreateResource(a.MqlRuntime, "aws.kms.key",
					map[string]*llx.RawData{
						"id":     llx.StringDataPtr(key.KeyId),
						"arn":    llx.StringDataPtr(key.KeyArn),
						"region": llx.StringData(region),
					})
				if err != nil {
					return nil, err
				}
				res = append(res, mqlKey)
			}

			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func (a *mqlAwsKmsKey) metadata() (any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	key := a.Arn.Data

	svc := conn.Kms(a.Region.Data)
	ctx := context.Background()

	keyMetadata, err := svc.DescribeKey(ctx, &kms.DescribeKeyInput{KeyId: &key})
	if err != nil {
		return nil, err
	}
	return convert.JsonToDict(keyMetadata.KeyMetadata)
}

func (a *mqlAwsKmsKey) keyRotationEnabled() (bool, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	keyId := a.Id.Data

	svc := conn.Kms(a.Region.Data)
	ctx := context.Background()

	key, err := svc.GetKeyRotationStatus(ctx, &kms.GetKeyRotationStatusInput{KeyId: &keyId})
	if err != nil {
		return false, err
	}
	return key.KeyRotationEnabled, nil
}

func (a *mqlAwsKmsKey) tags() (map[string]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	keyArn := a.Arn.Data

	svc := conn.Kms(a.Region.Data)
	ctx := context.Background()

	tags, err := svc.ListResourceTags(ctx, &kms.ListResourceTagsInput{KeyId: &keyArn})
	if err != nil {
		return nil, err
	}

	res := map[string]any{}
	for i := range tags.Tags {
		tag := tags.Tags[i]
		res[convert.ToValue(tag.TagKey)] = convert.ToValue(tag.TagValue)
	}

	return res, nil
}

func (a *mqlAwsKmsKey) aliases() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	keyArn := a.Arn.Data

	svc := conn.Kms(a.Region.Data)
	ctx := context.Background()

	resp, err := svc.ListAliases(ctx, &kms.ListAliasesInput{KeyId: &keyArn})
	if err != nil {
		return nil, err
	}

	aliases := []any{}
	for _, a := range resp.Aliases {
		aliases = append(aliases, *a.AliasName)
	}

	return aliases, nil
}

func (a *mqlAwsKmsKey) keyState() (string, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	keyArn := a.Arn.Data

	svc := conn.Kms(a.Region.Data)
	ctx := context.Background()

	keyMetadata, err := svc.DescribeKey(ctx, &kms.DescribeKeyInput{KeyId: &keyArn})
	if err != nil {
		return "", err
	}
	return string(keyMetadata.KeyMetadata.KeyState), nil
}

func (a *mqlAwsKmsKey) id() (string, error) {
	return a.Arn.Data, nil
}

func initAwsKmsKey(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}

	if len(args) == 0 {
		if ids := getAssetIdentifier(runtime); ids != nil {
			args["name"] = llx.StringData(ids.name)
			args["arn"] = llx.StringData(ids.arn)
		}
	}

	r := args["arn"]
	if r == nil {
		return nil, nil, errors.New("arn required to fetch aws kms key")
	}
	a, ok := r.Value.(string)
	if !ok {
		return nil, nil, errors.New("invalid arn")
	}

	conn := runtime.Connection.(*connection.AwsConnection)

	// Get region from args if provided (needed when input is just a key ID, not full ARN)
	var region string
	if regionArg := args["region"]; regionArg != nil {
		if r, ok := regionArg.Value.(string); ok {
			region = r
		}
	}

	// KMS Keys API calls use KeyID = support multiple formats aliases, UUID(ID) or arn format we would not need to normalize here but there seems to be some edge cases
	// for Example Secrets Manager can returns just the key ID instead of full ARN in KmsKeyId for EventBride connection secrets
	// Current code only provides arn to kms function
	arnVal, err := NormalizeKmsKeyRef(a, region, conn.AccountId())
	if err != nil {
		return nil, nil, err
	}

	// Use normalized ARN for cache lookup and resource creation
	normalizedArn := arnVal.String()
	args["arn"] = llx.StringData(normalizedArn)
	args["region"] = llx.StringData(arnVal.Region)

	if arnVal.AccountID != conn.AccountId() {
		// Cross-account key not yet supported
		// Todo isCrossAccount and accessible check to handle policy limitations in same and another account
		log.Error().Str("arn", normalizedArn).Str("keyAccount", arnVal.AccountID).Str("currentAccount", conn.AccountId()).Msg("cross-account KMS keys are not supported yet")
		return nil, nil, fmt.Errorf("cross-account KMS keys are not supported yet: %s (belongs to account %s)", normalizedArn, arnVal.AccountID)
	}

	obj, err := CreateResource(runtime, ResourceAwsKms, map[string]*llx.RawData{})
	if err != nil {
		return nil, nil, err
	}
	kms := obj.(*mqlAwsKms)

	rawResources := kms.GetKeys()
	if rawResources.Error != nil {
		return nil, nil, rawResources.Error
	}

	// Extract key ID from input for cache matching (handles both ARN and UUID inputs)
	inputKeyId := extractKmsKeyId(a)

	for _, rawResource := range rawResources.Data {
		key := rawResource.(*mqlAwsKmsKey)
		// Match by ARN or by key ID (for UUID-only inputs)
		if key.Arn.Data == normalizedArn || key.Id.Data == inputKeyId {
			// Use actual values from cache
			args["arn"] = llx.StringData(key.Arn.Data)
			args["region"] = llx.StringData(key.Region.Data)
			args["id"] = llx.StringData(key.Id.Data)
			return args, key, nil
		}
	}

	return nil, nil, errors.New("key not found")
}

// extractKmsKeyId extracts the key ID from an ARN resource string like "key/uuid"
func extractKmsKeyId(resource string) string {
	if len(resource) > 4 && resource[:4] == "key/" {
		return resource[4:]
	}
	return resource
}
