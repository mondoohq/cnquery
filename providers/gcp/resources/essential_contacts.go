// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"strings"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/gcp/connection"
	"go.mondoo.com/mql/v13/types"

	"google.golang.org/api/essentialcontacts/v1"
	"google.golang.org/api/option"
)

// essentialContactsForParent lists Essential Contacts for a resource parent of
// the form "projects/{id}", "folders/{id}", or "organizations/{id}" and builds
// the corresponding gcp.essentialContact resources.
func essentialContactsForParent(runtime *plugin.Runtime, conn *connection.GcpConnection, parent string) ([]any, error) {
	client, err := conn.Client(essentialcontacts.CloudPlatformScope)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	contactSvc, err := essentialcontacts.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}

	var mqlContacts []any
	process := func(page *essentialcontacts.GoogleCloudEssentialcontactsV1ListContactsResponse) error {
		for _, c := range page.Contacts {
			mqlC, err := CreateResource(runtime, "gcp.essentialContact", map[string]*llx.RawData{
				"resourcePath":           llx.StringData(c.Name),
				"email":                  llx.StringData(c.Email),
				"languageTag":            llx.StringData(c.LanguageTag),
				"notificationCategories": llx.ArrayData(convert.SliceAnyToInterface(c.NotificationCategorySubscriptions), types.String),
				"validated":              llx.TimeDataPtr(parseTime(c.ValidateTime)),
				"validationState":        llx.StringData(c.ValidationState),
			})
			if err != nil {
				return err
			}
			mqlContacts = append(mqlContacts, mqlC)
		}
		return nil
	}

	switch {
	case strings.HasPrefix(parent, "organizations/"):
		err = contactSvc.Organizations.Contacts.List(parent).Pages(ctx, process)
	case strings.HasPrefix(parent, "folders/"):
		err = contactSvc.Folders.Contacts.List(parent).Pages(ctx, process)
	default:
		err = contactSvc.Projects.Contacts.List(parent).Pages(ctx, process)
	}
	if err != nil {
		return nil, err
	}
	return mqlContacts, nil
}

func (g *mqlGcpProject) essentialContacts() ([]any, error) {
	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)

	if g.Id.Error != nil {
		return nil, g.Id.Error
	}
	projectId := g.Id.Data

	serviceEnabled, err := g.isServiceEnabled(service_essential_contacts)
	if err != nil {
		return nil, err
	}
	if !serviceEnabled {
		return nil, nil
	}

	return essentialContactsForParent(g.MqlRuntime, conn, "projects/"+projectId)
}

func (g *mqlGcpOrganization) essentialContacts() ([]any, error) {
	if g.Id.Error != nil {
		return nil, g.Id.Error
	}
	// Id is already in "organizations/{id}" form (see initGcpOrganization).
	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)
	return essentialContactsForParent(g.MqlRuntime, conn, g.Id.Data)
}

func (g *mqlGcpFolder) essentialContacts() ([]any, error) {
	if g.Id.Error != nil {
		return nil, g.Id.Error
	}
	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)
	return essentialContactsForParent(g.MqlRuntime, conn, "folders/"+g.Id.Data)
}

func (g *mqlGcpEssentialContact) id() (string, error) {
	return g.ResourcePath.Data, g.ResourcePath.Error
}
