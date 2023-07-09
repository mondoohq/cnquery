package gce

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"go.mondoo.com/cnquery/motor/platform"
	"go.mondoo.com/cnquery/motor/providers/os/powershell"
	"go.mondoo.com/cnquery/providers/os/connection/shared"
)

func MondooGcpInstanceID(project string, zone string, instanceID uint64) string {
	return "//platformid.api.mondoo.app/runtime/gcp/compute/v1/projects/" + project + "/zones/" + zone + "/instances/" + strconv.FormatUint(uint64(instanceID), 10)
}

type Identity struct {
	InstanceID string
	ProjectID  string
}

type InstanceIdentifier interface {
	Identify() (Identity, error)
}

func Resolve(conn shared.Connection, pf *platform.Platform) (InstanceIdentifier, error) {
	if pf.IsFamily(platform.FAMILY_UNIX) || pf.IsFamily(platform.FAMILY_WINDOWS) {
		return &commandInstanceMetadata{conn, pf}, nil
	}
	return nil, errors.New(fmt.Sprintf("gce id detector is not supported for your asset: %s %s", pf.Name, pf.Version))
}

const (
	metadataSvcURL                = "http://metadata.google.internal/computeMetadata/v1/"
	metadataIdentityScriptWindows = `
$projectID = Invoke-RestMethod -Headers @{"Metadata-Flavor"="Google"} -Method GET -URI http://metadata.google.internal/computeMetadata/v1/project/project-id?alt=json -UseBasicParsing
$instanceID = Invoke-RestMethod -Headers @{"Metadata-Flavor"="Google"} -Method GET -URI http://metadata.google.internal/computeMetadata/v1/instance/id?alt=json -UseBasicParsing
$zone = Invoke-RestMethod -Headers @{"Metadata-Flavor"="Google"} -Method GET -URI http://metadata.google.internal/computeMetadata/v1/instance/zone?alt=json -UseBasicParsing

$doc = New-Object -TypeName PSObject
$doc | Add-Member -MemberType NoteProperty -Value $projectID -Name ProjectID
$doc | Add-Member -MemberType NoteProperty -Value $instanceID -Name InstanceID
$doc | Add-Member -MemberType NoteProperty -Value $zone -Name ZoneInfo

$doc | ConvertTo-Json
`
)

type commandInstanceMetadata struct {
	conn     shared.Connection
	platform *platform.Platform
}

func (m *commandInstanceMetadata) curl(key string, v interface{}) error {
	cmd, err := m.conn.RunCommand("curl --noproxy '*' -H Metadata-Flavor:Google " + metadataSvcURL + key + "?alt=json")
	if err != nil {
		return err
	}

	return json.NewDecoder(cmd.Stdout).Decode(v)
}

func (m *commandInstanceMetadata) Identify() (Identity, error) {
	switch {
	case m.platform.IsFamily(platform.FAMILY_UNIX):
		var projectID string
		var instanceID uint64
		var zoneInfo string

		if err := m.curl("project/project-id", &projectID); err != nil {
			return Identity{}, err
		}

		if err := m.curl("instance/id", &instanceID); err != nil {
			return Identity{}, err
		}

		if err := m.curl("instance/zone", &zoneInfo); err != nil {
			return Identity{}, err
		}

		zone := zoneInfo[strings.LastIndex(zoneInfo, "/")+1:]
		return Identity{
			ProjectID:  "//platformid.api.mondoo.app/runtime/gcp/projects/" + projectID,
			InstanceID: MondooGcpInstanceID(projectID, zone, instanceID),
		}, nil
	case m.platform.IsFamily(platform.FAMILY_WINDOWS):
		cmd, err := m.conn.RunCommand(powershell.Encode(metadataIdentityScriptWindows))
		if err != nil {
			return Identity{}, err
		}
		data, err := ioutil.ReadAll(cmd.Stdout)
		if err != nil {
			return Identity{}, err
		}

		instanceDocument := strings.TrimSpace(string(data))
		doc := struct {
			ProjectID  string `json:"ProjectID"`
			InstanceID uint64 `json:"InstanceID"`
			ZoneInfo   string `json:"ZoneInfo"`
		}{}
		json.Unmarshal([]byte(instanceDocument), &doc)
		zone := doc.ZoneInfo[strings.LastIndex(doc.ZoneInfo, "/")+1:]

		return Identity{
			ProjectID:  "//platformid.api.mondoo.app/runtime/gcp/projects/" + doc.ProjectID,
			InstanceID: MondooGcpInstanceID(doc.ProjectID, zone, doc.InstanceID),
		}, nil
	default:
		return Identity{}, errors.New("your platform is not supported by azure metadata identifier resource")
	}
}
