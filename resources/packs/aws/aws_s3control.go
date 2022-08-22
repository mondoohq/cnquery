package aws

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3control"
	"github.com/aws/aws-sdk-go-v2/service/s3control/types"
	"go.mondoo.io/mondoo/resources/packs/core"
)

func (s *mqlAwsS3control) id() (string, error) {
	return "aws.s3control", nil
}

func (s *mqlAwsS3control) GetAccountPublicAccessBlock() (interface{}, error) {
	provider, err := awsProvider(s.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	account, err := provider.Account()
	if err != nil {
		return nil, err
	}

	svc := provider.S3Control("")
	ctx := context.Background()

	publicAccessBlock, err := svc.GetPublicAccessBlock(ctx, &s3control.GetPublicAccessBlockInput{
		AccountId: aws.String(account.ID),
	})
	if err != nil {
		var notFoundErr *types.NoSuchPublicAccessBlockConfiguration
		if errors.As(err, &notFoundErr) {
			return nil, nil
		}
		return nil, err
	}

	return core.JsonToDict(publicAccessBlock.PublicAccessBlockConfiguration)
}
