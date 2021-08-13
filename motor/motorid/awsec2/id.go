package awsec2

import (
	"regexp"
	"strings"

	"github.com/cockroachdb/errors"
)

// aws://ec2/v1/accounts/{account}/regions/{region}/instances/{instanceid}
func MondooInstanceID(account string, region string, instanceid string) string {
	return "//platformid.api.mondoo.app/runtime/aws/ec2/v1/accounts/" + account + "/regions/" + region + "/instances/" + instanceid
}

type MondooInstanceId struct {
	Account string
	Region  string
	Id      string
}

var VALID_MONDOO_INSTANCE_ID = regexp.MustCompile(`^//platformid.api.mondoo.app/runtime/aws/ec2/v1/accounts/\d{12}/regions\/(us(-gov)?|ap|ca|cn|eu|sa)-(central|(north|south)?(east|west)?)-\d\/instances\/.+$`)

func ParseMondooInstanceID(path string) (*MondooInstanceId, error) {
	if !IsValidMondooInstanceId(path) {
		return nil, errors.New("invalid mondoo aws ec2 instance id")
	}
	keyValues := strings.Split(path, "/")
	if len(keyValues) != 13 {
		return nil, errors.New("invalid instance id")
	}
	return &MondooInstanceId{Account: keyValues[8], Region: keyValues[10], Id: keyValues[12]}, nil
}

func IsValidMondooInstanceId(path string) bool {
	return VALID_MONDOO_INSTANCE_ID.MatchString(path)
}
