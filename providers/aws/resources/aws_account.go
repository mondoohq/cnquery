// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/organizations"
	"go.mondoo.com/cnquery/v10/llx"
	"go.mondoo.com/cnquery/v10/providers/aws/connection"
)

func (a *mqlAwsAccount) id() (string, error) {
	if conn, ok := a.MqlRuntime.Connection.(*connection.AwsConnection); ok {
		return "aws.account/" + conn.AccountId(), nil
	}
	return "", errors.New("wrong connection for aws account id call")
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
