package awsec2

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/ec2metadata"
)

func NewLocal(cfg aws.Config) *LocalEc2InstanceMetadata {
	return &LocalEc2InstanceMetadata{config: cfg}
}

// Ec2InstanceMetadata returns the instance id
// TODO: we may want to implement instance verification as documented in
// https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/instance-identity-documents.html
type LocalEc2InstanceMetadata struct {
	config aws.Config
}

func (m *LocalEc2InstanceMetadata) InstanceID() (string, error) {
	metadata := ec2metadata.New(m.config)
	ctx := context.Background()
	doc, err := metadata.GetInstanceIdentityDocument(ctx)
	if err != nil {
		return "", err
	}
	return MondooInstanceID(doc.AccountID, doc.Region, doc.InstanceID), nil
}
