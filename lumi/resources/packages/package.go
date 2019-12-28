package packages

type Package struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Arch        string `json:"arch"`
	Status      string `json:"status,omitempty"`
	Description string `json:"description"`

	// this may be the source package or an origin
	// e.g. on alpine it is used for parent  packages
	// o 	Package Origin - https://wiki.alpinelinux.org/wiki/Apk_spec
	Origin string `json:"origin"`
	Format string `json:"format"`
}
