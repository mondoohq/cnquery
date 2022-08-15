package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/organizations"
)

func (a *lumiAwsAccount) id() (string, error) {
	id, err := a.Id()
	if err != nil {
		return "", err
	}
	return "aws.account." + id, nil
}

func (a *lumiAwsAccount) GetId() (string, error) {
	at, err := awstransport(a.MotorRuntime.Motor.Transport)
	if err != nil {
		return "", nil
	}

	account, err := at.Account()
	if err != nil {
		return "", nil
	}

	return account.ID, nil
}

func (a *lumiAwsAccount) GetAliases() ([]interface{}, error) {
	at, err := awstransport(a.MotorRuntime.Motor.Transport)
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

func (a *lumiAwsAccount) GetOrganization() (interface{}, error) {
	at, err := awstransport(a.MotorRuntime.Motor.Transport)
	if err != nil {
		return nil, nil
	}

	client := organizations.NewFromConfig(at.Config())

	org, err := client.DescribeOrganization(context.TODO(), &organizations.DescribeOrganizationInput{})
	if err != nil {
		return nil, err
	}
	return a.MotorRuntime.CreateResource("aws.organization",
		"arn", toString(org.Organization.Arn),
		"featureSet", string(org.Organization.FeatureSet),
		"masterAccountId", toString(org.Organization.MasterAccountId),
		"masterAccountEmail", toString(org.Organization.MasterAccountEmail),
	)
}

func (a *lumiAwsOrganization) id() (string, error) {
	return a.Arn()
}
