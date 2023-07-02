package awsec2

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"

	"errors"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/ec2/imds"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"go.mondoo.com/cnquery/motor/platform"
	"go.mondoo.com/cnquery/motor/providers/os"
	"go.mondoo.com/cnquery/motor/providers/os/powershell"
)

const (
	identityUrl = "http://169.254.169.254/latest/dynamic/instance-identity/document"
	tagNameUrl  = "http://169.254.169.254/latest/meta-data/tags/instance/Name"
)

func NewCommandInstanceMetadata(provider os.OperatingSystemProvider, pf *platform.Platform, config *aws.Config) *CommandInstanceMetadata {
	return &CommandInstanceMetadata{
		provider: provider,
		platform: pf,
		config:   config,
	}
}

type CommandInstanceMetadata struct {
	provider os.OperatingSystemProvider
	platform *platform.Platform
	config   *aws.Config
}

func (m *CommandInstanceMetadata) Identify() (Identity, error) {
	instanceDocument, err := m.instanceIdentityDocument()
	if err != nil {
		return Identity{}, err
	}
	// parse into struct
	doc := imds.InstanceIdentityDocument{}
	if err := json.NewDecoder(strings.NewReader(instanceDocument)).Decode(&doc); err != nil {
		return Identity{}, errors.Join(err, errors.New("failed to decode EC2 instance identity document"))
	}

	name := ""
	// Note that the tags metadata service has to be enabled for this to work. If not, we fallback to trying to get the name
	// via the aws API (if there's a config provided).
	taggedName, err := m.instanceNameTag()
	if err == nil {
		name = taggedName
	} else if m.config != nil {
		ec2svc := ec2.NewFromConfig(*m.config)
		ctx := context.Background()
		filters := []ec2types.Filter{
			{
				Name:   aws.String("resource-id"),
				Values: []string{doc.InstanceID},
			},
		}
		tags, err := ec2svc.DescribeTags(ctx, &ec2.DescribeTagsInput{Filters: filters})
		if err == nil {
			for _, t := range tags.Tags {
				if t.Key != nil && *t.Key == "Name" && t.Value != nil {
					name = *t.Value
				}
			}
		}
	}
	return Identity{
		InstanceName: name,
		InstanceID:   MondooInstanceID(doc.AccountID, doc.Region, doc.InstanceID),
		AccountID:    "//platformid.api.mondoo.app/runtime/aws/accounts/" + doc.AccountID,
	}, nil
}

func curlWindows(url string) string {
	return fmt.Sprintf("Invoke-RestMethod -TimeoutSec 1 -URI %s -UseBasicParsing | ConvertTo-Json", url)
}

func (m *CommandInstanceMetadata) curlDocument(url string) (string, error) {
	switch {
	case m.platform.IsFamily(platform.FAMILY_UNIX):
		cmd, err := m.provider.RunCommand("curl " + url)
		if err != nil {
			return "", err
		}
		data, err := ioutil.ReadAll(cmd.Stdout)
		if err != nil {
			return "", err
		}

		return strings.TrimSpace(string(data)), nil
	case m.platform.IsFamily(platform.FAMILY_WINDOWS):
		curlCmd := curlWindows(url)
		encoded := powershell.Encode(curlCmd)
		cmd, err := m.provider.RunCommand(encoded)
		if err != nil {
			return "", err
		}
		data, err := ioutil.ReadAll(cmd.Stdout)
		if err != nil {
			return "", err
		}

		return strings.TrimSpace(string(data)), nil
	default:
		return "", errors.New("your platform is not supported by aws metadata identifier resource")
	}
}

func (m *CommandInstanceMetadata) instanceNameTag() (string, error) {
	res, err := m.curlDocument(tagNameUrl)
	if err != nil {
		return "", err
	}
	if strings.Contains(res, "Not Found") {
		return "", errors.New("metadata tags not enabled")
	}
	return res, nil
}

func (m *CommandInstanceMetadata) instanceIdentityDocument() (string, error) {
	return m.curlDocument(identityUrl)
}
