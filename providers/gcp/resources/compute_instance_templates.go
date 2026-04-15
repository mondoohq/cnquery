// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"fmt"
	"strconv"

	"go.mondoo.com/mql/v13/llx"
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
