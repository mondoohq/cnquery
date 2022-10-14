package awsec2

import (
	"encoding/json"
	"io/ioutil"
	"strings"

	"github.com/aws/aws-sdk-go-v2/feature/ec2/imds"
	"github.com/cockroachdb/errors"
	"go.mondoo.com/cnquery/motor/platform"
	"go.mondoo.com/cnquery/motor/providers/os"
	"go.mondoo.com/cnquery/motor/providers/os/powershell"
)

const (
	identityUrl                   = "http://169.254.169.254/latest/dynamic/instance-identity/document"
	metadataIdentityScriptWindows = "Invoke-RestMethod -TimeoutSec 1 -URI http://169.254.169.254/latest/dynamic/instance-identity/document -UseBasicParsing | ConvertTo-Json"
)

func NewCommandInstanceMetadata(provider os.OperatingSystemProvider, pf *platform.Platform) *CommandInstanceMetadata {
	return &CommandInstanceMetadata{
		provider: provider,
		platform: pf,
	}
}

type CommandInstanceMetadata struct {
	provider os.OperatingSystemProvider
	platform *platform.Platform
}

func (m *CommandInstanceMetadata) Identify() (Identity, error) {
	instanceDocument, err := m.instanceIdentityDocument()
	if err != nil {
		return Identity{}, err
	}

	// parse into struct
	doc := imds.InstanceIdentityDocument{}
	if err := json.NewDecoder(strings.NewReader(instanceDocument)).Decode(&doc); err != nil {
		return Identity{}, errors.Wrap(err, "failed to decode EC2 instance identity document")
	}
	return Identity{
		InstanceID: MondooInstanceID(doc.AccountID, doc.Region, doc.InstanceID),
		AccountID:  "//platformid.api.mondoo.app/runtime/aws/accounts/" + doc.AccountID,
	}, nil
}

func (m *CommandInstanceMetadata) instanceIdentityDocument() (string, error) {
	switch {
	case m.platform.IsFamily(platform.FAMILY_UNIX):
		cmd, err := m.provider.RunCommand("curl " + identityUrl)
		if err != nil {
			return "", err
		}
		data, err := ioutil.ReadAll(cmd.Stdout)
		if err != nil {
			return "", err
		}

		return strings.TrimSpace(string(data)), nil
	case m.platform.IsFamily(platform.FAMILY_WINDOWS):
		cmd, err := m.provider.RunCommand(powershell.Encode(metadataIdentityScriptWindows))
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
