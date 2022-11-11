package msgraphclient

import (
	kioabs "github.com/microsoft/kiota-abstractions-go"
	kioser "github.com/microsoft/kiota-abstractions-go/serialization"
	kiojson "github.com/microsoft/kiota-serialization-json-go"
	kiotext "github.com/microsoft/kiota-serialization-text-go"
	"github.com/microsoftgraph/msgraph-beta-sdk-go/applications"
	"github.com/microsoftgraph/msgraph-beta-sdk-go/devicemanagement"
	"github.com/microsoftgraph/msgraph-beta-sdk-go/domains"
	domainitem "github.com/microsoftgraph/msgraph-beta-sdk-go/domains/item"
	"github.com/microsoftgraph/msgraph-beta-sdk-go/groups"
	"github.com/microsoftgraph/msgraph-beta-sdk-go/organization"
	"github.com/microsoftgraph/msgraph-beta-sdk-go/policies"
	"github.com/microsoftgraph/msgraph-beta-sdk-go/rolemanagement"
	"github.com/microsoftgraph/msgraph-beta-sdk-go/security"
	"github.com/microsoftgraph/msgraph-beta-sdk-go/serviceprincipals"
	"github.com/microsoftgraph/msgraph-beta-sdk-go/settings"
	"github.com/microsoftgraph/msgraph-beta-sdk-go/users"
	usersitem "github.com/microsoftgraph/msgraph-beta-sdk-go/users/item"
)

// NOTE:
// This code is almost verbatim from https://github.com/microsoftgraph/msgraph-beta-sdk-go
// We avoid using that package directly because it brings in a bunch of things we don't
// use

type GraphServiceClient struct {
	// Path parameters for the request
	pathParameters map[string]string
	// The request adapter to use to execute the requests.
	requestAdapter kioabs.RequestAdapter
	// Url template to use to build the URL for the current request builder
	urlTemplate string
}

func NewGraphServiceClient(requestAdapter kioabs.RequestAdapter) *GraphServiceClient {
	m := &GraphServiceClient{}
	m.pathParameters = make(map[string]string)
	m.urlTemplate = "{+baseurl}"
	m.requestAdapter = requestAdapter
	kioabs.RegisterDefaultSerializer(func() kioser.SerializationWriterFactory {
		return kiojson.NewJsonSerializationWriterFactory()
	})
	kioabs.RegisterDefaultSerializer(func() kioser.SerializationWriterFactory {
		return kiotext.NewTextSerializationWriterFactory()
	})
	kioabs.RegisterDefaultDeserializer(func() kioser.ParseNodeFactory {
		return kiojson.NewJsonParseNodeFactory()
	})
	kioabs.RegisterDefaultDeserializer(func() kioser.ParseNodeFactory {
		return kiotext.NewTextParseNodeFactory()
	})
	if m.requestAdapter.GetBaseUrl() == "" {
		m.requestAdapter.SetBaseUrl("https://graph.microsoft.com/beta")
	}
	return m
}

func (m *GraphServiceClient) Settings() *settings.SettingsRequestBuilder {
	return settings.NewSettingsRequestBuilderInternal(m.pathParameters, m.requestAdapter)
}

func (m *GraphServiceClient) Organization() *organization.OrganizationRequestBuilder {
	return organization.NewOrganizationRequestBuilderInternal(m.pathParameters, m.requestAdapter)
}

func (m *GraphServiceClient) Groups() *groups.GroupsRequestBuilder {
	return groups.NewGroupsRequestBuilderInternal(m.pathParameters, m.requestAdapter)
}

func (m *GraphServiceClient) ServicePrincipals() *serviceprincipals.ServicePrincipalsRequestBuilder {
	return serviceprincipals.NewServicePrincipalsRequestBuilderInternal(m.pathParameters, m.requestAdapter)
}

func (m *GraphServiceClient) Users() *users.UsersRequestBuilder {
	return users.NewUsersRequestBuilderInternal(m.pathParameters, m.requestAdapter)
}

func (m *GraphServiceClient) Domains() *domains.DomainsRequestBuilder {
	return domains.NewDomainsRequestBuilderInternal(m.pathParameters, m.requestAdapter)
}

func (m *GraphServiceClient) DomainsById(id string) *domainitem.DomainItemRequestBuilder {
	urlTplParams := make(map[string]string)
	for idx, item := range m.pathParameters {
		urlTplParams[idx] = item
	}
	if id != "" {
		urlTplParams["domain%2Did"] = id
	}
	return domainitem.NewDomainItemRequestBuilderInternal(urlTplParams, m.requestAdapter)
}

func (m *GraphServiceClient) Applications() *applications.ApplicationsRequestBuilder {
	return applications.NewApplicationsRequestBuilderInternal(m.pathParameters, m.requestAdapter)
}

func (m *GraphServiceClient) UsersById(id string) *usersitem.UserItemRequestBuilder {
	urlTplParams := make(map[string]string)
	for idx, item := range m.pathParameters {
		urlTplParams[idx] = item
	}
	if id != "" {
		urlTplParams["user%2Did"] = id
	}
	return usersitem.NewUserItemRequestBuilderInternal(urlTplParams, m.requestAdapter)
}

func (m *GraphServiceClient) Security() *security.SecurityRequestBuilder {
	return security.NewSecurityRequestBuilderInternal(m.pathParameters, m.requestAdapter)
}

func (m *GraphServiceClient) Policies() *policies.PoliciesRequestBuilder {
	return policies.NewPoliciesRequestBuilderInternal(m.pathParameters, m.requestAdapter)
}

func (m *GraphServiceClient) RoleManagement() *rolemanagement.RoleManagementRequestBuilder {
	return rolemanagement.NewRoleManagementRequestBuilderInternal(m.pathParameters, m.requestAdapter)
}

func (m *GraphServiceClient) DeviceManagement() *devicemanagement.DeviceManagementRequestBuilder {
	return devicemanagement.NewDeviceManagementRequestBuilderInternal(m.pathParameters, m.requestAdapter)
}
