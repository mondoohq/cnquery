// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/organizations"
	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers/aws/connection"
)

func (a *mqlAwsAccount) id() (string, error) {
	return "aws.account/" + a.Id.Data, nil
}

func (a *mqlAwsAccount) aliases() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	client := conn.Iam("") // no region for iam, use configured region

	res, err := client.ListAccountAliases(context.TODO(), &iam.ListAccountAliasesInput{})
	if err != nil {
		return nil, err
	}
	result := []interface{}{}
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
	res, err := CreateResource(a.MqlRuntime, "aws.organization",
		map[string]*llx.RawData{
			"arn":                llx.StringDataPtr(org.Organization.Arn),
			"featureSet":         llx.StringData(string(org.Organization.FeatureSet)),
			"masterAccountId":    llx.StringDataPtr(org.Organization.MasterAccountId),
			"masterAccountEmail": llx.StringDataPtr(org.Organization.MasterAccountEmail),
		})
	return res.(*mqlAwsOrganization), err
}

func (a *mqlAwsOrganization) accounts() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	client := conn.Organizations("") // no region for orgs, use configured region

	orgAccounts, err := client.ListAccounts(context.TODO(), &organizations.ListAccountsInput{})
	if err != nil {
		return nil, err
	}
	accounts := []interface{}{}
	for i := range orgAccounts.Accounts {
		account := orgAccounts.Accounts[i]
		res, err := CreateResource(a.MqlRuntime, "aws.account",
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

// Method to list tags for the account
func (c *mqlAwsAccount) tags() (map[string]interface{}, error) {
	conn := c.MqlRuntime.Connection.(*connection.AwsConnection)
	client := conn.Organizations("") // no region for orgs, use configured region

	input := &organizations.ListTagsForResourceInput{
		ResourceId: &c.Id.Data,
	}

	tags := make(map[string]interface{})
	for {
		res, err := client.ListTagsForResource(context.TODO(), input)
		if err != nil {
			return nil, err
		}

		for _, tag := range res.Tags {
			tags[*tag.Key] = *tag.Value
		}

		if res.NextToken == nil {
			break
		}
		input.NextToken = res.NextToken
	}

	return tags, nil
}

func initAwsAccount(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) >= 2 {
		return args, nil, nil
	}
	if len(args) == 0 {
		if ids := getAssetIdentifier(runtime); ids != nil {
			id := strings.TrimPrefix(ids.arn, "arn:aws:sts::")
			args["id"] = llx.StringData(id)
		}
	}
	if args["id"] == nil {
		return args, nil, errors.New("no account id specified")
	}
	id := args["id"].Value.(string)
	res, err := CreateResource(runtime, "aws.account",
		map[string]*llx.RawData{
			"id": llx.StringData(id),
		})
	if err != nil {
		return nil, nil, err
	}
	return args, res, nil
}
