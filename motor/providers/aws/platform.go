package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/rs/zerolog/log"
)

func (t *Provider) Identifier() (string, error) {
	return "//platformid.api.mondoo.app/runtime/aws/accounts/" + t.info.Account, nil
}

// Info returns the connection information
func (t *Provider) Info() Info {
	return t.info
}

type Account struct {
	ID      string
	Aliases []string
}

func (t *Provider) Account() (Account, error) {
	accountid := t.info.Account
	ctx := context.Background()
	res, err := t.Iam("").ListAccountAliases(ctx, &iam.ListAccountAliasesInput{})
	if err != nil {
		log.Warn().Err(err).Msg("unable to list account aliases")
		return Account{
			ID: accountid,
		}, nil
	}
	return Account{
		ID:      accountid,
		Aliases: res.AccountAliases,
	}, nil
}
