package resources

func (a *lumiAwsAccount) id() (string, error) {
	id, err := a.Id()
	if err != nil {
		return "", err
	}
	return "aws.account." + id, nil
}

func (a *lumiAwsAccount) GetId() (string, error) {
	at, err := awstransport(a.Runtime.Motor.Transport)
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
	at, err := awstransport(a.Runtime.Motor.Transport)
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
