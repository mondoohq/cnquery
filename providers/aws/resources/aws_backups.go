// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/aws/aws-sdk-go-v2/service/backup"
	backuptypes "github.com/aws/aws-sdk-go-v2/service/backup/types"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
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

// ========================
// aws.backup.plan
// ========================

func (a *mqlAwsBackupPlan) id() (string, error) {
	return a.Arn.Data, nil
}

func (a *mqlAwsBackup) plans() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getPlans(conn), 5)
	poolOfJobs.Run()

	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	for i := range poolOfJobs.Jobs {
		res = append(res, poolOfJobs.Jobs[i].Result.([]any)...)
	}

	return res, nil
}

func (a *mqlAwsBackup) getPlans(conn *connection.AwsConnection) []*jobpool.Job {
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

			paginator := backup.NewListBackupPlansPaginator(svc, &backup.ListBackupPlansInput{})
			for paginator.HasMorePages() {
				page, err := paginator.NextPage(ctx)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS API")
						return res, nil
					}
					return nil, err
				}
				for _, plan := range page.BackupPlansList {
					advSettings, err := newMqlBackupAdvancedSettings(a.MqlRuntime, convert.ToValue(plan.BackupPlanArn), plan.AdvancedBackupSettings)
					if err != nil {
						return nil, err
					}

					mqlPlan, err := CreateResource(a.MqlRuntime, ResourceAwsBackupPlan,
						map[string]*llx.RawData{
							"__id":                   llx.StringDataPtr(plan.BackupPlanArn),
							"arn":                    llx.StringDataPtr(plan.BackupPlanArn),
							"id":                     llx.StringDataPtr(plan.BackupPlanId),
							"name":                   llx.StringDataPtr(plan.BackupPlanName),
							"versionId":              llx.StringDataPtr(plan.VersionId),
							"region":                 llx.StringData(region),
							"createdAt":              llx.TimeDataPtr(plan.CreationDate),
							"lastExecutionDate":      llx.TimeDataPtr(plan.LastExecutionDate),
							"deletionDate":           llx.TimeDataPtr(plan.DeletionDate),
							"advancedBackupSettings": llx.ArrayData(advSettings, types.Resource(ResourceAwsBackupPlanAdvancedBackupSetting)),
						})
					if err != nil {
						return nil, err
					}
					res = append(res, mqlPlan)
				}
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func (a *mqlAwsBackupPlan) rules() ([]any, error) {
	planId := a.Id.Data
	planArn := a.Arn.Data

	region, err := GetRegionFromArn(planArn)
	if err != nil {
		return nil, err
	}

	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	svc := conn.Backup(region)
	ctx := context.Background()

	resp, err := svc.GetBackupPlan(ctx, &backup.GetBackupPlanInput{
		BackupPlanId: &planId,
	})
	if err != nil {
		if Is400AccessDeniedError(err) {
			return nil, nil
		}
		return nil, err
	}

	if resp.BackupPlan == nil {
		return nil, nil
	}

	res := []any{}
	for _, rule := range resp.BackupPlan.Rules {
		mqlRule, err := newMqlBackupPlanRule(a.MqlRuntime, planArn, rule)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlRule)
	}
	return res, nil
}

func newMqlBackupPlanRule(runtime *plugin.Runtime, planArn string, rule backuptypes.BackupRule) (*mqlAwsBackupPlanRule, error) {
	ruleId := convert.ToValue(rule.RuleId)
	uniqueId := planArn + "\x00" + ruleId

	// Build lifecycle resource
	var lifecycle *mqlAwsBackupLifecycle
	if rule.Lifecycle != nil {
		lc, err := newMqlBackupLifecycle(runtime, uniqueId+"/lifecycle", rule.Lifecycle)
		if err != nil {
			return nil, err
		}
		lifecycle = lc
	}

	// Build copy actions
	copyActions := []any{}
	for _, ca := range rule.CopyActions {
		mqlCA, err := newMqlBackupCopyAction(runtime, uniqueId, ca)
		if err != nil {
			return nil, err
		}
		copyActions = append(copyActions, mqlCA)
	}

	// Convert recovery point tags
	var rpTags map[string]any
	if rule.RecoveryPointTags != nil {
		rpTags = toInterfaceMap(rule.RecoveryPointTags)
	}

	resource, err := CreateResource(runtime, ResourceAwsBackupPlanRule,
		map[string]*llx.RawData{
			"__id":                       llx.StringData(uniqueId),
			"id":                         llx.StringData(ruleId),
			"ruleName":                   llx.StringDataPtr(rule.RuleName),
			"targetBackupVaultName":      llx.StringDataPtr(rule.TargetBackupVaultName),
			"scheduleExpression":         llx.StringDataPtr(rule.ScheduleExpression),
			"scheduleExpressionTimezone": llx.StringDataPtr(rule.ScheduleExpressionTimezone),
			"startWindowMinutes":         llx.IntDataDefault(rule.StartWindowMinutes, 0),
			"completionWindowMinutes":    llx.IntDataDefault(rule.CompletionWindowMinutes, 0),
			"enableContinuousBackup":     llx.BoolDataPtr(rule.EnableContinuousBackup),
			"copyActions":                llx.ArrayData(copyActions, types.Resource(ResourceAwsBackupPlanRuleCopyAction)),
			"recoveryPointTags":          llx.MapData(rpTags, types.String),
		})
	if err != nil {
		return nil, err
	}

	mqlRule := resource.(*mqlAwsBackupPlanRule)
	if lifecycle != nil {
		mqlRule.Lifecycle = plugin.TValue[*mqlAwsBackupLifecycle]{Data: lifecycle, State: plugin.StateIsSet}
	} else {
		mqlRule.Lifecycle = plugin.TValue[*mqlAwsBackupLifecycle]{State: plugin.StateIsNull | plugin.StateIsSet}
	}

	return mqlRule, nil
}

func newMqlBackupLifecycle(runtime *plugin.Runtime, id string, lc *backuptypes.Lifecycle) (*mqlAwsBackupLifecycle, error) {
	resource, err := CreateResource(runtime, ResourceAwsBackupLifecycle,
		map[string]*llx.RawData{
			"__id":                                llx.StringData(id),
			"id":                                  llx.StringData(id),
			"deleteAfterDays":                     llx.IntDataDefault(lc.DeleteAfterDays, 0),
			"moveToColdStorageAfterDays":          llx.IntDataDefault(lc.MoveToColdStorageAfterDays, 0),
			"optInToArchiveForSupportedResources": llx.BoolDataPtr(lc.OptInToArchiveForSupportedResources),
		})
	if err != nil {
		return nil, err
	}
	return resource.(*mqlAwsBackupLifecycle), nil
}

func newMqlBackupCopyAction(runtime *plugin.Runtime, ruleId string, ca backuptypes.CopyAction) (*mqlAwsBackupPlanRuleCopyAction, error) {
	destArn := convert.ToValue(ca.DestinationBackupVaultArn)
	uniqueId := ruleId + "\x00copyAction\x00" + destArn

	var deleteAfterDays, moveToColdStorageDays int64
	var optInToArchive bool
	if ca.Lifecycle != nil {
		deleteAfterDays = convert.ToValue(ca.Lifecycle.DeleteAfterDays)
		moveToColdStorageDays = convert.ToValue(ca.Lifecycle.MoveToColdStorageAfterDays)
		if ca.Lifecycle.OptInToArchiveForSupportedResources != nil {
			optInToArchive = *ca.Lifecycle.OptInToArchiveForSupportedResources
		}
	}

	resource, err := CreateResource(runtime, ResourceAwsBackupPlanRuleCopyAction,
		map[string]*llx.RawData{
			"__id":                                llx.StringData(uniqueId),
			"id":                                  llx.StringData(uniqueId),
			"destinationBackupVaultArn":           llx.StringData(destArn),
			"deleteAfterDays":                     llx.IntData(deleteAfterDays),
			"moveToColdStorageAfterDays":          llx.IntData(moveToColdStorageDays),
			"optInToArchiveForSupportedResources": llx.BoolData(optInToArchive),
		})
	if err != nil {
		return nil, err
	}
	return resource.(*mqlAwsBackupPlanRuleCopyAction), nil
}

func newMqlBackupAdvancedSettings(runtime *plugin.Runtime, planArn string, settings []backuptypes.AdvancedBackupSetting) ([]any, error) {
	res := []any{}
	for _, s := range settings {
		resourceType := convert.ToValue(s.ResourceType)
		uniqueId := planArn + "\x00advSetting\x00" + resourceType

		mqlSetting, err := CreateResource(runtime, ResourceAwsBackupPlanAdvancedBackupSetting,
			map[string]*llx.RawData{
				"__id":          llx.StringData(uniqueId),
				"id":            llx.StringData(uniqueId),
				"resourceType":  llx.StringData(resourceType),
				"backupOptions": llx.MapData(toInterfaceMap(s.BackupOptions), types.String),
			})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlSetting)
	}
	return res, nil
}

func (a *mqlAwsBackupLifecycle) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAwsBackupPlanAdvancedBackupSetting) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAwsBackupPlanRule) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAwsBackupPlanRuleCopyAction) id() (string, error) {
	return a.Id.Data, nil
}
