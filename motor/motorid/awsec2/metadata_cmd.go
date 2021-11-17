package awsec2

import (
	"encoding/json"
	"io/ioutil"
	"strings"

	"github.com/aws/aws-sdk-go-v2/feature/ec2/imds"
	"github.com/cockroachdb/errors"

	"go.mondoo.io/mondoo/lumi/resources/powershell"
	"go.mondoo.io/mondoo/motor/platform"
	"go.mondoo.io/mondoo/motor/transports"
)

const (
	identityUrl                   = "http://169.254.169.254/latest/dynamic/instance-identity/document"
	metadataIdentityScriptWindows = "Invoke-RestMethod -URI http://169.254.169.254/latest/dynamic/instance-identity/document -UseBasicParsing | ConvertTo-Json"
)

func NewCommandInstanceMetadata(t transports.Transport, p *platform.Platform) *CommandInstanceMetadata {
	return &CommandInstanceMetadata{
		transport: t,
		platform:  p,
	}
}

type CommandInstanceMetadata struct {
	transport transports.Transport
	platform  *platform.Platform
}

func (m *CommandInstanceMetadata) InstanceID() (string, error) {

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
	case m.platform.IsFamily(platform.FAMILY_WINDOWS):
		cmd, err := m.transport.RunCommand(powershell.Encode(metadataIdentityScriptWindows))
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
	doc := imds.InstanceIdentityDocument{}
	if err := json.NewDecoder(strings.NewReader(instanceDocument)).Decode(&doc); err != nil {
		return "", errors.Wrap(err, "failed to decode EC2 instance identity document")
	}

	return MondooInstanceID(doc.AccountID, doc.Region, doc.InstanceID), nil
}
