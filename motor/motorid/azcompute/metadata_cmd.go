package azcompute

import (
	"encoding/json"
	"io/ioutil"
	"strings"

	"github.com/pkg/errors"
	"go.mondoo.com/cnquery/motor/discovery/azure"
	"go.mondoo.com/cnquery/motor/platform"
	"go.mondoo.com/cnquery/motor/providers/os"
	"go.mondoo.com/cnquery/motor/providers/os/powershell"
)

const (
	identityUrl                   = "http://169.254.169.254/metadata/instance?api-version=2021-02-01"
	metadataIdentityScriptWindows = `Invoke-RestMethod -Headers @{"Metadata"="true"} -Method GET -URI http://169.254.169.254/metadata/instance?api-version=2021-02-01 -UseBasicParsing | ConvertTo-Json`
)

func NewCommandInstanceMetadata(p os.OperatingSystemProvider, pf *platform.Platform) *CommandInstanceMetadata {
	return &CommandInstanceMetadata{
		provider: p,
		platform: pf,
	}
}

type CommandInstanceMetadata struct {
	provider os.OperatingSystemProvider
	platform *platform.Platform
}

func (m *CommandInstanceMetadata) InstanceID() (string, error) {
	var instanceDocument string
	switch {
	case m.platform.IsFamily(platform.FAMILY_UNIX):
		cmd, err := m.provider.RunCommand("curl --noproxy '*' -H Metadata:true " + identityUrl)
		if err != nil {
			return "", err
		}
		data, err := ioutil.ReadAll(cmd.Stdout)
		if err != nil {
			return "", err
		}

		instanceDocument = strings.TrimSpace(string(data))
	case m.platform.IsFamily(platform.FAMILY_WINDOWS):
		cmd, err := m.provider.RunCommand(powershell.Encode(metadataIdentityScriptWindows))
		if err != nil {
			return "", err
		}
		data, err := ioutil.ReadAll(cmd.Stdout)
		if err != nil {
			return "", err
		}

		instanceDocument = strings.TrimSpace(string(data))
	default:
		return "", errors.New("your platform is not supported by azure metadata identifier resource")
	}

	// parse into struct
	md := instanceMetadata{}
	if err := json.NewDecoder(strings.NewReader(instanceDocument)).Decode(&md); err != nil {
		return "", errors.Wrap(err, "failed to decode Azure Instance Metadata")
	}

	return azure.MondooAzureInstanceID(md.Compute.ResourceID), nil
}
