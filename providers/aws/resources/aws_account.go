package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/organizations"
	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/providers/aws/connection"
	"go.mondoo.com/cnquery/providers/aws/utils"
)

func (a *mqlAwsAccount) id() (string, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	return "aws.account/" + conn.AccountId(), nil
}

func (s *mqlAwsAccount) aliases() ([]interface{}, error) {
	conn := s.MqlRuntime.Connection.(*connection.AwsConnection)
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

func (s *mqlAwsAccount) organization() (*mqlAwsOrganization, error) {
	conn := s.MqlRuntime.Connection.(*connection.AwsConnection)
	client := conn.Organizations("") // no region for orgs, use configured region

	org, err := client.DescribeOrganization(context.TODO(), &organizations.DescribeOrganizationInput{})
	if err != nil {
		return nil, err
	}
	res, err := s.MqlRuntime.CreateResource(s.MqlRuntime, "aws.organization",
		map[string]*llx.RawData{
			"arn":                llx.StringData(utils.ToString(org.Organization.Arn)),
			"featureSet":         llx.StringData(string(org.Organization.FeatureSet)),
			"masterAccountId":    llx.StringData(utils.ToString(org.Organization.MasterAccountId)),
			"masterAccountEmail": llx.StringData(utils.ToString(org.Organization.MasterAccountEmail)),
		})
	return res.(*mqlAwsOrganization), err
}
