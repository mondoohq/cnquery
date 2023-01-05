package gcp

import (
	"context"

	"go.mondoo.com/cnquery/resources/packs/core"
	"google.golang.org/api/essentialcontacts/v1"
	"google.golang.org/api/option"
)

func (g *mqlGcpProject) GetEssentialContacts() (interface{}, error) {
	provider, err := gcpProvider(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	projectId, err := g.Id()
	if err != nil {
		return nil, err
	}

	client, err := provider.Client(essentialcontacts.CloudPlatformScope)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()

	contactSvc, err := essentialcontacts.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}

	contacts, err := contactSvc.Projects.Contacts.List("projects/" + projectId).Do()
	if err != nil {
		return nil, err
	}

	mqlContacts := make([]interface{}, 0, len(contacts.Contacts))
	for _, c := range contacts.Contacts {
		mqlC, err := g.MotorRuntime.CreateResource("gcp.essentialContact",
			"resourcePath", c.Name,
			"email", c.Email,
			"languageTag", c.LanguageTag,
			"notificationCategorySubscriptions", core.StrSliceToInterface(c.NotificationCategorySubscriptions),
			"validated", parseTime(c.ValidateTime),
			"validationState", c.ValidationState,
		)
		if err != nil {
			return nil, err
		}
		mqlContacts = append(mqlContacts, mqlC)
	}
	return mqlContacts, nil
}

func (g *mqlGcpEssentialContact) id() (string, error) {
	return g.ResourcePath()
}
