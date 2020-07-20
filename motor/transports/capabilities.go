package transports

type Capability int

const (
	Cabability_RunCommand Capability = iota
	Cabability_File
	Cabability_FileSearch
)

type Capabilities []Capability

func (c Capabilities) HasCapability(x Capability) bool {
	for i := range c {
		if c[i] == x {
			return true
		}
	}
	return false
}
