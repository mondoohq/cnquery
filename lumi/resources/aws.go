package resources

import (
	"errors"

	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"go.mondoo.io/mondoo/motor/transports"
	aws_transport "go.mondoo.io/mondoo/motor/transports/aws"
)

func (e *lumiAws) id() (string, error) {
	return "aws", nil
}

func (s *lumiAws) GetRegions() ([]interface{}, error) {
	at, err := awstransport(s.Runtime.Motor.Transport)
	if err != nil {
		return nil, err
	}
	regions, err := at.GetRegions()
	if err != nil {
		return nil, err
	}
	res := make([]interface{}, len(regions))
	for i := range regions {
		res[i] = regions[i]
	}
	return res, nil
}

func awstransport(t transports.Transport) (*aws_transport.Transport, error) {
	at, ok := t.(*aws_transport.Transport)
	if !ok {
		return nil, errors.New("aws resource is not supported on this transport")
	}
	return at, nil
}

func toString(i *string) string {
	if i == nil {
		return ""
	}
	return *i
}

func toBool(i *bool) bool {
	if i == nil {
		return false
	}
	return *i
}

func toInt64(i *int64) int64 {
	if i == nil {
		return 0
	}
	return *i
}

func toInt(i *int) int64 {
	if i == nil {
		return int64(0)
	}
	return int64(*i)
}

func getRegionFromArn(arnVal string) (string, error) {
	parsedArn, err := arn.Parse(arnVal)
	if err != nil {
		return "", err
	}
	return parsedArn.Region, nil
}
