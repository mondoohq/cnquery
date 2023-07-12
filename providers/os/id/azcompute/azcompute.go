package azcompute

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/pkg/errors"
	"go.mondoo.com/cnquery/motor/providers/os/powershell"
	"go.mondoo.com/cnquery/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/providers/os/connection/shared"
)

const (
	identityUrl                   = "http://169.254.169.254/metadata/instance?api-version=2021-02-01"
	metadataIdentityScriptWindows = `Invoke-RestMethod -TimeoutSec 1 -Headers @{"Metadata"="true"} -Method GET -URI http://169.254.169.254/metadata/instance?api-version=2021-02-01 -UseBasicParsing | ConvertTo-Json`
)

type instanceMetadata struct {
	Compute struct {
		ResourceID     string `json:"resourceID"`
		SubscriptionID string `json:"subscriptionId"`
		Tags           string `json:"tags"`
	} `json:"compute"`
}

type Identity struct {
	InstanceID string
	AccountID  string
}

type InstanceIdentifier interface {
	Identify() (Identity, error)
}

func Resolve(conn shared.Connection, pf *inventory.Platform) (InstanceIdentifier, error) {
	if pf.IsFamily(inventory.FAMILY_UNIX) || pf.IsFamily(inventory.FAMILY_WINDOWS) {
		return &commandInstanceMetadata{conn, pf}, nil
	}
	return nil, errors.New(fmt.Sprintf("azure compute id detector is not supported for your asset: %s %s", pf.Name, pf.Version))
}

type commandInstanceMetadata struct {
	conn     shared.Connection
	platform *inventory.Platform
}

func (m *commandInstanceMetadata) Identify() (Identity, error) {
	var instanceDocument string
	switch {
	case m.platform.IsFamily(inventory.FAMILY_UNIX):
		cmd, err := m.conn.RunCommand("curl --noproxy '*' -H Metadata:true " + identityUrl)
		if err != nil {
			return Identity{}, err
		}
		data, err := io.ReadAll(cmd.Stdout)
		if err != nil {
			return Identity{}, err
		}

		instanceDocument = strings.TrimSpace(string(data))
	case m.platform.IsFamily(inventory.FAMILY_WINDOWS):
		cmd, err := m.conn.RunCommand(powershell.Encode(metadataIdentityScriptWindows))
		if err != nil {
			return Identity{}, err
		}
		data, err := io.ReadAll(cmd.Stdout)
		if err != nil {
			return Identity{}, err
		}

		instanceDocument = strings.TrimSpace(string(data))
	default:
		return Identity{}, errors.New("your platform is not supported by azure metadata identifier resource")
	}

	// parse into struct
	md := instanceMetadata{}
	if err := json.NewDecoder(strings.NewReader(instanceDocument)).Decode(&md); err != nil {
		return Identity{}, errors.Wrap(err, "failed to decode Azure Instance Metadata")
	}

	return Identity{
		InstanceID: plugin.MondooAzureInstanceID(md.Compute.ResourceID),
		AccountID:  "//platformid.api.mondoo.app/runtime/azure/subscriptions/" + md.Compute.SubscriptionID,
	}, nil
}
