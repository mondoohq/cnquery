package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/organizations"
	"go.mondoo.io/mondoo/resources/packs/core"
)

func (a *mqlAwsAccount) id() (string, error) {
	id, err := a.Id()
	if err != nil {
		return "", err
	}
	return "aws.account." + id, nil
}

func (a *mqlAwsAccount) GetId() (string, error) {
	at, err := awsProvider(a.MotorRuntime.Motor.Provider)
	if err != nil {
		return "", nil
	}

	account, err := at.Account()
	if err != nil {
		return "", nil
	}

	return account.ID, nil
}

func (a *mqlAwsAccount) GetAliases() ([]interface{}, error) {
	at, err := awsProvider(a.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, nil
	}

	account, err := at.Account()
	if err != nil {
		return nil, nil
	}

	res := []interface{}{}

	for i := range account.Aliases {
		res = append(res, account.Aliases[i])
	}

	return res, nil
}

func (a *mqlAwsAccount) GetOrganization() (interface{}, error) {
	at, err := awsProvider(a.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, nil
	}

	client := organizations.NewFromConfig(at.Config())

	org, err := client.DescribeOrganization(context.TODO(), &organizations.DescribeOrganizationInput{})
	if err != nil {
		return nil, err
	}
	return a.MotorRuntime.CreateResource("aws.organization",
		"arn", core.ToString(org.Organization.Arn),
		"featureSet", string(org.Organization.FeatureSet),
		"masterAccountId", core.ToString(org.Organization.MasterAccountId),
		"masterAccountEmail", core.ToString(org.Organization.MasterAccountEmail),
	)
}

func (a *mqlAwsOrganization) id() (string, error) {
	return a.Arn()
}
