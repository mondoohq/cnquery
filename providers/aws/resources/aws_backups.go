// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
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

			paginator := backup.NewListBackupVaultsPaginator(svc, &backup.ListBackupVaultsInput{})
			for paginator.HasMorePages() {
				page, err := paginator.NextPage(ctx)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS API")
						return res, nil
					}
					return nil, err
				}
				for _, v := range page.BackupVaultList {
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
					"arn":                  llx.StringDataPtr(rp.RecoveryPointArn),
					"completionDate":       llx.TimeDataPtr(rp.CompletionDate),
					"createdAt":            llx.TimeDataPtr(rp.CreationDate),
					"createdBy":            llx.MapData(createdBy, types.String),
					"encryptionKeyArn":     llx.StringDataPtr(rp.EncryptionKeyArn),
					"iamRoleArn":           llx.StringDataPtr(rp.IamRoleArn),
					"isEncrypted":          llx.BoolData(rp.IsEncrypted),
					"resourceType":         llx.StringDataPtr(rp.ResourceType),
					"status":               llx.StringData(string(rp.Status)),
					"sourceResourceArn":    llx.StringDataPtr(rp.ResourceArn),
					"sourceBackupVaultArn": llx.StringDataPtr(rp.SourceBackupVaultArn),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlRP)
		}
	}
	return res, nil
}

func (a *mqlAwsBackupVault) accessPolicy() (string, error) {
	vArn := a.Arn.Data
	parsedArn, err := arn.Parse(vArn)
	if err != nil {
		return "", err
	}
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	svc := conn.Backup(parsedArn.Region)
	ctx := context.Background()

	name := strings.TrimPrefix(parsedArn.Resource, "backup-vault:")
	resp, err := svc.GetBackupVaultAccessPolicy(ctx, &backup.GetBackupVaultAccessPolicyInput{
		BackupVaultName: &name,
	})
	if err != nil {
		// vaults without an access policy return ResourceNotFoundException
		var rnf *backuptypes.ResourceNotFoundException
		if errors.As(err, &rnf) {
			return "", nil
		}
		return "", err
	}
	if resp.Policy == nil {
		return "", nil
	}
	return *resp.Policy, nil
}

func (a *mqlAwsBackupVault) policyStatements() ([]any, error) {
	return policyStatementsFromString(a.MqlRuntime, a.Arn.Data, a.GetAccessPolicy())
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

func (a *mqlAwsBackupPlan) selections() ([]any, error) {
	planId := a.Id.Data
	planArn := a.Arn.Data

	region, err := GetRegionFromArn(planArn)
	if err != nil {
		return nil, err
	}

	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	svc := conn.Backup(region)
	ctx := context.Background()

	res := []any{}
	var nextToken *string
	for {
		resp, err := svc.ListBackupSelections(ctx, &backup.ListBackupSelectionsInput{
			BackupPlanId: &planId,
			NextToken:    nextToken,
		})
		if err != nil {
			if Is400AccessDeniedError(err) {
				return nil, nil
			}
			return nil, err
		}
		for _, sel := range resp.BackupSelectionsList {
			d, _ := convert.JsonToDict(sel)
			res = append(res, d)
		}
		if resp.NextToken == nil {
			break
		}
		nextToken = resp.NextToken
	}
	return res, nil
}

// ========================
// aws.backup.scanJob
// ========================

func (a *mqlAwsBackup) scanJobs() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getScanJobs(conn), 5)
	poolOfJobs.Run()

	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	for i := range poolOfJobs.Jobs {
		res = append(res, poolOfJobs.Jobs[i].Result.([]any)...)
	}

	return res, nil
}

func (a *mqlAwsBackup) getScanJobs(conn *connection.AwsConnection) []*jobpool.Job {
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

			paginator := backup.NewListScanJobsPaginator(svc, &backup.ListScanJobsInput{})
			for paginator.HasMorePages() {
				page, err := paginator.NextPage(ctx)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS API")
						return res, nil
					}
					return nil, err
				}
				for _, sj := range page.ScanJobs {
					mqlScanJob, err := newMqlBackupScanJob(a.MqlRuntime, region, sj)
					if err != nil {
						return nil, err
					}
					res = append(res, mqlScanJob)
				}
			}

			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func newMqlBackupScanJob(runtime *plugin.Runtime, region string, sj backuptypes.ScanJob) (*mqlAwsBackupScanJob, error) {
	args := map[string]*llx.RawData{
		"__id":                    llx.StringDataPtr(sj.ScanJobId),
		"id":                      llx.StringDataPtr(sj.ScanJobId),
		"accountId":               llx.StringDataPtr(sj.AccountId),
		"region":                  llx.StringData(region),
		"resourceArn":             llx.StringDataPtr(sj.ResourceArn),
		"resourceName":            llx.StringDataPtr(sj.ResourceName),
		"resourceType":            llx.StringData(string(sj.ResourceType)),
		"malwareScanner":          llx.StringData(string(sj.MalwareScanner)),
		"scanMode":                llx.StringData(string(sj.ScanMode)),
		"scanId":                  llx.StringDataPtr(sj.ScanId),
		"state":                   llx.StringData(string(sj.State)),
		"statusMessage":           llx.StringDataPtr(sj.StatusMessage),
		"createdAt":               llx.TimeDataPtr(sj.CreationDate),
		"completionDate":          llx.TimeDataPtr(sj.CompletionDate),
		"continuousScanStartTime": llx.TimeDataPtr(sj.ContinuousScanStartTime),
		"continuousScanEndTime":   llx.TimeDataPtr(sj.ContinuousScanEndTime),
	}

	if sj.ScanResult != nil {
		args["scanResultStatus"] = llx.StringData(string(sj.ScanResult.ScanResultStatus))
	} else {
		args["scanResultStatus"] = llx.StringData("")
	}

	var backupPlanVersion, backupRuleId string
	if sj.CreatedBy != nil {
		backupPlanVersion = convert.ToValue(sj.CreatedBy.BackupPlanVersion)
		backupRuleId = convert.ToValue(sj.CreatedBy.BackupRuleId)
	}
	args["backupPlanVersion"] = llx.StringData(backupPlanVersion)
	args["backupRuleId"] = llx.StringData(backupRuleId)

	resource, err := CreateResource(runtime, ResourceAwsBackupScanJob, args)
	if err != nil {
		return nil, err
	}

	mqlScanJob := resource.(*mqlAwsBackupScanJob)
	mqlScanJob.cacheVaultArn = convert.ToValue(sj.BackupVaultArn)
	mqlScanJob.cacheRecoveryPointArn = convert.ToValue(sj.RecoveryPointArn)
	mqlScanJob.cacheBaseRecoveryPointArn = convert.ToValue(sj.ScanBaseRecoveryPointArn)
	mqlScanJob.cacheIamRoleArn = convert.ToValue(sj.IamRoleArn)
	mqlScanJob.cacheScannerRoleArn = convert.ToValue(sj.ScannerRoleArn)
	if sj.CreatedBy != nil {
		mqlScanJob.cacheBackupPlanArn = convert.ToValue(sj.CreatedBy.BackupPlanArn)
	}
	return mqlScanJob, nil
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

	// Build malware scan actions
	scanActions := make([]any, 0, len(rule.ScanActions))
	for _, sa := range rule.ScanActions {
		scanActions = append(scanActions, map[string]any{
			"malwareScanner": string(sa.MalwareScanner),
			"scanMode":       string(sa.ScanMode),
		})
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
			"scanActions":                llx.ArrayData(scanActions, types.Dict),
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

func (a *mqlAwsBackupVault) encryptionKey() (*mqlAwsKmsKey, error) {
	arnVal := a.EncryptionKeyArn.Data
	if arnVal == "" {
		a.EncryptionKey.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	res, err := NewResource(a.MqlRuntime, "aws.kms.key",
		map[string]*llx.RawData{"arn": llx.StringData(arnVal)})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAwsKmsKey), nil
}

func (a *mqlAwsBackupVaultRecoveryPoint) iamRole() (*mqlAwsIamRole, error) {
	arnVal := a.IamRoleArn.Data
	if arnVal == "" {
		a.IamRole.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	res, err := NewResource(a.MqlRuntime, "aws.iam.role",
		map[string]*llx.RawData{"arn": llx.StringData(arnVal)})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAwsIamRole), nil
}

func (a *mqlAwsBackupVaultRecoveryPoint) encryptionKey() (*mqlAwsKmsKey, error) {
	arnVal := a.EncryptionKeyArn.Data
	if arnVal == "" {
		a.EncryptionKey.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	res, err := NewResource(a.MqlRuntime, "aws.kms.key",
		map[string]*llx.RawData{"arn": llx.StringData(arnVal)})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAwsKmsKey), nil
}

func (a *mqlAwsBackupPlanRuleCopyAction) destinationVault() (*mqlAwsBackupVault, error) {
	arnVal := a.DestinationBackupVaultArn.Data
	if arnVal == "" {
		a.DestinationVault.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	res, err := NewResource(a.MqlRuntime, "aws.backup.vault",
		map[string]*llx.RawData{"arn": llx.StringData(arnVal)})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAwsBackupVault), nil
}

type mqlAwsBackupScanJobInternal struct {
	cacheVaultArn             string
	cacheRecoveryPointArn     string
	cacheBaseRecoveryPointArn string
	cacheIamRoleArn           string
	cacheScannerRoleArn       string
	cacheBackupPlanArn        string
}

func (a *mqlAwsBackupScanJob) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAwsBackupScanJob) vault() (*mqlAwsBackupVault, error) {
	if a.cacheVaultArn == "" {
		a.Vault.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	res, err := NewResource(a.MqlRuntime, "aws.backup.vault",
		map[string]*llx.RawData{"arn": llx.StringData(a.cacheVaultArn)})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAwsBackupVault), nil
}

func (a *mqlAwsBackupScanJob) recoveryPoint() (*mqlAwsBackupVaultRecoveryPoint, error) {
	if a.cacheRecoveryPointArn == "" {
		a.RecoveryPoint.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	res, err := NewResource(a.MqlRuntime, "aws.backup.vaultRecoveryPoint",
		map[string]*llx.RawData{"arn": llx.StringData(a.cacheRecoveryPointArn)})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAwsBackupVaultRecoveryPoint), nil
}

func (a *mqlAwsBackupScanJob) baseRecoveryPoint() (*mqlAwsBackupVaultRecoveryPoint, error) {
	if a.cacheBaseRecoveryPointArn == "" {
		a.BaseRecoveryPoint.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	res, err := NewResource(a.MqlRuntime, "aws.backup.vaultRecoveryPoint",
		map[string]*llx.RawData{"arn": llx.StringData(a.cacheBaseRecoveryPointArn)})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAwsBackupVaultRecoveryPoint), nil
}

func (a *mqlAwsBackupScanJob) iamRole() (*mqlAwsIamRole, error) {
	if a.cacheIamRoleArn == "" {
		a.IamRole.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	res, err := NewResource(a.MqlRuntime, "aws.iam.role",
		map[string]*llx.RawData{"arn": llx.StringData(a.cacheIamRoleArn)})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAwsIamRole), nil
}

func (a *mqlAwsBackupScanJob) scannerRole() (*mqlAwsIamRole, error) {
	if a.cacheScannerRoleArn == "" {
		a.ScannerRole.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	res, err := NewResource(a.MqlRuntime, "aws.iam.role",
		map[string]*llx.RawData{"arn": llx.StringData(a.cacheScannerRoleArn)})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAwsIamRole), nil
}

func (a *mqlAwsBackupScanJob) backupPlan() (*mqlAwsBackupPlan, error) {
	if a.cacheBackupPlanArn == "" {
		a.BackupPlan.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	res, err := NewResource(a.MqlRuntime, "aws.backup.plan",
		map[string]*llx.RawData{"arn": llx.StringData(a.cacheBackupPlanArn)})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAwsBackupPlan), nil
}
