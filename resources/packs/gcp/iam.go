package gcp

import (
	"context"
	"fmt"

	admin "cloud.google.com/go/iam/admin/apiv1"
	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/resources/packs/core"
	"google.golang.org/api/cloudresourcemanager/v1"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	adminpb "google.golang.org/genproto/googleapis/iam/admin/v1"
)

func (g *mqlGcpIamPolicy) id() (string, error) {
	return g.Id()
}

func (g *mqlGcpProjectIamService) id() (string, error) {
	projectId, err := g.ProjectId()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/gcp.project.iamService", projectId), nil
}

func (g *mqlGcpProject) GetIam() (interface{}, error) {
	projectId, err := g.Id()
	if err != nil {
		return nil, err
	}

	return g.MotorRuntime.CreateResource("gcp.project.iamService",
		"projectId", projectId,
	)
}

func (g *mqlGcpProjectIamServiceServiceAccount) id() (string, error) {
	return g.UniqueId()
}

func (g *mqlGcpProjectIamServiceServiceAccountKey) id() (string, error) {
	return g.Name()
}

func (g *mqlGcpProjectIamService) GetServiceAccounts() ([]interface{}, error) {
	projectId, err := g.ProjectId()
	if err != nil {
		return nil, err
	}

	provider, err := gcpProvider(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	creds, err := provider.Credentials(admin.DefaultAuthScopes()...)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()

	adminSvc, err := admin.NewIamClient(ctx, option.WithCredentials(creds))
	if err != nil {
		return nil, err
	}
	defer adminSvc.Close()

	var serviceAccounts []interface{}
	it := adminSvc.ListServiceAccounts(ctx, &adminpb.ListServiceAccountsRequest{Name: fmt.Sprintf("projects/%s", projectId)})
	for {
		s, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		mqlSA, err := g.MotorRuntime.CreateResource("gcp.project.iamService.serviceAccount",
			"projectId", s.ProjectId,
			"name", s.Name,
			"uniqueId", s.UniqueId,
			"email", s.Email,
			"displayName", s.DisplayName,
			"description", s.Description,
			"oauth2ClientId", s.Oauth2ClientId,
			"disabled", s.Disabled,
		)
		if err != nil {
			return nil, err
		}
		serviceAccounts = append(serviceAccounts, mqlSA)
	}
	return serviceAccounts, nil
}

func (g *mqlGcpProjectIamServiceServiceAccount) GetKeys() ([]interface{}, error) {
	projectId, err := g.ProjectId()
	if err != nil {
		return nil, err
	}

	email, err := g.Email()
	if err != nil {
		return nil, err
	}

	provider, err := gcpProvider(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	creds, err := provider.Credentials(admin.DefaultAuthScopes()...)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()

	adminSvc, err := admin.NewIamClient(ctx, option.WithCredentials(creds))
	if err != nil {
		return nil, err
	}
	defer adminSvc.Close()

	resp, err := adminSvc.ListServiceAccountKeys(ctx, &adminpb.ListServiceAccountKeysRequest{Name: fmt.Sprintf("projects/%s/serviceAccounts/%s", projectId, email)})
	if err != nil {
		return nil, err
	}
	mqlKeys := make([]interface{}, 0, len(resp.Keys))
	for _, k := range resp.Keys {
		mqlKey, err := g.MotorRuntime.CreateResource("gcp.project.iamService.serviceAccount.key",
			"name", k.Name,
			"keyAlgorithm", k.KeyAlgorithm.String(),
			"validAfterTime", timestampAsTimePtr(k.ValidAfterTime),
			"validBeforeTime", timestampAsTimePtr(k.ValidBeforeTime),
			"keyOrigin", k.KeyOrigin.String(),
			"keyType", k.KeyType.String(),
			"disabled", k.Disabled,
		)
		if err != nil {
			return nil, err
		}
		mqlKeys = append(mqlKeys, mqlKey)
	}
	return mqlKeys, nil
}

func auditConfigsToMql(runtime *resources.Runtime, auditCfgs []*cloudresourcemanager.AuditConfig, idPrefix string) ([]interface{}, error) {
	mqlAuditCfgs := make([]interface{}, 0, len(auditCfgs))
	for _, a := range auditCfgs {
		cfgs := make([]interface{}, 0, len(a.AuditLogConfigs))
		for _, c := range a.AuditLogConfigs {
			cfgs = append(cfgs, map[string]interface{}{
				"exemptedMembers": core.StrSliceToInterface(c.ExemptedMembers),
				"logType":         c.LogType,
			})
		}
		mqlAuditCfgs = append(mqlAuditCfgs, map[string]interface{}{
			"auditLogConfigs": cfgs,
			"service":         a.Service,
		})
	}
	return mqlAuditCfgs, nil
}

func bindingsToMql(runtime *resources.Runtime, bindings []*cloudresourcemanager.Binding, idPrefix string) ([]interface{}, error) {
	mqlBindings := make([]interface{}, 0, len(bindings))
	for i, b := range bindings {
		mqlServiceaccount, err := runtime.CreateResource("gcp.iamPolicy.binding",
			"id", fmt.Sprintf("%s/%d", idPrefix, i),
			"role", b.Role,
			"members", core.StrSliceToInterface(b.Members),
		)
		if err != nil {
			return nil, err
		}
		mqlBindings = append(mqlBindings, mqlServiceaccount)
	}
	return mqlBindings, nil
}
