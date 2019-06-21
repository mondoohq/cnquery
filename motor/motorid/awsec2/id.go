package awsec2

// aws://ec2/v1/accounts/{account}/regions/{region}/instances/{instanceid}
func MondooInstanceID(account string, region string, instanceid string) string {
	return "//platformid.api.mondoo.app/runtime/aws/ec2/v1/accounts/" + account + "/regions/" + region + "/instances/" + instanceid
}
