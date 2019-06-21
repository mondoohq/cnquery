package awsec2

import (
	"encoding/json"
	"io/ioutil"
	"strings"

	"github.com/pkg/errors"

	"github.com/aws/aws-sdk-go-v2/aws/ec2metadata"
	motor "go.mondoo.io/mondoo/motor/motoros"
	"go.mondoo.io/mondoo/motor/motoros/platform"
)

func NewUnix(m *motor.Motor) *UnixInstanceMetadata {
	return &UnixInstanceMetadata{motor: m}
}

type UnixInstanceMetadata struct {
	motor *motor.Motor
}

func (m *UnixInstanceMetadata) InstanceID() (string, error) {
	motor := m.motor
	identityUrl := "http://169.254.169.254/latest/dynamic/instance-identity/document"

	pi, err := motor.Platform()
	if err != nil {
		return "", err
	}

	var instanceDocument string
	switch {
	case pi.IsFamily(platform.FAMILY_UNIX):
		cmd, err := motor.Transport.RunCommand("curl " + identityUrl)
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
