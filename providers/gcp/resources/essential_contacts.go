// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/gcp/connection"
	"go.mondoo.com/mql/v13/types"

	"google.golang.org/api/essentialcontacts/v1"
	"google.golang.org/api/option"
)

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
	if err := contactSvc.Projects.Contacts.List("projects/"+projectId).Pages(ctx, func(page *essentialcontacts.GoogleCloudEssentialcontactsV1ListContactsResponse) error {
		for _, c := range page.Contacts {
			mqlC, err := CreateResource(g.MqlRuntime, "gcp.essentialContact", map[string]*llx.RawData{
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
	}); err != nil {
		return nil, err
	}
	return mqlContacts, nil
}

func (g *mqlGcpEssentialContact) id() (string, error) {
	return g.ResourcePath.Data, g.ResourcePath.Error
}
