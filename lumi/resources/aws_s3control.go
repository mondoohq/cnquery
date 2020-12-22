package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3control"
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

	publicAccessBlock, err := svc.GetPublicAccessBlockRequest(&s3control.GetPublicAccessBlockInput{
		AccountId: aws.String(account.ID),
	}).Send(ctx)
	isAwsErr, code := IsAwsCode(err)
	if err != nil && isAwsErr && code == "NoSuchPublicAccessBlockConfiguration" {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	return jsonToDict(publicAccessBlock.PublicAccessBlockConfiguration)
}
