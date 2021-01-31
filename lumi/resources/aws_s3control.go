package resources

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3control"
	"github.com/aws/aws-sdk-go-v2/service/s3control/types"
)

func (s *lumiAwsS3control) id() (string, error) {
	return "aws.s3control", nil
}

func (s *lumiAwsS3control) GetAccountPublicAccessBlock() (interface{}, error) {
	at, err := awstransport(s.Runtime.Motor.Transport)
	if err != nil {
		return nil, err
	}

	account, err := at.Account()
	if err != nil {
		return nil, err
	}

	svc := at.S3Control("")
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

	return jsonToDict(publicAccessBlock.PublicAccessBlockConfiguration)
}
