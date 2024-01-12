// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/backup"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v10/llx"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/cnquery/v10/providers/aws/connection"
	"go.mondoo.com/cnquery/v10/types"
)

func (a *mqlAwsBackup) id() (string, error) {
	return "aws.backup", nil
}

func (a *mqlAwsBackupVault) id() (string, error) {
	return a.Arn.Data, nil
}

func (a *mqlAwsBackupVaultRecoveryPoint) id() (string, error) {
	return a.Arn.Data, nil
}

func (a *mqlAwsBackup) vaults() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(a.getVaults(conn), 5)
	poolOfJobs.Run()

	// check for errors
	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	// get all the results
	for i := range poolOfJobs.Jobs {
		res = append(res, poolOfJobs.Jobs[i].Result.([]interface{})...)
	}

	return res, nil
}

func (a *mqlAwsBackup) getVaults(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		regionVal := region
		f := func() (jobpool.JobResult, error) {
			svc := conn.Backup(regionVal)
			ctx := context.Background()
			res := []interface{}{}

			vaults, err := svc.ListBackupVaults(ctx, &backup.ListBackupVaultsInput{})
			if err != nil {
				if Is400AccessDeniedError(err) {
					log.Warn().Str("region", regionVal).Msg("error accessing region for AWS API")
					return res, nil
				}
				return nil, err
			}
			for _, v := range vaults.BackupVaultList {
				mqlGroup, err := CreateResource(a.MqlRuntime, "aws.backup.vault",
					map[string]*llx.RawData{
						"arn":              llx.StringDataPtr(v.BackupVaultArn),
						"name":             llx.StringDataPtr(v.BackupVaultName),
						"createdAt":        llx.TimeDataPtr(v.CreationDate),
						"region":           llx.StringData(regionVal),
						"locked":           llx.BoolDataPtr(v.Locked),
						"encryptionKeyArn": llx.StringDataPtr(v.EncryptionKeyArn),
					})
				if err != nil {
					return nil, err
				}
				res = append(res, mqlGroup)
			}

			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func (a *mqlAwsBackupVault) recoveryPoints() ([]interface{}, error) {
	vArn := a.Arn.Data
	parsedArn, err := arn.Parse(vArn)
	if err != nil {
		return nil, err
	}
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	svc := conn.Backup(parsedArn.Region)
	ctx := context.Background()
	res := []interface{}{}

	name := strings.TrimPrefix(parsedArn.Resource, "backup-vault:")
	nextToken := aws.String("no_token_to_start_with")
	params := &backup.ListRecoveryPointsByBackupVaultInput{BackupVaultName: &name}
	for nextToken != nil {
		recovPoints, err := svc.ListRecoveryPointsByBackupVault(ctx, params)
		if err != nil {
			return nil, err
		}
		nextToken = recovPoints.NextToken
		if recovPoints.NextToken != nil {
			params.NextToken = nextToken
		}
		for _, rp := range recovPoints.RecoveryPoints {
			createdBy, err := convert.JsonToDict(rp.CreatedBy)
			if err != nil {
				return nil, err
			}
			mqlRP, err := CreateResource(a.MqlRuntime, "aws.backup.vaultRecoveryPoint",
				map[string]*llx.RawData{
					"arn":              llx.StringDataPtr(rp.RecoveryPointArn),
					"resourceType":     llx.StringDataPtr(rp.ResourceType),
					"createdBy":        llx.MapData(createdBy, types.String),
					"iamRoleArn":       llx.StringDataPtr(rp.IamRoleArn),
					"status":           llx.StringData(string(rp.Status)),
					"creationDate":     llx.TimeDataPtr(rp.CreationDate),
					"completionDate":   llx.TimeDataPtr(rp.CompletionDate),
					"encryptionKeyArn": llx.StringDataPtr(rp.EncryptionKeyArn),
					"isEncrypted":      llx.BoolData(convert.ToBool(&rp.IsEncrypted)),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlRP)
		}
	}
	return res, nil
}
