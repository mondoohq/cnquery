package transports

import "strconv"

type Capability int

const (
	Capability_RunCommand Capability = iota
	Capability_File
	Capability_FileSearch
	Capability_AWS
	Capability_vSphere
	Capability_Azure
	Capability_Gcp
	Capability_Arista
	Capability_Microsoft365
	Capability_Ipmi
	Capability_Equinix
	Capability_Github
	Capability_Aws_Ebs
	Capability_Gitlab
)

var CapabilityNames = map[Capability]string{
	Capability_RunCommand:   "run-command",
	Capability_File:         "file",
	Capability_FileSearch:   "file-search",
	Capability_AWS:          "api-aws",
	Capability_vSphere:      "api-vsphere",
	Capability_Azure:        "api-azure",
	Capability_Gcp:          "api-gcp",
	Capability_Arista:       "api-arista",
	Capability_Microsoft365: "api-ms365",
	Capability_Ipmi:         "api-ipmi",
	Capability_Equinix:      "api-equinix",
	Capability_Github:       "api-github",
	Capability_Aws_Ebs:      "aws-ebs",
	Capability_Gitlab:       "api-gitlab",
}

func (c Capability) String() string {
	v, ok := CapabilityNames[c]
	if ok {
		return v
	}
	return strconv.Itoa(int(c))
}

type Capabilities []Capability

func (c Capabilities) HasCapability(x Capability) bool {
	for i := range c {
		if c[i] == x {
			return true
		}
	}
	return false
}
