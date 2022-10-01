package explorer

type ResolvedVersion string

const (
	PreMassResolved ResolvedVersion = "v0"
	MassResolved    ResolvedVersion = "v1"
	V2Code          ResolvedVersion = "v2"
)

var globalEmpty = &Empty{}

type Services struct {
	QueryHub
	QueryConductor
}

// LocalServices is a bundle of all the services for handling policies.
// It has an optional upstream-handler embedded. If a local service does not
// yield results for a request, and the upstream handler is defined, it will
// be used instead.
type LocalServices struct {
	DataLake  DataLake
	Upstream  *Services
	Incognito bool
}

// NewLocalServices initializes a reasonably configured local services struct
func NewLocalServices(datalake DataLake, uuid string) *LocalServices {
	return &LocalServices{
		DataLake:  datalake,
		Upstream:  nil,
		Incognito: false,
	}
}
