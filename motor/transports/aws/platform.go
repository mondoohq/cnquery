package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/rs/zerolog/log"
)

func (t *Transport) Identifier() (string, error) {
	return "//platformid.api.mondoo.app/runtime/aws/accounts/" + t.info.Account, nil
}

// Info returns the connection information
func (t *Transport) Info() Info {
	return t.info
}

type Account struct {
	ID   string
	Name string
}

func (t *Transport) Account() (Account, error) {
	accountid := t.info.Account
	ctx := context.Background()
	res, err := t.Iam().ListAccountAliasesRequest(&iam.ListAccountAliasesInput{}).Send(ctx)
	if err != nil {
		return Account{}, err
	}
	var accountName string
	if len(res.AccountAliases) == 0 {
		// if account has no alias, log a warning and use account id
		log.Warn().Msgf("no alias found for account %s", accountid)
		accountName = accountid
	} else {
		accountName = res.AccountAliases[0]
	}
	return Account{
		ID:   accountid,
		Name: accountName,
	}, nil
}
