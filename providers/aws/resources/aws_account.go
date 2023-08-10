package resources

import (
	"go.mondoo.com/cnquery/providers/aws/connection"
)

func (a *mqlAwsAccount) id() (string, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	return "aws.account/" + conn.AccountId(), nil
}
