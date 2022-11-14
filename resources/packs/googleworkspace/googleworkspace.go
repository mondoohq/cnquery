package googleworkspace

import (
	"context"
	"errors"

	"go.mondoo.com/cnquery/motor/providers"
	google_provider "go.mondoo.com/cnquery/motor/providers/google"
	"go.mondoo.com/cnquery/resources/packs/googleworkspace/info"
	directory "google.golang.org/api/admin/directory/v1"
	reports "google.golang.org/api/admin/reports/v1"
	cloudidentity "google.golang.org/api/cloudidentity/v1"
	groupssettings "google.golang.org/api/groupssettings/v1"
	"google.golang.org/api/option"
)

var Registry = info.Registry

func init() {
	Init(Registry)
}

func (k *mqlGoogleworkspace) id() (string, error) {
	return "googleworkspace", nil
}

func workspaceProvider(p providers.Instance) (*google_provider.Provider, error) {
	gwp, ok := p.(*google_provider.Provider)
	if !ok {
		return nil, errors.New("okta resource is not supported on this provider")
	}
	return gwp, nil
}

func reportsService(p providers.Instance) (*google_provider.Provider, *reports.Service, error) {
	provider, err := workspaceProvider(p)
	if err != nil {
		return nil, nil, err
	}

	client, err := provider.Client(reports.AdminReportsAuditReadonlyScope, reports.AdminReportsUsageReadonlyScope)
	if err != nil {
		return nil, nil, err
	}

	service, err := reports.NewService(context.Background(), option.WithHTTPClient(client))
	return provider, service, err
}

func directoryService(p providers.Instance, scopes ...string) (*google_provider.Provider, *directory.Service, error) {
	provider, err := workspaceProvider(p)
	if err != nil {
		return nil, nil, err
	}

	client, err := provider.Client(scopes...)
	if err != nil {
		return nil, nil, err
	}

	directoryService, err := directory.NewService(context.Background(), option.WithHTTPClient(client))
	return provider, directoryService, err
}

func cloudIdentityService(p providers.Instance, scopes ...string) (*google_provider.Provider, *cloudidentity.Service, error) {
	provider, err := workspaceProvider(p)
	if err != nil {
		return nil, nil, err
	}

	client, err := provider.Client(scopes...)
	if err != nil {
		return nil, nil, err
	}

	cloudIdentityService, err := cloudidentity.NewService(context.Background(), option.WithHTTPClient(client))
	return provider, cloudIdentityService, err
}

func groupSettingsService(p providers.Instance, scopes ...string) (*google_provider.Provider, *groupssettings.Service, error) {
	provider, err := workspaceProvider(p)
	if err != nil {
		return nil, nil, err
	}

	client, err := provider.Client(scopes...)
	if err != nil {
		return nil, nil, err
	}

	groupssettingsService, err := groupssettings.NewService(context.Background(), option.WithHTTPClient(client))
	return provider, groupssettingsService, err
}
