// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package aws

import (
	"context"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/backup"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/rs/zerolog/log"
	aws_provider "go.mondoo.com/cnquery/motor/providers/aws"
	"go.mondoo.com/cnquery/resources/library/jobpool"
	"go.mondoo.com/cnquery/resources/packs/core"
)

func (a *mqlAwsBackup) id() (string, error) {
	return "aws.backup", nil
}

func (a *mqlAwsBackupVault) id() (string, error) {
	return a.Arn()
}

func (a *mqlAwsBackupVaultRecoveryPoint) id() (string, error) {
	return a.Arn()
}

func (a *mqlAwsBackup) GetVaults() ([]interface{}, error) {
	provider, err := awsProvider(a.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(a.getVaults(provider), 5)
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

func (a *mqlAwsBackup) getVaults(provider *aws_provider.Provider) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := provider.GetRegions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		regionVal := region
		f := func() (jobpool.JobResult, error) {
			svc := provider.Backup(regionVal)
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
				mqlGroup, err := a.MotorRuntime.CreateResource("aws.backup.vault",
					"arn", core.ToString(v.BackupVaultArn),
					"name", core.ToString(v.BackupVaultName),
				)
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

func (a *mqlAwsBackupVault) GetRecoveryPoints() ([]interface{}, error) {
	vArn, err := a.Arn()
	if err != nil {
		return nil, err
	}
	parsedArn, err := arn.Parse(vArn)
	if err != nil {
		return nil, err
	}
	at, err := awsProvider(a.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	svc := at.Backup(parsedArn.Region)
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
			createdBy, err := core.JsonToDict(rp.CreatedBy)
			if err != nil {
				return nil, err
			}
			mqlRP, err := a.MotorRuntime.CreateResource("aws.backup.vaultRecoveryPoint",
				"arn", core.ToString(rp.RecoveryPointArn),
				"resourceType", core.ToString(rp.ResourceType),
				"createdBy", createdBy,
				"iamRoleArn", core.ToString(rp.IamRoleArn),
				"status", string(rp.Status),
				"creationDate", rp.CreationDate,
				"completionDate", rp.CompletionDate,
				"encryptionKeyArn", core.ToString(rp.EncryptionKeyArn),
				"isEncrypted", core.ToBool(&rp.IsEncrypted),
			)
			if err != nil {
				return nil, err
			}
			res = append(res, mqlRP)
		}
	}
	return res, nil
}
