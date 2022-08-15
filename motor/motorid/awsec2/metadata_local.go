package awsec2

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/ec2/imds"
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

func (m *LocalEc2InstanceMetadata) Identify() (Identity, error) {
	metadata := imds.NewFromConfig(m.config)
	ctx := context.Background()
	doc, err := metadata.GetInstanceIdentityDocument(ctx, &imds.GetInstanceIdentityDocumentInput{})
	if err != nil {
		return Identity{}, err
	}
	return Identity{
		InstanceID: MondooInstanceID(doc.AccountID, doc.Region, doc.InstanceID),
		AccountID:  "//platformid.api.mondoo.app/runtime/aws/accounts/" + doc.AccountID,
	}, nil
}
