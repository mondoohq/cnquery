package aws

import (
	"errors"

	aws_sdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

//go:generate packr2 --ignore-imports

const MISSING_REGION_MSG = `
The AWS region must be set for the deployment. Please use environment variables 
or AWS profiles. Further details are available at:
- https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-envvars.html
- https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-profiles.html
`

// CheckRegion verifies that the config includes a region
func CheckRegion(cfg aws_sdk.Config) error {
	if len(cfg.Region) == 0 {
		return errors.New(MISSING_REGION_MSG)
	}
	return nil
}

func CheckIam(cfg aws_sdk.Config) (*sts.GetCallerIdentityOutput, error) {
	stsSvr := sts.New(cfg)
	req := stsSvr.GetCallerIdentityRequest(&sts.GetCallerIdentityInput{})
	resp, err := req.Send()
	if err != nil {
		return nil, err
	} else if resp.Account == nil || resp.UserId == nil {
		return nil, errors.New("could not read iam user")
	} else {
		return resp, nil
	}
}
