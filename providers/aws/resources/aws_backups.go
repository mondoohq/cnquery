// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/aws/aws-sdk-go-v2/service/backup"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/mql/v13/providers/aws/connection"
	"go.mondoo.com/mql/v13/types"
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

func (a *mqlAwsBackup) vaults() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getVaults(conn), 5)
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

func (a *mqlAwsBackup) getVaults(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		f := func() (jobpool.JobResult, error) {
			svc := conn.Backup(region)
			ctx := context.Background()
			res := []any{}

			vaults, err := svc.ListBackupVaults(ctx, &backup.ListBackupVaultsInput{})
			if err != nil {
				if Is400AccessDeniedError(err) {
					log.Warn().Str("region", region).Msg("error accessing region for AWS API")
					return res, nil
				}
				return nil, err
			}
			for _, v := range vaults.BackupVaultList {
				mqlGroup, err := CreateResource(a.MqlRuntime, "aws.backup.vault",
					map[string]*llx.RawData{
						"arn":              llx.StringDataPtr(v.BackupVaultArn),
						"createdAt":        llx.TimeDataPtr(v.CreationDate),
						"encryptionKeyArn": llx.StringDataPtr(v.EncryptionKeyArn),
						"locked":           llx.BoolDataPtr(v.Locked),
						"lockedAt":         llx.TimeDataPtr(v.LockDate),
						"maxRetentionDays": llx.IntDataPtr(v.MaxRetentionDays),
						"minRetentionDays": llx.IntDataPtr(v.MinRetentionDays),
						"name":             llx.StringDataPtr(v.BackupVaultName),
						"region":           llx.StringData(region),
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

func (a *mqlAwsBackupVault) recoveryPoints() ([]any, error) {
	vArn := a.Arn.Data
	parsedArn, err := arn.Parse(vArn)
	if err != nil {
		return nil, err
	}
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	svc := conn.Backup(parsedArn.Region)
	ctx := context.Background()
	res := []any{}

	name := strings.TrimPrefix(parsedArn.Resource, "backup-vault:")
	params := &backup.ListRecoveryPointsByBackupVaultInput{BackupVaultName: &name}
	paginator := backup.NewListRecoveryPointsByBackupVaultPaginator(svc, params)
	for paginator.HasMorePages() {
		recovPoints, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, rp := range recovPoints.RecoveryPoints {
			createdBy, err := convert.JsonToDict(rp.CreatedBy)
			if err != nil {
				return nil, err
			}
			mqlRP, err := CreateResource(a.MqlRuntime, "aws.backup.vaultRecoveryPoint",
				map[string]*llx.RawData{
					"arn":              llx.StringDataPtr(rp.RecoveryPointArn),
					"completionDate":   llx.TimeDataPtr(rp.CompletionDate),
					"createdAt":        llx.TimeDataPtr(rp.CreationDate),
					"createdBy":        llx.MapData(createdBy, types.String),
					"creationDate":     llx.TimeDataPtr(rp.CreationDate),
					"encryptionKeyArn": llx.StringDataPtr(rp.EncryptionKeyArn),
					"iamRoleArn":       llx.StringDataPtr(rp.IamRoleArn),
					"isEncrypted":      llx.BoolData(rp.IsEncrypted),
					"resourceType":     llx.StringDataPtr(rp.ResourceType),
					"status":           llx.StringData(string(rp.Status)),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlRP)
		}
	}
	return res, nil
}
