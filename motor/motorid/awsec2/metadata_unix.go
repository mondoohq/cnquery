package awsec2

import (
	"encoding/json"
	"io/ioutil"
	"strings"

	"github.com/pkg/errors"

	"github.com/aws/aws-sdk-go-v2/aws/ec2metadata"
	"go.mondoo.io/mondoo/motor/platform"
	"go.mondoo.io/mondoo/motor/transports"
)

func NewUnix(t transports.Transport, p *platform.Platform) *UnixInstanceMetadata {
	return &UnixInstanceMetadata{
		transport: t,
		platform:  p,
	}
}

type UnixInstanceMetadata struct {
	transport transports.Transport
	platform  *platform.Platform
}

func (m *UnixInstanceMetadata) InstanceID() (string, error) {
	identityUrl := "http://169.254.169.254/latest/dynamic/instance-identity/document"

	var instanceDocument string
	switch {
	case m.platform.IsFamily(platform.FAMILY_UNIX):
		cmd, err := m.transport.RunCommand("curl " + identityUrl)
		if err != nil {
			return "", err
		}
		data, err := ioutil.ReadAll(cmd.Stdout)
		if err != nil {
			return "", err
		}

		instanceDocument = strings.TrimSpace(string(data))
	default:
		return "", errors.New("your platform is not supported by aws metadata identifier resource")
	}

	// parse into struct
	doc := ec2metadata.EC2InstanceIdentityDocument{}
	if err := json.NewDecoder(strings.NewReader(instanceDocument)).Decode(&doc); err != nil {
		return "", errors.Wrap(err, "failed to decode EC2 instance identity document")
	}

	return MondooInstanceID(doc.AccountID, doc.Region, doc.InstanceID), nil
}
