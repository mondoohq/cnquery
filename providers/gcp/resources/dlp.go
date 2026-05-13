// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"fmt"

	dlp "cloud.google.com/go/dlp/apiv2"
	"cloud.google.com/go/dlp/apiv2/dlppb"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/gcp/connection"
	"go.mondoo.com/mql/v13/types"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/protobuf/proto"
)

type mqlGcpProjectDlpServiceInternal struct {
	serviceEnabled bool
}

func (g *mqlGcpProject) dlp() (*mqlGcpProjectDlpService, error) {
	if g.Id.Error != nil {
		return nil, g.Id.Error
	}
	projectId := g.Id.Data

	res, err := CreateResource(g.MqlRuntime, "gcp.project.dlpService", map[string]*llx.RawData{
		"projectId": llx.StringData(projectId),
	})
	if err != nil {
		return nil, err
	}

	serviceEnabled, err := g.isServiceEnabled(service_dlp)
	if err != nil {
		return nil, err
	}

	svc := res.(*mqlGcpProjectDlpService)
	svc.serviceEnabled = serviceEnabled
	if !serviceEnabled {
		log.Debug().Str("service", service_dlp).Msg("gcp service is not enabled, skipping")
	}

	return svc, nil
}

func initGcpProjectDlpService(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 0 {
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

func (g *mqlGcpProjectDlpService) id() (string, error) {
	if g.ProjectId.Error != nil {
		return "", g.ProjectId.Error
	}
	return fmt.Sprintf("%s/gcp.project.dlpService", g.ProjectId.Data), nil
}

// ---------------------------------------------------------------
// Inspect Templates
// ---------------------------------------------------------------

func (g *mqlGcpProjectDlpServiceInspectTemplate) id() (string, error) {
	return g.Name.Data, g.Name.Error
}

func (g *mqlGcpProjectDlpService) inspectTemplates() ([]any, error) {
	if !g.serviceEnabled {
		return nil, nil
	}

	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)
	creds, err := conn.Credentials(dlp.DefaultAuthScopes()...)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	client, err := dlp.NewClient(ctx, option.WithCredentials(creds))
	if err != nil {
		return nil, err
	}
	defer client.Close()

	it := client.ListInspectTemplates(ctx, &dlppb.ListInspectTemplatesRequest{
		Parent: fmt.Sprintf("projects/%s", projectId),
	})

	var res []any
	for {
		tmpl, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			if isGRPCSkippable(err) {
				log.Warn().Err(err).Msg("could not list DLP inspect templates")
				return nil, nil
			}
			return nil, err
		}

		inspectConfig, err := protoToDict(tmpl.InspectConfig)
		if err != nil {
			return nil, err
		}

		mqlTmpl, err := CreateResource(g.MqlRuntime, "gcp.project.dlpService.inspectTemplate", map[string]*llx.RawData{
			"name":          llx.StringData(tmpl.Name),
			"displayName":   llx.StringData(tmpl.DisplayName),
			"description":   llx.StringData(tmpl.Description),
			"inspectConfig": llx.DictData(inspectConfig),
			"created":       llx.TimeDataPtr(timestampAsTimePtr(tmpl.CreateTime)),
			"updated":       llx.TimeDataPtr(timestampAsTimePtr(tmpl.UpdateTime)),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlTmpl)
	}

	return res, nil
}

// ---------------------------------------------------------------
// Deidentify Templates
// ---------------------------------------------------------------

func (g *mqlGcpProjectDlpServiceDeidentifyTemplate) id() (string, error) {
	return g.Name.Data, g.Name.Error
}

func (g *mqlGcpProjectDlpService) deidentifyTemplates() ([]any, error) {
	if !g.serviceEnabled {
		return nil, nil
	}

	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)
	creds, err := conn.Credentials(dlp.DefaultAuthScopes()...)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	client, err := dlp.NewClient(ctx, option.WithCredentials(creds))
	if err != nil {
		return nil, err
	}
	defer client.Close()

	it := client.ListDeidentifyTemplates(ctx, &dlppb.ListDeidentifyTemplatesRequest{
		Parent: fmt.Sprintf("projects/%s", projectId),
	})

	var res []any
	for {
		tmpl, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			if isGRPCSkippable(err) {
				log.Warn().Err(err).Msg("could not list DLP deidentify templates")
				return nil, nil
			}
			return nil, err
		}

		deidentifyConfig, err := protoToDict(tmpl.DeidentifyConfig)
		if err != nil {
			return nil, err
		}

		mqlTmpl, err := CreateResource(g.MqlRuntime, "gcp.project.dlpService.deidentifyTemplate", map[string]*llx.RawData{
			"name":             llx.StringData(tmpl.Name),
			"displayName":      llx.StringData(tmpl.DisplayName),
			"description":      llx.StringData(tmpl.Description),
			"deidentifyConfig": llx.DictData(deidentifyConfig),
			"created":          llx.TimeDataPtr(timestampAsTimePtr(tmpl.CreateTime)),
			"updated":          llx.TimeDataPtr(timestampAsTimePtr(tmpl.UpdateTime)),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlTmpl)
	}

	return res, nil
}

// ---------------------------------------------------------------
// Job Triggers
// ---------------------------------------------------------------

func (g *mqlGcpProjectDlpServiceJobTrigger) id() (string, error) {
	return g.Name.Data, g.Name.Error
}

func (g *mqlGcpProjectDlpService) jobTriggers() ([]any, error) {
	if !g.serviceEnabled {
		return nil, nil
	}

	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)
	creds, err := conn.Credentials(dlp.DefaultAuthScopes()...)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	client, err := dlp.NewClient(ctx, option.WithCredentials(creds))
	if err != nil {
		return nil, err
	}
	defer client.Close()

	it := client.ListJobTriggers(ctx, &dlppb.ListJobTriggersRequest{
		Parent: fmt.Sprintf("projects/%s", projectId),
	})

	var res []any
	for {
		jt, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			if isGRPCSkippable(err) {
				log.Warn().Err(err).Msg("could not list DLP job triggers")
				return nil, nil
			}
			return nil, err
		}

		inspectJob, err := protoToDict(jt.GetInspectJob())
		if err != nil {
			return nil, err
		}

		triggers := make([]any, 0, len(jt.Triggers))
		for _, t := range jt.Triggers {
			d, err := protoToDict(t)
			if err != nil {
				return nil, err
			}
			triggers = append(triggers, d)
		}

		errs := make([]any, 0, len(jt.Errors))
		for _, e := range jt.Errors {
			d, err := protoToDict(e)
			if err != nil {
				return nil, err
			}
			errs = append(errs, d)
		}

		mqlJt, err := CreateResource(g.MqlRuntime, "gcp.project.dlpService.jobTrigger", map[string]*llx.RawData{
			"name":        llx.StringData(jt.Name),
			"displayName": llx.StringData(jt.DisplayName),
			"description": llx.StringData(jt.Description),
			"status":      llx.StringData(jt.Status.String()),
			"inspectJob":  llx.DictData(inspectJob),
			"triggers":    llx.ArrayData(triggers, types.Dict),
			"errors":      llx.ArrayData(errs, types.Dict),
			"created":     llx.TimeDataPtr(timestampAsTimePtr(jt.CreateTime)),
			"updated":     llx.TimeDataPtr(timestampAsTimePtr(jt.UpdateTime)),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlJt)
	}

	return res, nil
}

// ---------------------------------------------------------------
// Stored Info Types
// ---------------------------------------------------------------

func (g *mqlGcpProjectDlpServiceStoredInfoType) id() (string, error) {
	return g.Name.Data, g.Name.Error
}

func (g *mqlGcpProjectDlpService) storedInfoTypes() ([]any, error) {
	if !g.serviceEnabled {
		return nil, nil
	}

	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)
	creds, err := conn.Credentials(dlp.DefaultAuthScopes()...)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	client, err := dlp.NewClient(ctx, option.WithCredentials(creds))
	if err != nil {
		return nil, err
	}
	defer client.Close()

	it := client.ListStoredInfoTypes(ctx, &dlppb.ListStoredInfoTypesRequest{
		Parent: fmt.Sprintf("projects/%s", projectId),
	})

	var res []any
	for {
		sit, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			if isGRPCSkippable(err) {
				log.Warn().Err(err).Msg("could not list DLP stored info types")
				return nil, nil
			}
			return nil, err
		}

		currentVersion, err := protoToDict(sit.CurrentVersion)
		if err != nil {
			return nil, err
		}

		pendingVersions := make([]any, 0, len(sit.PendingVersions))
		for _, pv := range sit.PendingVersions {
			d, err := protoToDict(pv)
			if err != nil {
				return nil, err
			}
			pendingVersions = append(pendingVersions, d)
		}

		mqlSit, err := CreateResource(g.MqlRuntime, "gcp.project.dlpService.storedInfoType", map[string]*llx.RawData{
			"name":            llx.StringData(sit.Name),
			"currentVersion":  llx.DictData(currentVersion),
			"pendingVersions": llx.ArrayData(pendingVersions, types.Dict),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlSit)
	}

	return res, nil
}

// ---------------------------------------------------------------
// Helpers shared by new accessors
// ---------------------------------------------------------------

const dlpDataProfilesLocation = "global"

func dlpProtoSliceToDict[T proto.Message](items []T) ([]any, error) {
	res := make([]any, 0, len(items))
	for _, it := range items {
		d, err := protoToDict(it)
		if err != nil {
			return nil, err
		}
		res = append(res, d)
	}
	return res, nil
}

// ---------------------------------------------------------------
// DLP Jobs
// ---------------------------------------------------------------

func (g *mqlGcpProjectDlpServiceDlpJob) id() (string, error) {
	return g.Name.Data, g.Name.Error
}

func (g *mqlGcpProjectDlpService) dlpJobs() ([]any, error) {
	if !g.serviceEnabled {
		return nil, nil
	}
	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)
	creds, err := conn.Credentials(dlp.DefaultAuthScopes()...)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	client, err := dlp.NewClient(ctx, option.WithCredentials(creds))
	if err != nil {
		return nil, err
	}
	defer client.Close()

	it := client.ListDlpJobs(ctx, &dlppb.ListDlpJobsRequest{
		Parent: fmt.Sprintf("projects/%s", projectId),
	})

	var res []any
	for {
		job, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			if isGRPCSkippable(err) {
				log.Warn().Err(err).Msg("could not list DLP jobs")
				return nil, nil
			}
			return nil, err
		}

		var details any
		switch d := job.Details.(type) {
		case *dlppb.DlpJob_InspectDetails:
			details, _ = protoToDict(d.InspectDetails)
		case *dlppb.DlpJob_RiskDetails:
			details, _ = protoToDict(d.RiskDetails)
		}

		errs, err := dlpProtoSliceToDict(job.Errors)
		if err != nil {
			return nil, err
		}
		actionDetails, err := dlpProtoSliceToDict(job.ActionDetails)
		if err != nil {
			return nil, err
		}

		mqlJob, err := CreateResource(g.MqlRuntime, "gcp.project.dlpService.dlpJob", map[string]*llx.RawData{
			"name":          llx.StringData(job.Name),
			"type":          llx.StringData(job.Type.String()),
			"state":         llx.StringData(job.State.String()),
			"jobTrigger":    llx.StringData(job.JobTriggerName),
			"details":       llx.DictData(details),
			"errors":        llx.ArrayData(errs, types.Dict),
			"actionDetails": llx.ArrayData(actionDetails, types.Dict),
			"created":       llx.TimeDataPtr(timestampAsTimePtr(job.CreateTime)),
			"started":       llx.TimeDataPtr(timestampAsTimePtr(job.StartTime)),
			"ended":         llx.TimeDataPtr(timestampAsTimePtr(job.EndTime)),
			"lastModified":  llx.TimeDataPtr(timestampAsTimePtr(job.LastModified)),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlJob)
	}

	return res, nil
}

// ---------------------------------------------------------------
// Discovery Configs
// ---------------------------------------------------------------

func (g *mqlGcpProjectDlpServiceDiscoveryConfig) id() (string, error) {
	return g.Name.Data, g.Name.Error
}

func (g *mqlGcpProjectDlpService) discoveryConfigs() ([]any, error) {
	if !g.serviceEnabled {
		return nil, nil
	}
	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)
	creds, err := conn.Credentials(dlp.DefaultAuthScopes()...)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	client, err := dlp.NewClient(ctx, option.WithCredentials(creds))
	if err != nil {
		return nil, err
	}
	defer client.Close()

	it := client.ListDiscoveryConfigs(ctx, &dlppb.ListDiscoveryConfigsRequest{
		Parent: fmt.Sprintf("projects/%s/locations/%s", projectId, dlpDataProfilesLocation),
	})

	var res []any
	for {
		dc, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			if isGRPCSkippable(err) {
				log.Warn().Err(err).Msg("could not list DLP discovery configs")
				return nil, nil
			}
			return nil, err
		}

		targets, err := dlpProtoSliceToDict(dc.Targets)
		if err != nil {
			return nil, err
		}
		errs, err := dlpProtoSliceToDict(dc.Errors)
		if err != nil {
			return nil, err
		}
		actions, err := dlpProtoSliceToDict(dc.Actions)
		if err != nil {
			return nil, err
		}
		orgConfig, _ := protoToDict(dc.OrgConfig)

		mqlDc, err := CreateResource(g.MqlRuntime, "gcp.project.dlpService.discoveryConfig", map[string]*llx.RawData{
			"name":             llx.StringData(dc.Name),
			"displayName":      llx.StringData(dc.DisplayName),
			"status":           llx.StringData(dc.Status.String()),
			"targets":          llx.ArrayData(targets, types.Dict),
			"errors":           llx.ArrayData(errs, types.Dict),
			"inspectTemplates": llx.ArrayData(stringsToAnySlice(dc.InspectTemplates), types.String),
			"actions":          llx.ArrayData(actions, types.Dict),
			"orgConfig":        llx.DictData(orgConfig),
			"lastRunTime":      llx.TimeDataPtr(timestampAsTimePtr(dc.LastRunTime)),
			"created":          llx.TimeDataPtr(timestampAsTimePtr(dc.CreateTime)),
			"updated":          llx.TimeDataPtr(timestampAsTimePtr(dc.UpdateTime)),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlDc)
	}

	return res, nil
}

func stringsToAnySlice(in []string) []any {
	out := make([]any, len(in))
	for i, s := range in {
		out[i] = s
	}
	return out
}

// ---------------------------------------------------------------
// Connections
// ---------------------------------------------------------------

func (g *mqlGcpProjectDlpServiceConnection) id() (string, error) {
	return g.Name.Data, g.Name.Error
}

func (g *mqlGcpProjectDlpService) connections() ([]any, error) {
	if !g.serviceEnabled {
		return nil, nil
	}
	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)
	creds, err := conn.Credentials(dlp.DefaultAuthScopes()...)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	client, err := dlp.NewClient(ctx, option.WithCredentials(creds))
	if err != nil {
		return nil, err
	}
	defer client.Close()

	it := client.ListConnections(ctx, &dlppb.ListConnectionsRequest{
		Parent: fmt.Sprintf("projects/%s/locations/%s", projectId, dlpDataProfilesLocation),
	})

	var res []any
	for {
		c, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			if isGRPCSkippable(err) {
				log.Warn().Err(err).Msg("could not list DLP connections")
				return nil, nil
			}
			return nil, err
		}

		errs, err := dlpProtoSliceToDict(c.Errors)
		if err != nil {
			return nil, err
		}
		var properties any
		switch p := c.Properties.(type) {
		case *dlppb.Connection_CloudSql:
			d, _ := protoToDict(p.CloudSql)
			properties = map[string]any{"cloudSql": d}
		}

		mqlConn, err := CreateResource(g.MqlRuntime, "gcp.project.dlpService.connection", map[string]*llx.RawData{
			"name":       llx.StringData(c.Name),
			"state":      llx.StringData(c.State.String()),
			"errors":     llx.ArrayData(errs, types.Dict),
			"properties": llx.DictData(properties),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlConn)
	}

	return res, nil
}

// ---------------------------------------------------------------
// Project Data Profiles
// ---------------------------------------------------------------

func (g *mqlGcpProjectDlpServiceProjectDataProfile) id() (string, error) {
	return g.Name.Data, g.Name.Error
}

func (g *mqlGcpProjectDlpService) projectDataProfiles() ([]any, error) {
	if !g.serviceEnabled {
		return nil, nil
	}
	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)
	creds, err := conn.Credentials(dlp.DefaultAuthScopes()...)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	client, err := dlp.NewClient(ctx, option.WithCredentials(creds))
	if err != nil {
		return nil, err
	}
	defer client.Close()

	it := client.ListProjectDataProfiles(ctx, &dlppb.ListProjectDataProfilesRequest{
		Parent: fmt.Sprintf("projects/%s/locations/%s", projectId, dlpDataProfilesLocation),
	})

	var res []any
	for {
		p, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			if isGRPCSkippable(err) {
				log.Warn().Err(err).Msg("could not list DLP project data profiles")
				return nil, nil
			}
			return nil, err
		}

		sensitivity, _ := protoToDict(p.SensitivityScore)
		riskLevel, _ := protoToDict(p.DataRiskLevel)
		status, _ := protoToDict(p.ProfileStatus)

		mqlP, err := CreateResource(g.MqlRuntime, "gcp.project.dlpService.projectDataProfile", map[string]*llx.RawData{
			"name":                      llx.StringData(p.Name),
			"projectId":                 llx.StringData(p.ProjectId),
			"sensitivityScore":          llx.DictData(sensitivity),
			"dataRiskLevel":             llx.DictData(riskLevel),
			"profileStatus":             llx.DictData(status),
			"tableDataProfileCount":     llx.IntData(p.TableDataProfileCount),
			"fileStoreDataProfileCount": llx.IntData(p.FileStoreDataProfileCount),
			"profileLastGenerated":      llx.TimeDataPtr(timestampAsTimePtr(p.ProfileLastGenerated)),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlP)
	}

	return res, nil
}

// ---------------------------------------------------------------
// Table Data Profiles
// ---------------------------------------------------------------

func (g *mqlGcpProjectDlpServiceTableDataProfile) id() (string, error) {
	return g.Name.Data, g.Name.Error
}

func (g *mqlGcpProjectDlpService) tableDataProfiles() ([]any, error) {
	if !g.serviceEnabled {
		return nil, nil
	}
	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)
	creds, err := conn.Credentials(dlp.DefaultAuthScopes()...)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	client, err := dlp.NewClient(ctx, option.WithCredentials(creds))
	if err != nil {
		return nil, err
	}
	defer client.Close()

	it := client.ListTableDataProfiles(ctx, &dlppb.ListTableDataProfilesRequest{
		Parent: fmt.Sprintf("projects/%s/locations/%s", projectId, dlpDataProfilesLocation),
	})

	var res []any
	for {
		t, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			if isGRPCSkippable(err) {
				log.Warn().Err(err).Msg("could not list DLP table data profiles")
				return nil, nil
			}
			return nil, err
		}

		mqlT, err := newMqlTableDataProfile(g.MqlRuntime, t)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlT)
	}

	return res, nil
}

func newMqlTableDataProfile(runtime *plugin.Runtime, t *dlppb.TableDataProfile) (plugin.Resource, error) {
	sensitivity, _ := protoToDict(t.SensitivityScore)
	riskLevel, _ := protoToDict(t.DataRiskLevel)
	status, _ := protoToDict(t.ProfileStatus)
	predicted, err := dlpProtoSliceToDict(t.PredictedInfoTypes)
	if err != nil {
		return nil, err
	}
	other, err := dlpProtoSliceToDict(t.OtherInfoTypes)
	if err != nil {
		return nil, err
	}
	return CreateResource(runtime, "gcp.project.dlpService.tableDataProfile", map[string]*llx.RawData{
		"name":                 llx.StringData(t.Name),
		"datasetProjectId":     llx.StringData(t.DatasetProjectId),
		"datasetLocation":      llx.StringData(t.DatasetLocation),
		"datasetId":            llx.StringData(t.DatasetId),
		"tableId":              llx.StringData(t.TableId),
		"fullResource":         llx.StringData(t.FullResource),
		"state":                llx.StringData(t.State.String()),
		"sensitivityScore":     llx.DictData(sensitivity),
		"dataRiskLevel":        llx.DictData(riskLevel),
		"profileStatus":        llx.DictData(status),
		"predictedInfoTypes":   llx.ArrayData(predicted, types.Dict),
		"otherInfoTypes":       llx.ArrayData(other, types.Dict),
		"encryptionStatus":     llx.StringData(t.EncryptionStatus.String()),
		"resourceVisibility":   llx.StringData(t.ResourceVisibility.String()),
		"scannedColumnCount":   llx.IntData(t.ScannedColumnCount),
		"failedColumnCount":    llx.IntData(t.FailedColumnCount),
		"tableSizeBytes":       llx.IntData(t.TableSizeBytes),
		"rowCount":             llx.IntData(t.RowCount),
		"resourceLabels":       llx.MapData(strMapToAny(t.ResourceLabels), types.String),
		"profileLastGenerated": llx.TimeDataPtr(timestampAsTimePtr(t.ProfileLastGenerated)),
		"lastModifiedTime":     llx.TimeDataPtr(timestampAsTimePtr(t.LastModifiedTime)),
		"expirationTime":       llx.TimeDataPtr(timestampAsTimePtr(t.ExpirationTime)),
		"created":              llx.TimeDataPtr(timestampAsTimePtr(t.CreateTime)),
	})
}

func strMapToAny(in map[string]string) map[string]any {
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func (g *mqlGcpProjectDlpServiceTableDataProfile) bigqueryTable() (*mqlGcpProjectBigqueryServiceTable, error) {
	datasetId := g.DatasetId.Data
	tableId := g.TableId.Data
	projectId := g.DatasetProjectId.Data
	if datasetId == "" || tableId == "" || projectId == "" {
		g.BigqueryTable.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	tblId := fmt.Sprintf("%s.%s.%s", projectId, datasetId, tableId)
	mqlTbl, err := NewResource(g.MqlRuntime, "gcp.project.bigqueryService.table", map[string]*llx.RawData{
		"id": llx.StringData(tblId),
	})
	if err != nil {
		return nil, err
	}
	return mqlTbl.(*mqlGcpProjectBigqueryServiceTable), nil
}

// ---------------------------------------------------------------
// Column Data Profiles
// ---------------------------------------------------------------

func (g *mqlGcpProjectDlpServiceColumnDataProfile) id() (string, error) {
	return g.Name.Data, g.Name.Error
}

func (g *mqlGcpProjectDlpService) columnDataProfiles() ([]any, error) {
	if !g.serviceEnabled {
		return nil, nil
	}
	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)
	creds, err := conn.Credentials(dlp.DefaultAuthScopes()...)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	client, err := dlp.NewClient(ctx, option.WithCredentials(creds))
	if err != nil {
		return nil, err
	}
	defer client.Close()

	it := client.ListColumnDataProfiles(ctx, &dlppb.ListColumnDataProfilesRequest{
		Parent: fmt.Sprintf("projects/%s/locations/%s", projectId, dlpDataProfilesLocation),
	})

	var res []any
	for {
		c, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			if isGRPCSkippable(err) {
				log.Warn().Err(err).Msg("could not list DLP column data profiles")
				return nil, nil
			}
			return nil, err
		}

		sensitivity, _ := protoToDict(c.SensitivityScore)
		riskLevel, _ := protoToDict(c.DataRiskLevel)
		columnInfoType, _ := protoToDict(c.ColumnInfoType)
		otherMatches, err := dlpProtoSliceToDict(c.OtherMatches)
		if err != nil {
			return nil, err
		}

		mqlC, err := CreateResource(g.MqlRuntime, "gcp.project.dlpService.columnDataProfile", map[string]*llx.RawData{
			"name":                 llx.StringData(c.Name),
			"column":               llx.StringData(c.Column),
			"datasetId":            llx.StringData(c.DatasetId),
			"tableId":              llx.StringData(c.TableId),
			"tableFullResource":    llx.StringData(c.TableFullResource),
			"state":                llx.StringData(c.State.String()),
			"sensitivityScore":     llx.DictData(sensitivity),
			"dataRiskLevel":        llx.DictData(riskLevel),
			"columnInfoType":       llx.DictData(columnInfoType),
			"otherMatches":         llx.ArrayData(otherMatches, types.Dict),
			"freeTextScore":        llx.FloatData(c.FreeTextScore),
			"columnType":           llx.StringData(c.ColumnType.String()),
			"policyState":          llx.StringData(c.PolicyState.String()),
			"profileLastGenerated": llx.TimeDataPtr(timestampAsTimePtr(c.ProfileLastGenerated)),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlC)
	}

	return res, nil
}

// ---------------------------------------------------------------
// File Store Data Profiles
// ---------------------------------------------------------------

func (g *mqlGcpProjectDlpServiceFileStoreDataProfile) id() (string, error) {
	return g.Name.Data, g.Name.Error
}

func (g *mqlGcpProjectDlpService) fileStoreDataProfiles() ([]any, error) {
	if !g.serviceEnabled {
		return nil, nil
	}
	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)
	creds, err := conn.Credentials(dlp.DefaultAuthScopes()...)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	client, err := dlp.NewClient(ctx, option.WithCredentials(creds))
	if err != nil {
		return nil, err
	}
	defer client.Close()

	it := client.ListFileStoreDataProfiles(ctx, &dlppb.ListFileStoreDataProfilesRequest{
		Parent: fmt.Sprintf("projects/%s/locations/%s", projectId, dlpDataProfilesLocation),
	})

	var res []any
	for {
		f, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			if isGRPCSkippable(err) {
				log.Warn().Err(err).Msg("could not list DLP file-store data profiles")
				return nil, nil
			}
			return nil, err
		}

		mqlF, err := newMqlFileStoreDataProfile(g.MqlRuntime, f)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlF)
	}

	return res, nil
}

func newMqlFileStoreDataProfile(runtime *plugin.Runtime, f *dlppb.FileStoreDataProfile) (plugin.Resource, error) {
	sensitivity, _ := protoToDict(f.SensitivityScore)
	riskLevel, _ := protoToDict(f.DataRiskLevel)
	status, _ := protoToDict(f.ProfileStatus)
	dataSourceType := ""
	if f.DataSourceType != nil {
		dataSourceType = f.DataSourceType.DataSource
	}
	clusterSummaries, err := dlpProtoSliceToDict(f.FileClusterSummaries)
	if err != nil {
		return nil, err
	}
	infoTypeSummaries, err := dlpProtoSliceToDict(f.FileStoreInfoTypeSummaries)
	if err != nil {
		return nil, err
	}
	resourceAttributes := map[string]any{}
	for k, v := range f.ResourceAttributes {
		d, err := protoToDict(v)
		if err != nil {
			return nil, err
		}
		resourceAttributes[k] = d
	}

	return CreateResource(runtime, "gcp.project.dlpService.fileStoreDataProfile", map[string]*llx.RawData{
		"name":                       llx.StringData(f.Name),
		"projectId":                  llx.StringData(f.ProjectId),
		"dataSourceType":             llx.StringData(dataSourceType),
		"fileStoreLocation":          llx.StringData(f.FileStoreLocation),
		"dataStorageLocations":       llx.ArrayData(stringsToAnySlice(f.DataStorageLocations), types.String),
		"locationType":               llx.StringData(f.LocationType),
		"fileStorePath":              llx.StringData(f.FileStorePath),
		"fullResource":               llx.StringData(f.FullResource),
		"profileStatus":              llx.DictData(status),
		"state":                      llx.StringData(f.State.String()),
		"resourceVisibility":         llx.StringData(f.ResourceVisibility.String()),
		"sensitivityScore":           llx.DictData(sensitivity),
		"dataRiskLevel":              llx.DictData(riskLevel),
		"fileClusterSummaries":       llx.ArrayData(clusterSummaries, types.Dict),
		"resourceAttributes":         llx.DictData(resourceAttributes),
		"resourceLabels":             llx.MapData(strMapToAny(f.ResourceLabels), types.String),
		"fileStoreInfoTypeSummaries": llx.ArrayData(infoTypeSummaries, types.Dict),
		"fileStoreIsEmpty":           llx.BoolData(f.FileStoreIsEmpty),
		"profileLastGenerated":       llx.TimeDataPtr(timestampAsTimePtr(f.ProfileLastGenerated)),
		"created":                    llx.TimeDataPtr(timestampAsTimePtr(f.CreateTime)),
		"lastModifiedTime":           llx.TimeDataPtr(timestampAsTimePtr(f.LastModifiedTime)),
	})
}

func (g *mqlGcpProjectDlpServiceFileStoreDataProfile) bucket() (*mqlGcpProjectStorageServiceBucket, error) {
	path := g.FileStorePath.Data
	bucketName := ""
	const gsPrefix = "gs://"
	if len(path) > len(gsPrefix) && path[:len(gsPrefix)] == gsPrefix {
		bucketName = path[len(gsPrefix):]
	}
	if bucketName == "" {
		g.Bucket.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	mqlBucket, err := NewResource(g.MqlRuntime, "gcp.project.storageService.bucket", map[string]*llx.RawData{
		"id": llx.StringData(bucketName),
	})
	if err != nil {
		return nil, err
	}
	return mqlBucket.(*mqlGcpProjectStorageServiceBucket), nil
}

// ---------------------------------------------------------------
// Inverse traversals
// ---------------------------------------------------------------

func (g *mqlGcpProjectStorageServiceBucket) dlpDataProfile() (*mqlGcpProjectDlpServiceFileStoreDataProfile, error) {
	bucketName := g.Id.Data
	if g.ProjectNumber.Error != nil {
		g.DlpDataProfile.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)
	projectId := conn.ResourceID()
	if projectId == "" || bucketName == "" {
		g.DlpDataProfile.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}

	ctx := context.Background()
	creds, err := conn.Credentials(dlp.DefaultAuthScopes()...)
	if err != nil {
		return nil, err
	}
	client, err := dlp.NewClient(ctx, option.WithCredentials(creds))
	if err != nil {
		return nil, err
	}
	defer client.Close()

	wantPath := "gs://" + bucketName
	it := client.ListFileStoreDataProfiles(ctx, &dlppb.ListFileStoreDataProfilesRequest{
		Parent: fmt.Sprintf("projects/%s/locations/%s", projectId, dlpDataProfilesLocation),
		Filter: fmt.Sprintf("file_store_path=\"%s\"", wantPath),
	})
	for {
		f, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			if isGRPCSkippable(err) {
				log.Warn().Err(err).Msg("could not look up DLP file-store data profile for bucket")
				g.DlpDataProfile.State = plugin.StateIsSet | plugin.StateIsNull
				return nil, nil
			}
			return nil, err
		}
		if f.FileStorePath == wantPath {
			res, err := newMqlFileStoreDataProfile(g.MqlRuntime, f)
			if err != nil {
				return nil, err
			}
			return res.(*mqlGcpProjectDlpServiceFileStoreDataProfile), nil
		}
	}

	g.DlpDataProfile.State = plugin.StateIsSet | plugin.StateIsNull
	return nil, nil
}

func (g *mqlGcpProjectBigqueryServiceTable) dlpDataProfile() (*mqlGcpProjectDlpServiceTableDataProfile, error) {
	tableId := g.Id.Data
	datasetId := g.DatasetId.Data
	projectId := g.ProjectId.Data
	if tableId == "" || datasetId == "" || projectId == "" {
		g.DlpDataProfile.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)
	ctx := context.Background()
	creds, err := conn.Credentials(dlp.DefaultAuthScopes()...)
	if err != nil {
		return nil, err
	}
	client, err := dlp.NewClient(ctx, option.WithCredentials(creds))
	if err != nil {
		return nil, err
	}
	defer client.Close()

	currentProject := conn.ResourceID()
	if currentProject == "" {
		currentProject = projectId
	}
	it := client.ListTableDataProfiles(ctx, &dlppb.ListTableDataProfilesRequest{
		Parent: fmt.Sprintf("projects/%s/locations/%s", currentProject, dlpDataProfilesLocation),
		Filter: fmt.Sprintf("table_id=\"%s\" AND dataset_id=\"%s\" AND project_id=\"%s\"", tableId, datasetId, projectId),
	})
	for {
		t, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			if isGRPCSkippable(err) {
				log.Warn().Err(err).Msg("could not look up DLP table data profile for BigQuery table")
				g.DlpDataProfile.State = plugin.StateIsSet | plugin.StateIsNull
				return nil, nil
			}
			return nil, err
		}
		if t.TableId == tableId && t.DatasetId == datasetId && t.DatasetProjectId == projectId {
			res, err := newMqlTableDataProfile(g.MqlRuntime, t)
			if err != nil {
				return nil, err
			}
			return res.(*mqlGcpProjectDlpServiceTableDataProfile), nil
		}
	}

	g.DlpDataProfile.State = plugin.StateIsSet | plugin.StateIsNull
	return nil, nil
}
