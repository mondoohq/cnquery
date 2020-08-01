package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/organizations"
)

func (t *Transport) Identifier() (string, error) {
	return "//platformid.api.mondoo.app/runtime/aws/accounts/" + t.info.Account, nil
}

// Info returns the connection information
func (t *Transport) Info() Info {
	return t.info
}

type Account struct {
	ID     string
	Name   string
	Arn    string
	Status string
}

func (t *Transport) Account() (Account, error) {
	accountid := t.info.Account
	// get account id
	ctx := context.Background()
	res, err := t.Organizations().DescribeAccountRequest(&organizations.DescribeAccountInput{AccountId: &accountid}).Send(ctx)
	if err != nil {
		return Account{}, err
	}
	return Account{
		ID:     toString(res.Account.Id),
		Name:   toString(res.Account.Name),
		Arn:    toString(res.Account.Arn),
		Status: string(res.Account.Status),
	}, nil
}
