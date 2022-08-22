package aws

import (
	"errors"

	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"go.mondoo.io/mondoo/motor/providers"
	aws_provider "go.mondoo.io/mondoo/motor/providers/aws"
	"go.mondoo.io/mondoo/resources/packs/aws/info"
	"go.mondoo.io/mondoo/resources/packs/core"
)

var Registry = info.Registry

func init() {
	Init(Registry)
	Registry.Add(core.Registry)
}

func (e *mqlAws) id() (string, error) {
	return "aws", nil
}

func (s *mqlAws) GetRegions() ([]interface{}, error) {
	provider, err := awsProvider(s.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	regions, err := provider.GetRegions()
	if err != nil {
		return nil, err
	}
	res := make([]interface{}, len(regions))
	for i := range regions {
		res[i] = regions[i]
	}
	return res, nil
}

func awsProvider(t providers.Transport) (*aws_provider.Provider, error) {
	provider, ok := t.(*aws_provider.Provider)
	if !ok {
		return nil, errors.New("aws resource is not supported on this transport; please run with -t aws")
	}
	return provider, nil
}

func GetRegionFromArn(arnVal string) (string, error) {
	parsedArn, err := arn.Parse(arnVal)
	if err != nil {
		return "", err
	}
	return parsedArn.Region, nil
}
