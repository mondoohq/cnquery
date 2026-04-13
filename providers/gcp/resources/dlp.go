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
