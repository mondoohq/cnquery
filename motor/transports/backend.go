package transports

import (
	"errors"
	"strconv"

	"github.com/rs/zerolog/log"
)

var TransportBackend_scheme = map[TransportBackend]string{
	TransportBackend_CONNECTION_LOCAL_OS:                "local",
	TransportBackend_CONNECTION_DOCKER_ENGINE_IMAGE:     "docker+image",
	TransportBackend_CONNECTION_DOCKER_ENGINE_CONTAINER: "docker+container",
	TransportBackend_CONNECTION_SSH:                     "ssh",
	TransportBackend_CONNECTION_WINRM:                   "winrm",
	TransportBackend_CONNECTION_AWS_SSM_RUN_COMMAND:     "aws+ssm",
	TransportBackend_CONNECTION_CONTAINER_REGISTRY:      "cr",
	TransportBackend_CONNECTION_TAR:                     "tar",
	TransportBackend_CONNECTION_MOCK:                    "mock",
	TransportBackend_CONNECTION_VSPHERE:                 "vsphere",
	TransportBackend_CONNECTION_ARISTAEOS:               "arista",
	TransportBackend_CONNECTION_CONTAINER_TAR:           "container+tar",
	TransportBackend_CONNECTION_AWS:                     "aws",
	TransportBackend_CONNECTION_GCP:                     "gcp",
	TransportBackend_CONNECTION_AZURE:                   "azure",
	TransportBackend_CONNECTION_MS365:                   "ms365",
	TransportBackend_CONNECTION_IPMI:                    "ipmi",
	TransportBackend_CONNECTION_VSPHERE_VM:              "vsphere+vm",
	TransportBackend_CONNECTION_FS:                      "fs",
}

var TransportBackend_schemevalue = map[string]TransportBackend{
	"local":            TransportBackend_CONNECTION_LOCAL_OS,
	"docker+image":     TransportBackend_CONNECTION_DOCKER_ENGINE_IMAGE,
	"docker+container": TransportBackend_CONNECTION_DOCKER_ENGINE_CONTAINER,
	"ssh":              TransportBackend_CONNECTION_SSH,
	"winrm":            TransportBackend_CONNECTION_WINRM,
	"aws+ssm":          TransportBackend_CONNECTION_AWS_SSM_RUN_COMMAND,
	"cr":               TransportBackend_CONNECTION_CONTAINER_REGISTRY,
	"tar":              TransportBackend_CONNECTION_TAR,
	"mock":             TransportBackend_CONNECTION_MOCK,
	"vsphere":          TransportBackend_CONNECTION_VSPHERE,
	"arista":           TransportBackend_CONNECTION_ARISTAEOS,
	"container+tar":    TransportBackend_CONNECTION_CONTAINER_TAR,
	"aws":              TransportBackend_CONNECTION_AWS,
	"gcp":              TransportBackend_CONNECTION_GCP,
	"azure":            TransportBackend_CONNECTION_AZURE,
	"ms365":            TransportBackend_CONNECTION_MS365,
	"ipmi":             TransportBackend_CONNECTION_IPMI,
	"vsphere+vm":       TransportBackend_CONNECTION_VSPHERE_VM,
	"fs":               TransportBackend_CONNECTION_FS,
}

func (x TransportBackend) Scheme() string {
	s, ok := TransportBackend_scheme[x]
	if ok {
		return s
	}
	log.Warn().Str("backend", x.String()).Msg("cannot return scheme for backend")
	return strconv.Itoa(int(x))
}

func MapSchemeBackend(scheme string) (TransportBackend, error) {
	s, ok := TransportBackend_schemevalue[scheme]
	if ok {
		return s, nil
	}

	return TransportBackend_CONNECTION_LOCAL_OS, errors.New("unknown connection scheme: " + scheme)
}
