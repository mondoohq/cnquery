package gce

import (
	"encoding/json"
	"io"
	"strings"

	"github.com/pkg/errors"
	"go.mondoo.com/cnquery/motor/platform"
	"go.mondoo.com/cnquery/motor/providers/os"
	"go.mondoo.com/cnquery/motor/providers/os/powershell"
)

const (
	metadataSvcURL                = "http://metadata.google.internal/computeMetadata/v1/"
	metadataIdentityScriptWindows = `
$projectID = Invoke-RestMethod -Headers @{"Metadata-Flavor"="Google"} -Method GET -URI http://metadata.google.internal/computeMetadata/v1/project/project-id?alt=json -UseBasicParsing
$instanceID = Invoke-RestMethod -Headers @{"Metadata-Flavor"="Google"} -Method GET -URI http://metadata.google.internal/computeMetadata/v1/instance/id?alt=json -UseBasicParsing
$instanceName = Invoke-RestMethod -Headers @{"Metadata-Flavor"="Google"} -Method GET -URI http://metadata.google.internal/computeMetadata/v1/instance/name?alt=json -UseBasicParsing
$zone = Invoke-RestMethod -Headers @{"Metadata-Flavor"="Google"} -Method GET -URI http://metadata.google.internal/computeMetadata/v1/instance/zone?alt=json -UseBasicParsing

$doc = New-Object -TypeName PSObject
$doc | Add-Member -MemberType NoteProperty -Value $projectID -Name ProjectID
$doc | Add-Member -MemberType NoteProperty -Value $instanceID -Name InstanceID
$doc | Add-Member -MemberType NoteProperty -Value $instanceName -Name InstanceName
$doc | Add-Member -MemberType NoteProperty -Value $zone -Name ZoneInfo

$doc | ConvertTo-Json
`
)

func NewCommandInstanceMetadata(provider os.OperatingSystemProvider, platform *platform.Platform) *CommandInstanceMetadata {
	return &CommandInstanceMetadata{
		provider: provider,
		platform: platform,
	}
}

type CommandInstanceMetadata struct {
	provider os.OperatingSystemProvider
	platform *platform.Platform
}

func (m *CommandInstanceMetadata) curl(key string, v interface{}) error {
	cmd, err := m.provider.RunCommand("curl --noproxy '*' -H Metadata-Flavor:Google " + metadataSvcURL + key + "?alt=json")
	if err != nil {
		return err
	}

	return json.NewDecoder(cmd.Stdout).Decode(v)
}

func (m *CommandInstanceMetadata) Identify() (Identity, error) {
	switch {
	case m.platform.IsFamily(platform.FAMILY_UNIX):
		var projectID string
		var instanceID uint64
		var instanceName string
		var zoneInfo string

		if err := m.curl("project/project-id", &projectID); err != nil {
			return Identity{}, err
		}

		if err := m.curl("instance/id", &instanceID); err != nil {
			return Identity{}, err
		}

		if err := m.curl("instance/name", &instanceName); err != nil {
			return Identity{}, err
		}

		if err := m.curl("instance/zone", &zoneInfo); err != nil {
			return Identity{}, err
		}

		zone := zoneInfo[strings.LastIndex(zoneInfo, "/")+1:]
		return Identity{
			ProjectID:   "//platformid.api.mondoo.app/runtime/gcp/projects/" + projectID,
			InstanceID:  MondooGcpInstanceID(projectID, zone, instanceID),
			PlatformMrn: MondooGcpInstancePlatformMrn(projectID, zone, instanceName),
		}, nil
	case m.platform.IsFamily(platform.FAMILY_WINDOWS):
		cmd, err := m.provider.RunCommand(powershell.Encode(metadataIdentityScriptWindows))
		if err != nil {
			return Identity{}, err
		}
		data, err := io.ReadAll(cmd.Stdout)
		if err != nil {
			return Identity{}, err
		}

		instanceDocument := strings.TrimSpace(string(data))
		doc := struct {
			ProjectID    string `json:"ProjectID"`
			InstanceID   uint64 `json:"InstanceID"`
			InstanceName string `json:"InstanceName"`
			ZoneInfo     string `json:"ZoneInfo"`
		}{}
		json.Unmarshal([]byte(instanceDocument), &doc)
		zone := doc.ZoneInfo[strings.LastIndex(doc.ZoneInfo, "/")+1:]

		return Identity{
			ProjectID:   "//platformid.api.mondoo.app/runtime/gcp/projects/" + doc.ProjectID,
			InstanceID:  MondooGcpInstanceID(doc.ProjectID, zone, doc.InstanceID),
			PlatformMrn: MondooGcpInstancePlatformMrn(doc.ProjectID, zone, doc.InstanceName),
		}, nil
	default:
		return Identity{}, errors.New("your platform is not supported by azure metadata identifier resource")
	}
}
