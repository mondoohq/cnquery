package awsec2

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/ec2/imds"
	"github.com/aws/aws-sdk-go-v2/service/ec2"

	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
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
	ec2svc := ec2.NewFromConfig(m.config)
	ctx := context.Background()
	doc, err := metadata.GetInstanceIdentityDocument(ctx, &imds.GetInstanceIdentityDocumentInput{})
	if err != nil {
		return Identity{}, err
	}
	filters := []ec2types.Filter{
		{
			Name:   aws.String("resource-id"),
			Values: []string{doc.InstanceID},
		},
	}
	tags, err := ec2svc.DescribeTags(ctx, &ec2.DescribeTagsInput{Filters: filters})
	name := ""
	if err == nil {
		for _, t := range tags.Tags {
			if t.Key != nil && *t.Key == "Name" && t.Value != nil {
				name = *t.Value
			}
		}
	}
	return Identity{
		InstanceName: name,
		InstanceID:   MondooInstanceID(doc.AccountID, doc.Region, doc.InstanceID),
		AccountID:    "//platformid.api.mondoo.app/runtime/aws/accounts/" + doc.AccountID,
	}, nil
}
