package aws

import (
	"errors"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/aws/smithy-go/transport/http"
	"go.mondoo.com/cnquery/motor/providers"
	aws_provider "go.mondoo.com/cnquery/motor/providers/aws"
	"go.mondoo.com/cnquery/resources/packs/aws/info"
	"go.mondoo.com/cnquery/resources/packs/core"
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

func awsProvider(t providers.Instance) (*aws_provider.Provider, error) {
	provider, ok := t.(*aws_provider.Provider)
	if !ok {
		return nil, errors.New("aws resource is not supported on this provider; please run with -t aws")
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

func Is400AccessDeniedError(err error) bool {
	var respErr *http.ResponseError
	if errors.As(err, &respErr) {
		if respErr.HTTPStatusCode() == 400 && strings.Contains(respErr.Error(), "AccessDeniedException") {
			return true
		}
	}
	return false
}
