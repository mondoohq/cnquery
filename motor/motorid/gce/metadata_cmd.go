package gce

import (
	"encoding/json"
	"io/ioutil"
	"strings"

	"github.com/pkg/errors"
	"go.mondoo.io/mondoo/motor/platform"
	"go.mondoo.io/mondoo/motor/providers/os"
	"go.mondoo.io/mondoo/resources/packs/os/powershell"
)

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

func (m *CommandInstanceMetadata) InstanceID() (string, error) {
	switch {
	case m.platform.IsFamily(platform.FAMILY_UNIX):
		var projectID string
		var instanceID uint64
		var zoneInfo string

		if err := m.curl("project/project-id", &projectID); err != nil {
			return "", err
		}

		if err := m.curl("instance/id", &instanceID); err != nil {
			return "", err
		}

		if err := m.curl("instance/zone", &zoneInfo); err != nil {
			return "", err
		}

		zone := zoneInfo[strings.LastIndex(zoneInfo, "/")+1:]

		return MondooGcpInstanceID(projectID, zone, instanceID), nil
	case m.platform.IsFamily(platform.FAMILY_WINDOWS):
		cmd, err := m.provider.RunCommand(powershell.Encode(metadataIdentityScriptWindows))
		if err != nil {
			return "", err
		}
		data, err := ioutil.ReadAll(cmd.Stdout)
		if err != nil {
			return "", err
		}

		instanceDocument := strings.TrimSpace(string(data))
		doc := struct {
			ProjectID  string `json:"ProjectID"`
			InstanceID uint64 `json:"InstanceID"`
			ZoneInfo   string `json:"ZoneInfo"`
		}{}
		json.Unmarshal([]byte(instanceDocument), &doc)
		zone := doc.ZoneInfo[strings.LastIndex(doc.ZoneInfo, "/")+1:]

		return MondooGcpInstanceID(doc.ProjectID, zone, doc.InstanceID), nil
	default:
		return "", errors.New("your platform is not supported by azure metadata identifier resource")
	}
}
