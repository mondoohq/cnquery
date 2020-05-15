package capabilities

type Capability int

const (
	RunCommand Capability = iota
	File
	FileSearch
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
