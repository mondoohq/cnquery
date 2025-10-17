// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/account"
	"github.com/aws/aws-sdk-go-v2/service/account/types"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/organizations"
	"go.mondoo.com/cnquery/v12/llx"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v12/providers/aws/connection"
)

func (a *mqlAwsAccount) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAwsAccount) aliases() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	client := conn.Iam("") // no region for iam, use configured region

	res, err := client.ListAccountAliases(context.TODO(), &iam.ListAccountAliasesInput{})
	if err != nil {
		return nil, err
	}
	result := []any{}
	for i := range res.AccountAliases {
		result = append(result, res.AccountAliases[i])
	}
	return result, nil
}

func (a *mqlAwsAccount) organization() (*mqlAwsOrganization, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	client := conn.Organizations("") // no region for orgs, use configured region

	org, err := client.DescribeOrganization(context.TODO(), &organizations.DescribeOrganizationInput{})
	if err != nil {
		return nil, err
	}
	res, err := CreateResource(a.MqlRuntime, ResourceAwsOrganization,
		map[string]*llx.RawData{
			"arn":                llx.StringDataPtr(org.Organization.Arn),
			"id":                 llx.StringDataPtr(org.Organization.Id),
			"featureSet":         llx.StringData(string(org.Organization.FeatureSet)),
			"masterAccountId":    llx.StringDataPtr(org.Organization.MasterAccountId),
			"masterAccountEmail": llx.StringDataPtr(org.Organization.MasterAccountEmail),
		})
	return res.(*mqlAwsOrganization), err
}

func (a *mqlAwsOrganization) accounts() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	client := conn.Organizations("") // no region for orgs, use configured region

	orgAccounts, err := client.ListAccounts(context.TODO(), &organizations.ListAccountsInput{})
	if err != nil {
		return nil, err
	}
	accounts := []any{}
	for i := range orgAccounts.Accounts {
		account := orgAccounts.Accounts[i]
		res, err := CreateResource(a.MqlRuntime, ResourceAwsAccount,
			map[string]*llx.RawData{
				"id": llx.StringDataPtr(account.Id),
			})
		if err != nil {
			return nil, err
		}
		accounts = append(accounts, res.(*mqlAwsAccount))
	}
	return accounts, nil
}

// tags retrieves a map of tags for a given AWS resource.
func (c *mqlAwsAccount) tags() (map[string]any, error) {
	conn := c.MqlRuntime.Connection.(*connection.AwsConnection)
	client := conn.Organizations("") // no region for orgs, use configured region

	input := &organizations.ListTagsForResourceInput{
		ResourceId: &c.Id.Data,
	}

	// Note: This operation can only be called from the organization's management
	// account or by a member account that is a delegated administrator for an
	// Amazon Web Services service.
	tags := make(map[string]any)
	paginator := organizations.NewListTagsForResourcePaginator(client, input)
	for paginator.HasMorePages() {
		res, err := paginator.NextPage(context.Background())
		if err != nil {
			return nil, err
		}

		for _, tag := range res.Tags {
			tags[*tag.Key] = *tag.Value
		}
	}

	return tags, nil
}

func (a *mqlAwsAccount) contactInformation() (map[string]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	client := conn.Account("") // no region for account service, use configured region

	resp, err := client.GetContactInformation(context.TODO(), &account.GetContactInformationInput{})
	if err != nil {
		return nil, err
	}

	// Convert the contact information to map[string]any
	result := make(map[string]any)
	if resp.ContactInformation != nil {
		if resp.ContactInformation.AddressLine1 != nil {
			result["addressLine1"] = *resp.ContactInformation.AddressLine1
		}
		if resp.ContactInformation.AddressLine2 != nil {
			result["addressLine2"] = *resp.ContactInformation.AddressLine2
		}
		if resp.ContactInformation.AddressLine3 != nil {
			result["addressLine3"] = *resp.ContactInformation.AddressLine3
		}
		if resp.ContactInformation.City != nil {
			result["city"] = *resp.ContactInformation.City
		}
		if resp.ContactInformation.CompanyName != nil {
			result["companyName"] = *resp.ContactInformation.CompanyName
		}
		if resp.ContactInformation.CountryCode != nil {
			result["countryCode"] = *resp.ContactInformation.CountryCode
		}
		if resp.ContactInformation.DistrictOrCounty != nil {
			result["districtOrCounty"] = *resp.ContactInformation.DistrictOrCounty
		}
		if resp.ContactInformation.FullName != nil {
			result["fullName"] = *resp.ContactInformation.FullName
		}
		if resp.ContactInformation.PhoneNumber != nil {
			result["phoneNumber"] = *resp.ContactInformation.PhoneNumber
		}
		if resp.ContactInformation.PostalCode != nil {
			result["postalCode"] = *resp.ContactInformation.PostalCode
		}
		if resp.ContactInformation.StateOrRegion != nil {
			result["stateOrRegion"] = *resp.ContactInformation.StateOrRegion
		}
		if resp.ContactInformation.WebsiteUrl != nil {
			result["websiteUrl"] = *resp.ContactInformation.WebsiteUrl
		}
	}
	return result, nil
}

func (a *mqlAwsAccount) alternateContacts() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	client := conn.Account("") // no region for account service, use configured region

	contactTypes := []types.AlternateContactType{
		types.AlternateContactTypeBilling,
		types.AlternateContactTypeOperations,
		types.AlternateContactTypeSecurity,
	}

	var contacts []any

	for _, cType := range contactTypes {
		resp, err := client.GetAlternateContact(context.TODO(), &account.GetAlternateContactInput{
			AlternateContactType: cType,
		})

		if err != nil {
			// Check if contact not configured (ResourceNotFoundException)
			var notFoundErr *types.ResourceNotFoundException
			if errors.As(err, &notFoundErr) {
				// Contact not configured - create resource with exists=false
				contact, resErr := CreateResource(a.MqlRuntime, ResourceAwsAccountAlternateContact,
					map[string]*llx.RawData{
						"accountId":    llx.StringData(a.Id.Data),
						"contactType":  llx.StringData(string(cType)),
						"emailAddress": llx.StringData(""),
						"name":         llx.StringData(""),
						"phoneNumber":  llx.StringData(""),
						"title":        llx.StringData(""),
						"exists":       llx.BoolData(false),
					})
				if resErr != nil {
					return nil, resErr
				}
				contacts = append(contacts, contact)
				continue
			}
			// Other error - return it
			return nil, err
		}

		// Contact is configured - create resource with data
		contact, err := CreateResource(a.MqlRuntime, ResourceAwsAccountAlternateContact,
			map[string]*llx.RawData{
				"accountId":    llx.StringData(a.Id.Data),
				"contactType":  llx.StringData(string(cType)),
				"emailAddress": llx.StringData(aws.ToString(resp.AlternateContact.EmailAddress)),
				"name":         llx.StringData(aws.ToString(resp.AlternateContact.Name)),
				"phoneNumber":  llx.StringData(aws.ToString(resp.AlternateContact.PhoneNumber)),
				"title":        llx.StringData(aws.ToString(resp.AlternateContact.Title)),
				"exists":       llx.BoolData(true),
			})
		if err != nil {
			return nil, err
		}
		contacts = append(contacts, contact)
	}

	return contacts, nil
}

func (a *mqlAwsAccount) securityContact() (*mqlAwsAccountAlternateContact, error) {
	contacts, err := a.alternateContacts()
	if err != nil {
		return nil, err
	}

	for _, c := range contacts {
		contact := c.(*mqlAwsAccountAlternateContact)
		if contact.ContactType.Data == string(types.AlternateContactTypeSecurity) {
			return contact, nil
		}
	}

	// Should not happen as alternateContacts() always returns all three types
	return nil, errors.New("security contact not found")
}

func (a *mqlAwsAccount) billingContact() (*mqlAwsAccountAlternateContact, error) {
	contacts, err := a.alternateContacts()
	if err != nil {
		return nil, err
	}

	for _, c := range contacts {
		contact := c.(*mqlAwsAccountAlternateContact)
		if contact.ContactType.Data == string(types.AlternateContactTypeBilling) {
			return contact, nil
		}
	}

	// Should not happen as alternateContacts() always returns all three types
	return nil, errors.New("billing contact not found")
}

func (a *mqlAwsAccount) operationsContact() (*mqlAwsAccountAlternateContact, error) {
	contacts, err := a.alternateContacts()
	if err != nil {
		return nil, err
	}

	for _, c := range contacts {
		contact := c.(*mqlAwsAccountAlternateContact)
		if contact.ContactType.Data == string(types.AlternateContactTypeOperations) {
			return contact, nil
		}
	}

	// Should not happen as alternateContacts() always returns all three types
	return nil, errors.New("operations contact not found")
}

func initAwsAccount(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) >= 2 {
		return args, nil, nil
	}

	if len(args) == 0 {
		conn := runtime.Connection.(*connection.AwsConnection)
		args["id"] = llx.StringData(conn.AccountId())
	}
	if args["id"] == nil {
		return args, nil, errors.New("no account id specified")
	}
	id := args["id"].Value.(string)
	res, err := CreateResource(runtime, ResourceAwsAccount,
		map[string]*llx.RawData{
			"id": llx.StringData(id),
		})
	if err != nil {
		return nil, nil, err
	}
	return args, res, nil
}

func (a *mqlAwsAccountAlternateContact) id() (string, error) {
	return a.AccountId.Data + "/" + a.ContactType.Data, nil
}

func initAwsAccountAlternateContact(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	return args, nil, nil
}
