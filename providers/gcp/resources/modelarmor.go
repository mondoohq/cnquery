// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"fmt"

	modelarmor "cloud.google.com/go/modelarmor/apiv1"
	"cloud.google.com/go/modelarmor/apiv1/modelarmorpb"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/gcp/connection"
	"go.mondoo.com/mql/v13/types"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type mqlGcpProjectModelArmorServiceInternal struct {
	serviceEnabled bool
}

func (g *mqlGcpProject) modelArmor() (*mqlGcpProjectModelArmorService, error) {
	if g.Id.Error != nil {
		return nil, g.Id.Error
	}
	res, err := CreateResource(g.MqlRuntime, "gcp.project.modelArmorService", map[string]*llx.RawData{
		"projectId": llx.StringData(g.Id.Data),
	})
	if err != nil {
		return nil, err
	}

	serviceEnabled, err := g.isServiceEnabled(service_modelarmor)
	if err != nil {
		return nil, err
	}

	svc := res.(*mqlGcpProjectModelArmorService)
	svc.serviceEnabled = serviceEnabled
	if !serviceEnabled {
		log.Debug().Str("service", service_modelarmor).Msg("gcp service is not enabled, skipping")
	}

	return svc, nil
}

func initGcpProjectModelArmorService(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 0 {
		return args, nil, nil
	}
	conn, ok := runtime.Connection.(*connection.GcpConnection)
	if !ok {
		return nil, nil, errors.New("invalid connection provided, it is not a GCP connection")
	}
	args = map[string]*llx.RawData{
		"projectId": llx.StringData(conn.ResourceID()),
	}
	return args, nil, nil
}

func (g *mqlGcpProjectModelArmorService) id() (string, error) {
	if g.ProjectId.Error != nil {
		return "", g.ProjectId.Error
	}
	return fmt.Sprintf("gcp.project/%s/modelArmorService", g.ProjectId.Data), nil
}

func (g *mqlGcpProjectModelArmorService) templates() ([]any, error) {
	if !g.serviceEnabled {
		return nil, nil
	}

	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)
	creds, err := conn.Credentials(modelarmor.DefaultAuthScopes()...)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	client, err := modelarmor.NewClient(ctx, option.WithCredentials(creds), connection.GRPCClientTraceOption())
	if err != nil {
		return nil, err
	}
	defer client.Close()

	it := client.ListTemplates(ctx, &modelarmorpb.ListTemplatesRequest{
		Parent: fmt.Sprintf("projects/%s/locations/-", projectId),
	})

	var res []any
	for {
		template, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, err
		}

		mqlTemplate, err := newMqlModelArmorServiceTemplate(g.MqlRuntime, projectId, template)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlTemplate)
	}

	return res, nil
}

// newMqlModelArmorServiceTemplate maps a Model Armor Template proto into the MQL
// resource. Shared by templates() and the discovered-asset init.
func newMqlModelArmorServiceTemplate(runtime *plugin.Runtime, projectId string, template *modelarmorpb.Template) (*mqlGcpProjectModelArmorServiceTemplate, error) {
	filterConfig, err := protoToDict(template.FilterConfig)
	if err != nil {
		return nil, err
	}
	templateMetadata, err := protoToDict(template.TemplateMetadata)
	if err != nil {
		return nil, err
	}

	res, err := CreateResource(runtime, "gcp.project.modelArmorService.template", map[string]*llx.RawData{
		"name":             llx.StringData(template.Name),
		"projectId":        llx.StringData(projectId),
		"createdAt":        llx.TimeDataPtr(timestampAsTimePtr(template.CreateTime)),
		"updatedAt":        llx.TimeDataPtr(timestampAsTimePtr(template.UpdateTime)),
		"labels":           llx.MapData(convert.MapToInterfaceMap(template.Labels), types.String),
		"filterConfig":     llx.DictData(filterConfig),
		"templateMetadata": llx.DictData(templateMetadata),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlGcpProjectModelArmorServiceTemplate), nil
}

// initGcpProjectModelArmorServiceTemplate resolves a single Model Armor template.
// When invoked for a discovered gcp-modelarmor-template asset (no args), it
// reconstructs the resource name from the asset identifier and fetches it so the
// asset resolves to exactly one template instead of an empty husk.
func initGcpProjectModelArmorServiceTemplate(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	// Fully populated (e.g., from CreateResource in templates()); nothing to do.
	if len(args) > 1 {
		return args, nil, nil
	}
	if args == nil {
		args = make(map[string]*llx.RawData)
	}

	// Resolve from the asset identifier when accessed as a discovered asset.
	if len(args) == 0 {
		ids := getAssetIdentifier(runtime)
		if ids == nil {
			return nil, nil, errors.New("no asset identifier found")
		}
		args["name"] = llx.StringData(fmt.Sprintf("projects/%s/locations/%s/templates/%s", ids.project, ids.region, ids.name))
	}

	nameRaw := args["name"]
	if nameRaw == nil {
		return args, nil, nil
	}
	name := nameRaw.Value.(string)

	conn, ok := runtime.Connection.(*connection.GcpConnection)
	if !ok {
		return nil, nil, errors.New("invalid connection provided, it is not a GCP connection")
	}
	creds, err := conn.Credentials(modelarmor.DefaultAuthScopes()...)
	if err != nil {
		return nil, nil, err
	}
	ctx := context.Background()
	client, err := modelarmor.NewClient(ctx, option.WithCredentials(creds), connection.GRPCClientTraceOption())
	if err != nil {
		return nil, nil, err
	}
	defer client.Close()

	template, err := client.GetTemplate(ctx, &modelarmorpb.GetTemplateRequest{Name: name})
	if err != nil {
		return nil, nil, err
	}

	res, err := newMqlModelArmorServiceTemplate(runtime, parseProjectFromPath(name), template)
	if err != nil {
		return nil, nil, err
	}
	return args, res, nil
}

func (g *mqlGcpProjectModelArmorServiceTemplate) id() (string, error) {
	return g.Name.Data, g.Name.Error
}

func (g *mqlGcpProjectModelArmorServiceFloorSetting) id() (string, error) {
	return g.Name.Data, g.Name.Error
}

func (g *mqlGcpProjectModelArmorService) floorSetting() (*mqlGcpProjectModelArmorServiceFloorSetting, error) {
	if !g.serviceEnabled {
		g.FloorSetting.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}

	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)
	creds, err := conn.Credentials(modelarmor.DefaultAuthScopes()...)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	client, err := modelarmor.NewClient(ctx, option.WithCredentials(creds), connection.GRPCClientTraceOption())
	if err != nil {
		return nil, err
	}
	defer client.Close()

	fs, err := client.GetFloorSetting(ctx, &modelarmorpb.GetFloorSettingRequest{
		Name: fmt.Sprintf("projects/%s/locations/global/floorSetting", projectId),
	})
	if err != nil {
		if s, ok := status.FromError(err); ok && (s.Code() == codes.NotFound || s.Code() == codes.PermissionDenied) {
			g.FloorSetting.State = plugin.StateIsNull | plugin.StateIsSet
			return nil, nil
		}
		return nil, err
	}

	filterConfig, err := protoToDict(fs.FilterConfig)
	if err != nil {
		return nil, err
	}
	aiPlatformFloorSetting, err := protoToDict(fs.AiPlatformFloorSetting)
	if err != nil {
		return nil, err
	}

	integratedServices := make([]any, 0, len(fs.IntegratedServices))
	for _, is := range fs.IntegratedServices {
		integratedServices = append(integratedServices, is.String())
	}

	res, err := CreateResource(g.MqlRuntime, "gcp.project.modelArmorService.floorSetting", map[string]*llx.RawData{
		"name":                          llx.StringData(fs.Name),
		"filterConfig":                  llx.DictData(filterConfig),
		"enableFloorSettingEnforcement": llx.BoolData(fs.GetEnableFloorSettingEnforcement()),
		"integratedServices":            llx.ArrayData(integratedServices, types.String),
		"aiPlatformFloorSetting":        llx.DictData(aiPlatformFloorSetting),
		"created":                       llx.TimeDataPtr(timestampAsTimePtr(fs.CreateTime)),
		"updated":                       llx.TimeDataPtr(timestampAsTimePtr(fs.UpdateTime)),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlGcpProjectModelArmorServiceFloorSetting), nil
}
