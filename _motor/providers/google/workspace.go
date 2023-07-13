package google

import (
	"context"

	directory "google.golang.org/api/admin/directory/v1"
	reports "google.golang.org/api/admin/reports/v1"
	cloudidentity "google.golang.org/api/cloudidentity/v1"
	"google.golang.org/api/option"
)

var DefaultWorkspaceClientScopes = []string{
	directory.AdminChromePrintersReadonlyScope,
	directory.AdminDirectoryCustomerReadonlyScope,
	directory.AdminDirectoryDeviceChromeosReadonlyScope,
	directory.AdminDirectoryDeviceMobileReadonlyScope,
	directory.AdminDirectoryDomainReadonlyScope,
	directory.AdminDirectoryGroupMemberReadonlyScope,
	directory.AdminDirectoryGroupReadonlyScope,
	directory.AdminDirectoryOrgunitReadonlyScope,
	directory.AdminDirectoryResourceCalendarReadonlyScope,
	directory.AdminDirectoryRolemanagementReadonlyScope,
	directory.AdminDirectoryUserAliasReadonlyScope,
	directory.AdminDirectoryUserReadonlyScope,
	directory.AdminDirectoryUserschemaReadonlyScope,
	directory.AdminDirectoryUserSecurityScope,
	reports.AdminReportsAuditReadonlyScope,
	reports.AdminReportsUsageReadonlyScope,
	cloudidentity.CloudIdentityGroupsReadonlyScope,
}

func (p *Provider) GetCustomerID() string {
	return p.id
}

func (p *Provider) GetWorkspaceCustomer(customerID string) (*directory.Customer, error) {
	client, err := p.Client(directory.AdminDirectoryCustomerReadonlyScope)
	if err != nil {
		return nil, err
	}

	service, err := directory.NewService(context.Background(), option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}
	return service.Customers.Get(customerID).Do()
}
