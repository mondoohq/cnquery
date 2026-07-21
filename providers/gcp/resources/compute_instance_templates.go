// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/gcp/connection"
	"google.golang.org/api/cloudresourcemanager/v3"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/iam/v1"
	"google.golang.org/api/option"
)

func (g *mqlGcpProjectComputeService) instanceTemplates() ([]any, error) {
	// when the service is not enabled, we return nil
	if !g.GetEnabled().Data {
		return nil, nil
	}

	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)

	client, err := conn.Client(cloudresourcemanager.CloudPlatformReadOnlyScope, iam.CloudPlatformScope, compute.CloudPlatformScope)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()

	computeSvc, err := compute.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}

	var res []any
	req := computeSvc.InstanceTemplates.List(projectId)
	if err := req.Pages(ctx, func(page *compute.InstanceTemplateList) error {
		for _, tmpl := range page.Items {
			properties, err := convert.JsonToDict(tmpl.Properties)
			if err != nil {
				return err
			}

			mqlTmpl, err := CreateResource(g.MqlRuntime, "gcp.project.computeService.instanceTemplate", map[string]*llx.RawData{
				"id":                llx.StringData(strconv.FormatUint(tmpl.Id, 10)),
				"name":              llx.StringData(tmpl.Name),
				"description":       llx.StringData(tmpl.Description),
				"selfLink":          llx.StringData(tmpl.SelfLink),
				"sourceInstance":    llx.StringData(tmpl.SourceInstance),
				"properties":        llx.DictData(properties),
				"creationTimestamp": llx.TimeDataPtr(parseTime(tmpl.CreationTimestamp)),
			})
			if err != nil {
				return err
			}
			res = append(res, mqlTmpl)
		}
		return nil
	}); err != nil {
		return nil, err
	}

	return res, nil
}

func (g *mqlGcpProjectComputeServiceInstanceTemplate) id() (string, error) {
	if g.Id.Error != nil {
		return "", g.Id.Error
	}
	id := g.Id.Data
	return fmt.Sprintf("gcp.project.computeService.instanceTemplate/%s", id), nil
}

func (g *mqlGcpProjectComputeServiceInstanceTemplate) sourceInstanceRef() (*mqlGcpProjectComputeServiceInstance, error) {
	if g.SourceInstance.Error != nil {
		return nil, g.SourceInstance.Error
	}
	url := g.SourceInstance.Data
	if url == "" {
		g.SourceInstanceRef.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}

	// URL format: https://www.googleapis.com/compute/v1/projects/{project}/zones/{zone}/instances/{name}
	params := strings.TrimPrefix(url, "https://www.googleapis.com/compute/v1/")
	params = strings.TrimPrefix(params, "https://compute.googleapis.com/compute/v1/")
	parts := strings.Split(params, "/")
	if len(parts) < 6 || parts[0] != "projects" || parts[2] != "zones" || parts[4] != "instances" {
		g.SourceInstanceRef.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	project, zone, name := parts[1], parts[3], parts[5]

	res, err := NewResource(g.MqlRuntime, "gcp.project.computeService.instance", map[string]*llx.RawData{
		"name":      llx.StringData(name),
		"region":    llx.StringData(zone),
		"projectId": llx.StringData(project),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlGcpProjectComputeServiceInstance), nil
}
