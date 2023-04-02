package okta

import (
	"context"

	"go.mondoo.com/cnquery/resources"
)

func (o *mqlOktaOrganization) id() (string, error) {
	id, err := o.Id()
	if err != nil {
		return "", err
	}
	return "okta/organization/" + id, nil
}

func (o *mqlOktaOrganization) init(args *resources.Args) (*resources.Args, OktaOrganization, error) {
	if len(*args) > 2 {
		return args, nil, nil
	}

	op, err := oktaProvider(o.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, nil, err
	}

	ctx := context.Background()
	client := op.Client()
	settings, _, err := client.OrgSetting.GetOrgSettings(ctx)
	if err != nil {
		return nil, nil, err
	}

	(*args)["id"] = settings.Id
	(*args)["companyName"] = settings.CompanyName
	(*args)["status"] = settings.Status
	(*args)["subdomain"] = settings.Subdomain
	(*args)["address1"] = settings.Address1
	(*args)["address2"] = settings.Address2
	(*args)["city"] = settings.City
	(*args)["state"] = settings.State
	(*args)["phoneNumber"] = settings.PhoneNumber
	(*args)["postalCode"] = settings.PostalCode
	(*args)["country"] = settings.Country
	(*args)["supportPhoneNumber"] = settings.SupportPhoneNumber
	(*args)["website"] = settings.Website
	(*args)["endUserSupportHelpURL"] = settings.EndUserSupportHelpURL
	(*args)["created"] = settings.Created
	(*args)["lastUpdated"] = settings.LastUpdated
	(*args)["expiresAt"] = settings.ExpiresAt

	return args, nil, nil
}

func (o *mqlOktaOrganization) GetOptOutCommunicationEmails() (bool, error) {
	op, err := oktaProvider(o.MotorRuntime.Motor.Provider)
	if err != nil {
		return false, err
	}

	ctx := context.Background()
	client := op.Client()
	settings, _, err := client.OrgSetting.GetOktaCommunicationSettings(ctx)
	if err != nil {
		return false, err
	}

	if settings.OptOutEmailUsers == nil {
		return *settings.OptOutEmailUsers, nil
	}

	return false, nil
}

func (o *mqlOktaOrganization) GetBillingContact() (interface{}, error) {
	op, err := oktaProvider(o.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	client := op.Client()
	contactUser, _, err := client.OrgSetting.GetOrgContactUser(ctx, "BILLING")
	if err != nil {
		return nil, err
	}
	uid := contactUser.UserId

	usr, _, err := client.User.GetUser(
		ctx,
		uid,
	)
	if err != nil {
		return nil, err
	}

	return newMqlOktaUser(o.MotorRuntime, usr)
}

func (o *mqlOktaOrganization) GetTechnicalContact() (interface{}, error) {
	op, err := oktaProvider(o.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	client := op.Client()
	contactUser, _, err := client.OrgSetting.GetOrgContactUser(ctx, "TECHNICAL")
	if err != nil {
		return nil, err
	}

	uid := contactUser.UserId

	usr, _, err := client.User.GetUser(
		ctx,
		uid,
	)
	if err != nil {
		return nil, err
	}

	return newMqlOktaUser(o.MotorRuntime, usr)
}
