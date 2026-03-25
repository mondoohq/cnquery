// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/aws/aws-sdk-go-v2/service/kms/types"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/mql/v13/providers/aws/connection"

	mqlTypes "go.mondoo.com/mql/v13/types"
)

const (
	kmsKeyArnPattern = "arn:aws:kms:%s:%s:key/%s"
)

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
	md, err := a.getKeyMetadata()
	if err != nil {
		return nil, err
	}
	return convert.JsonToDict(md)
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

	res := map[string]any{}
	paginator := kms.NewListResourceTagsPaginator(svc, &kms.ListResourceTagsInput{KeyId: &keyArn})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for i := range page.Tags {
			tag := page.Tags[i]
			res[convert.ToValue(tag.TagKey)] = convert.ToValue(tag.TagValue)
		}
	}

	return res, nil
}

func (a *mqlAwsKmsKey) aliases() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	keyArn := a.Arn.Data

	svc := conn.Kms(a.Region.Data)
	ctx := context.Background()

	aliases := []any{}
	paginator := kms.NewListAliasesPaginator(svc, &kms.ListAliasesInput{KeyId: &keyArn})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, alias := range page.Aliases {
			aliases = append(aliases, *alias.AliasName)
		}
	}

	return aliases, nil
}

func (a *mqlAwsKmsKey) keyState() (string, error) {
	md, err := a.getKeyMetadata()
	if err != nil {
		return "", err
	}
	return string(md.KeyState), nil
}

type mqlAwsKmsKeyInternal struct {
	cachedKeyMetadata *types.KeyMetadata
	metadataLock      sync.Mutex
}

func (a *mqlAwsKmsKey) getKeyMetadata() (*types.KeyMetadata, error) {
	if a.cachedKeyMetadata != nil {
		return a.cachedKeyMetadata, nil
	}
	a.metadataLock.Lock()
	defer a.metadataLock.Unlock()
	if a.cachedKeyMetadata != nil {
		return a.cachedKeyMetadata, nil
	}

	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	keyArn := a.Arn.Data

	svc := conn.Kms(a.Region.Data)
	ctx := context.Background()

	resp, err := svc.DescribeKey(ctx, &kms.DescribeKeyInput{KeyId: &keyArn})
	if err != nil {
		return nil, err
	}
	a.cachedKeyMetadata = resp.KeyMetadata
	return a.cachedKeyMetadata, nil
}

func (a *mqlAwsKmsKey) createdAt() (*time.Time, error) {
	md, err := a.getKeyMetadata()
	if err != nil {
		return nil, err
	}
	return md.CreationDate, nil
}

func (a *mqlAwsKmsKey) deletedAt() (*time.Time, error) {
	md, err := a.getKeyMetadata()
	if err != nil {
		return nil, err
	}
	return md.DeletionDate, nil
}

func (a *mqlAwsKmsKey) enabled() (bool, error) {
	md, err := a.getKeyMetadata()
	if err != nil {
		return false, err
	}
	return md.Enabled, nil
}

func (a *mqlAwsKmsKey) description() (string, error) {
	md, err := a.getKeyMetadata()
	if err != nil {
		return "", err
	}
	return convert.ToValue(md.Description), nil
}

func (a *mqlAwsKmsKey) keyManager() (string, error) {
	md, err := a.getKeyMetadata()
	if err != nil {
		return "", err
	}
	return string(md.KeyManager), nil
}

func (a *mqlAwsKmsKey) policy() (string, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	keyArn := a.Arn.Data

	svc := conn.Kms(a.Region.Data)
	ctx := context.Background()

	resp, err := svc.GetKeyPolicy(ctx, &kms.GetKeyPolicyInput{KeyId: &keyArn})
	if err != nil {
		if Is400AccessDeniedError(err) {
			log.Debug().Str("key", keyArn).Msg("access denied when retrieving KMS key policy")
			return "", nil
		}
		return "", err
	}
	return convert.ToValue(resp.Policy), nil
}

func (a *mqlAwsKmsKey) id() (string, error) {
	return a.Arn.Data, nil
}

func (a *mqlAwsKmsKey) grants() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	keyArn := a.Arn.Data

	svc := conn.Kms(a.Region.Data)
	ctx := context.Background()

	res := []any{}
	params := &kms.ListGrantsInput{KeyId: &keyArn}
	paginator := kms.NewListGrantsPaginator(svc, params)
	for paginator.HasMorePages() {
		grantsResp, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, grant := range grantsResp.Grants {
			operations := make([]any, len(grant.Operations))
			for i, op := range grant.Operations {
				operations[i] = string(op)
			}
			mqlGrant, err := CreateResource(a.MqlRuntime, "aws.kms.grant",
				map[string]*llx.RawData{
					"__id":              llx.StringData(keyArn + "/grant/" + convert.ToValue(grant.GrantId)),
					"grantId":           llx.StringDataPtr(grant.GrantId),
					"keyArn":            llx.StringData(keyArn),
					"granteePrincipal":  llx.StringDataPtr(grant.GranteePrincipal),
					"retiringPrincipal": llx.StringDataPtr(grant.RetiringPrincipal),
					"issuingAccount":    llx.StringDataPtr(grant.IssuingAccount),
					"operations":        llx.ArrayData(operations, mqlTypes.String),
					"createdAt":         llx.TimeDataPtr(grant.CreationDate),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlGrant)
		}
	}
	return res, nil
}

func (a *mqlAwsKmsGrant) id() (string, error) {
	return a.KeyArn.Data + "/grant/" + a.GrantId.Data, nil
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
	arnVal, err := arn.Parse(a)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid ARN %q: %w", a, err)
	}
	if arnVal.AccountID != runtime.Connection.(*connection.AwsConnection).AccountId() {
		// Cross-account key: we can't fetch details, but we should still return the ARN
		// so security tools can see which KMS key is referenced
		log.Warn().Str("arn", a).Str("currentAccount", runtime.Connection.(*connection.AwsConnection).AccountId()).Str("keyAccount", arnVal.AccountID).Msg("cross-account KMS keys are not supported yet, returning ARN only")
		// Extract key ID from the ARN resource part (e.g., "key/uuid" -> "uuid")
		keyId := arnVal.Resource
		if strings.HasPrefix(keyId, "key/") {
			keyId = keyId[4:]
		}
		args["id"] = llx.StringData(keyId)
		args["arn"] = llx.StringData(a)
		args["region"] = llx.StringData(arnVal.Region)
		return args, nil, nil
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
	for _, rawResource := range rawResources.Data {
		key := rawResource.(*mqlAwsKmsKey)
		if key.Arn.Data == a {
			return args, key, nil
		}
	}
	return nil, nil, errors.New("key not found")
}
