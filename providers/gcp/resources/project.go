// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/gcp/connection"
	"go.mondoo.com/mql/v13/types"

	"google.golang.org/api/cloudresourcemanager/v3"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/iam/v1"
	"google.golang.org/api/option"
)

func (g *mqlGcpProjects) id() (string, error) {
	if g.ParentId.Error != nil {
		return "", g.ParentId.Error
	}
	id := g.ParentId.Data
	return fmt.Sprintf("gcp.projects/%s", id), nil
}

type mqlGcpProjectInternal struct {
	// serviceEnabled services
	enabledServices     map[string]struct{}
	enabledServicesOnce sync.Once
	enabledServicesErr  error
	iamPolicyOnce       sync.Once
	iamPolicyCache      *cloudresourcemanager.Policy
	iamPolicyErr        error
}

func initGcpProject(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}

	if args == nil {
		args = make(map[string]*llx.RawData)
	}

	conn, ok := runtime.Connection.(*connection.GcpConnection)
	if !ok {
		return nil, nil, errors.New("invalid connection provided, it is not a GCP connection")
	}

	client, err := conn.Client(cloudresourcemanager.CloudPlatformReadOnlyScope, iam.CloudPlatformScope, compute.CloudPlatformScope)
	if err != nil {
		return nil, nil, err
	}

	ctx := context.Background()
	svc, err := cloudresourcemanager.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, nil, err
	}

	projectId := fmt.Sprintf("projects/%s", conn.ResourceID())
	project, err := svc.Projects.Get(projectId).Do()
	if err != nil {
		return nil, nil, err
	}

	args["id"] = llx.StringData(project.ProjectId)
	args["name"] = llx.StringData(project.DisplayName)
	args["parentId"] = llx.StringData(project.Parent)
	args["state"] = llx.StringData(project.State)
	args["createTime"] = llx.TimeDataPtr(parseTime(project.CreateTime))
	args["labels"] = llx.MapData(convert.MapToInterfaceMap(project.Labels), types.String)
	args["deleteTime"] = llx.TimeDataPtr(parseTime(project.DeleteTime))
	args["number"] = llx.StringData(strings.TrimPrefix(project.Name, "projects/"))
	// TODO: add organization gcp.organization
	return args, nil, nil
}

func (g *mqlGcpProject) id() (string, error) {
	return g.Id.Data, g.Id.Error
}

func (g *mqlGcpProject) name() (string, error) {
	// placeholder to convince MQL that this is an optional field
	// should never be called since the data is initialized in init
	return "", errors.New("not implemented")
}

func (g *mqlGcpProject) parentId() (string, error) {
	// placeholder to convince MQL that this is an optional field
	// should never be called since the data is initialized in init
	return "", errors.New("not implemented")
}

func (g *mqlGcpProject) state() (string, error) {
	// placeholder to convince MQL that this is an optional field
	// should never be called since the data is initialized in init
	return "", errors.New("not implemented")
}

func (g *mqlGcpProject) createTime() (*time.Time, error) {
	// placeholder to convince MQL that this is an optional field
	// should never be called since the data is initialized in init
	return nil, errors.New("not implemented")
}

func (g *mqlGcpProject) labels() (map[string]any, error) {
	// placeholder to convince MQL that this is an optional field
	// should never be called since the data is initialized in init
	return nil, errors.New("not implemented")
}

func (g *mqlGcpProject) deleteTime() (*time.Time, error) {
	return nil, errors.New("not implemented")
}

func (g *mqlGcpProject) number() (string, error) {
	return "", errors.New("not implemented")
}

func (g *mqlGcpProject) fetchIamPolicy() (*cloudresourcemanager.Policy, error) {
	g.iamPolicyOnce.Do(func() {
		projectId := g.Id.Data
		conn := g.MqlRuntime.Connection.(*connection.GcpConnection)
		client, err := conn.Client(cloudresourcemanager.CloudPlatformReadOnlyScope, iam.CloudPlatformScope, compute.CloudPlatformScope)
		if err != nil {
			g.iamPolicyErr = err
			return
		}
		ctx := context.Background()
		svc, err := cloudresourcemanager.NewService(ctx, option.WithHTTPClient(client))
		if err != nil {
			g.iamPolicyErr = err
			return
		}
		g.iamPolicyCache, g.iamPolicyErr = svc.Projects.GetIamPolicy(fmt.Sprintf("projects/%s", projectId), &cloudresourcemanager.GetIamPolicyRequest{Options: &cloudresourcemanager.GetPolicyOptions{RequestedPolicyVersion: 3}}).Do()
	})
	return g.iamPolicyCache, g.iamPolicyErr
}

func (g *mqlGcpProject) iamPolicy() ([]any, error) {
	if g.Id.Error != nil {
		return nil, g.Id.Error
	}
	projectId := g.Id.Data

	policy, err := g.fetchIamPolicy()
	if err != nil {
		return nil, err
	}

	res := []any{}
	for i := range policy.Bindings {
		b := policy.Bindings[i]

		condTitle, condExpr, condDesc := "", "", ""
		if b.Condition != nil {
			condTitle = b.Condition.Title
			condExpr = b.Condition.Expression
			condDesc = b.Condition.Description
		}

		mqlServiceaccount, err := CreateResource(g.MqlRuntime, "gcp.resourcemanager.binding", map[string]*llx.RawData{
			"id":                   llx.StringData(projectId + "-" + strconv.Itoa(i)),
			"role":                 llx.StringData(b.Role),
			"members":              llx.ArrayData(convert.SliceAnyToInterface(b.Members), types.String),
			"conditionTitle":       llx.StringData(condTitle),
			"conditionExpression":  llx.StringData(condExpr),
			"conditionDescription": llx.StringData(condDesc),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlServiceaccount)
	}

	return res, nil
}

func (g *mqlGcpProject) hasPublicIamBinding() (bool, error) {
	bindings := g.GetIamPolicy()
	if bindings.Error != nil {
		return false, bindings.Error
	}
	return iamPolicyHasPublicMember(bindings.Data)
}

func (g *mqlGcpProject) primitiveRoleBindings() ([]any, error) {
	bindings := g.GetIamPolicy()
	if bindings.Error != nil {
		return nil, bindings.Error
	}
	res := make([]any, 0)
	for _, raw := range bindings.Data {
		b, ok := raw.(*mqlGcpResourcemanagerBinding)
		if !ok || b == nil {
			continue
		}
		role := b.GetRole()
		if role.Error != nil {
			return nil, role.Error
		}
		switch role.Data {
		case "roles/owner", "roles/editor", "roles/viewer":
			res = append(res, b)
		}
	}
	return res, nil
}

func (g *mqlGcpProject) dataAccessLoggingEnabled() (bool, error) {
	configs := g.GetAuditConfig()
	if configs.Error != nil {
		return false, configs.Error
	}
	for _, raw := range configs.Data {
		cfg, ok := raw.(*mqlGcpResourcemanagerAuditConfig)
		if !ok || cfg == nil {
			continue
		}
		service := cfg.GetService()
		if service.Error != nil {
			return false, service.Error
		}
		if service.Data != "allServices" {
			continue
		}
		logConfigs := cfg.GetAuditLogConfigs()
		if logConfigs.Error != nil {
			return false, logConfigs.Error
		}
		var hasDataRead, hasDataWrite bool
		for _, lcRaw := range logConfigs.Data {
			lc, ok := lcRaw.(*mqlGcpResourcemanagerAuditConfigLogConfig)
			if !ok || lc == nil {
				continue
			}
			logType := lc.GetLogType()
			if logType.Error != nil {
				return false, logType.Error
			}
			exempted := lc.GetExemptedMembers()
			if exempted.Error != nil {
				return false, exempted.Error
			}
			if len(exempted.Data) > 0 {
				continue
			}
			switch logType.Data {
			case "DATA_READ":
				hasDataRead = true
			case "DATA_WRITE":
				hasDataWrite = true
			}
		}
		if hasDataRead && hasDataWrite {
			return true, nil
		}
	}
	return false, nil
}

func (g *mqlGcpProject) auditConfig() ([]any, error) {
	if g.Id.Error != nil {
		return nil, g.Id.Error
	}
	projectId := g.Id.Data

	policy, err := g.fetchIamPolicy()
	if err != nil {
		return nil, err
	}

	return extractAuditConfigs(g.MqlRuntime, fmt.Sprintf("projects/%s", projectId), policy.AuditConfigs)
}

func (g *mqlGcpProject) commonInstanceMetadata() (map[string]any, error) {
	if g.Id.Error != nil {
		return nil, g.Id.Error
	}
	projectId := g.Id.Data

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

	p, err := computeSvc.Projects.Get(projectId).Do()
	if err != nil {
		return nil, err
	}

	metadata := make(map[string]string)
	if p.CommonInstanceMetadata != nil {
		for _, item := range p.CommonInstanceMetadata.Items {
			value := ""
			if item.Value != nil {
				value = *item.Value
			}
			metadata[item.Key] = value
		}
	}
	return convert.MapToInterfaceMap(metadata), nil
}

func (g *mqlGcpProject) liens() ([]any, error) {
	if g.Id.Error != nil {
		return nil, g.Id.Error
	}
	projectId := g.Id.Data

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)
	client, err := conn.Client(cloudresourcemanager.CloudPlatformReadOnlyScope)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	svc, err := cloudresourcemanager.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}

	var res []any
	err = svc.Liens.List().Parent(fmt.Sprintf("projects/%s", projectId)).Pages(ctx, func(page *cloudresourcemanager.ListLiensResponse) error {
		for _, l := range page.Liens {
			mqlLien, err := CreateResource(g.MqlRuntime, "gcp.project.lien", map[string]*llx.RawData{
				"name":         llx.StringData(l.Name),
				"origin":       llx.StringData(l.Origin),
				"reason":       llx.StringData(l.Reason),
				"restrictions": llx.ArrayData(convert.SliceAnyToInterface(l.Restrictions), types.String),
				"createTime":   llx.TimeDataPtr(parseTime(l.CreateTime)),
			})
			if err != nil {
				return err
			}
			res = append(res, mqlLien)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (g *mqlGcpProjectLien) id() (string, error) {
	return g.Name.Data, g.Name.Error
}

func (g *mqlGcpProject) tagBindings() ([]any, error) {
	if g.Id.Error != nil {
		return nil, g.Id.Error
	}
	projectId := g.Id.Data

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)
	client, err := conn.Client(cloudresourcemanager.CloudPlatformReadOnlyScope)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	svc, err := cloudresourcemanager.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}

	// TagBindings.List requires the project number in the full resource name.
	// The number is already resolved when the gcp.project resource is created,
	// so read it rather than making another Projects.Get call; fall back to the
	// API only if it was not populated at creation.
	projectNumber := g.GetNumber().Data
	if projectNumber == "" {
		p, err := svc.Projects.Get(fmt.Sprintf("projects/%s", projectId)).Do()
		if err != nil {
			return nil, err
		}
		projectNumber = strings.TrimPrefix(p.Name, "projects/")
	}
	parent := fmt.Sprintf("//cloudresourcemanager.googleapis.com/projects/%s", projectNumber)

	var res []any
	err = svc.TagBindings.List().Parent(parent).Pages(ctx, func(page *cloudresourcemanager.ListTagBindingsResponse) error {
		for _, tb := range page.TagBindings {
			mqlTb, err := CreateResource(g.MqlRuntime, "gcp.project.tagBinding", map[string]*llx.RawData{
				"name":                   llx.StringData(tb.Name),
				"tagValue":               llx.StringData(tb.TagValue),
				"tagValueNamespacedName": llx.StringData(tb.TagValueNamespacedName),
				"resource":               llx.StringData(tb.Parent),
			})
			if err != nil {
				return err
			}
			res = append(res, mqlTb)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (g *mqlGcpProjectTagBinding) id() (string, error) {
	return g.Name.Data, g.Name.Error
}

func (g *mqlGcpProjects) children() ([]any, error) {
	if g.ParentId.Error != nil {
		return nil, g.ParentId.Error
	}
	parentId := g.ParentId.Data

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)

	client, err := conn.Client(cloudresourcemanager.CloudPlatformReadOnlyScope, iam.CloudPlatformScope, compute.CloudPlatformScope)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	svc, err := cloudresourcemanager.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}

	var mqlProjects []any
	if err := svc.Projects.List().Parent(parentId).Pages(ctx, func(page *cloudresourcemanager.ListProjectsResponse) error {
		for _, p := range page.Projects {
			mqlP, err := projectToMql(g.MqlRuntime, p)
			if err != nil {
				return err
			}
			mqlProjects = append(mqlProjects, mqlP)
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return mqlProjects, nil
}

func (g *mqlGcpProjects) list() ([]any, error) {
	if g.ParentId.Error != nil {
		return nil, g.ParentId.Error
	}
	parentId := g.ParentId.Data

	obj, err := CreateResource(g.MqlRuntime, "gcp.folders", map[string]*llx.RawData{
		"parentId": llx.StringData(parentId),
	})
	if err != nil {
		return nil, err
	}
	foldersSvc := obj.(*mqlGcpFolders)
	folders := foldersSvc.GetList()
	if folders.Error != nil {
		return nil, folders.Error
	}

	foldersMap := map[string]struct{}{parentId: {}}
	for _, f := range folders.Data {
		id := f.(*mqlGcpFolder).GetId()
		if id.Error != nil {
			return nil, id.Error
		}
		foldersMap[id.Data] = struct{}{}
	}

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)

	client, err := conn.Client(cloudresourcemanager.CloudPlatformReadOnlyScope, iam.CloudPlatformScope, compute.CloudPlatformScope)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	svc, err := cloudresourcemanager.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}

	var mqlProjects []any
	if err := svc.Projects.Search().Pages(ctx, func(page *cloudresourcemanager.SearchProjectsResponse) error {
		for _, p := range page.Projects {
			if _, ok := foldersMap[p.Parent]; ok {
				mqlP, err := projectToMql(g.MqlRuntime, p)
				if err != nil {
					return err
				}
				mqlProjects = append(mqlProjects, mqlP)
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return mqlProjects, nil
}

func projectToMql(runtime *plugin.Runtime, p *cloudresourcemanager.Project) (*mqlGcpProject, error) {
	res, err := CreateResource(runtime, "gcp.project", map[string]*llx.RawData{
		"id":         llx.StringData(p.ProjectId),
		"name":       llx.StringData(p.DisplayName),
		"parentId":   llx.StringData(p.Parent),
		"state":      llx.StringData(p.State),
		"createTime": llx.TimeDataPtr(parseTime(p.CreateTime)),
		"labels":     llx.MapData(convert.MapToInterfaceMap(p.Labels), types.String),
		"deleteTime": llx.TimeDataPtr(parseTime(p.DeleteTime)),
		"number":     llx.StringData(strings.TrimPrefix(p.Name, "projects/")),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlGcpProject), nil
}

func (g *mqlGcpProject) getEnabledServices() (map[string]struct{}, error) {
	g.enabledServicesOnce.Do(func() {
		services := make(map[string]struct{})
		enabledServices, err := g.fetchServices("state:ENABLED")
		if err != nil {
			g.enabledServicesErr = err
			return
		}

		for i := range enabledServices {
			srv := enabledServices[i].(*mqlGcpService)
			services[srv.Name.Data] = struct{}{}
		}
		// publish the fully-built map only after population so concurrent
		// readers never observe a partially-filled map
		g.enabledServices = services
	})

	return g.enabledServices, g.enabledServicesErr
}

// isServiceEnabled is an internal helper function to check if a service is serviceEnabled
func (g *mqlGcpProject) isServiceEnabled(serviceName string) (bool, error) {
	enabledServices, err := g.getEnabledServices()
	if err != nil {
		return false, err
	}

	if _, ok := enabledServices[serviceName]; ok {
		return true, nil
	}

	return false, nil
}
