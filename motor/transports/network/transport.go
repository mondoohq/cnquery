package network

import (
	"strconv"

	"github.com/cockroachdb/errors"
	"github.com/spf13/afero"
	"go.mondoo.io/mondoo/motor/transports"
	"go.mondoo.io/mondoo/motor/transports/fsutil"
)

type Transport struct {
	FQDN    string
	Port    int32
	Scheme  string
	Family  []string
	Options map[string]string
}

func New(conf *transports.TransportConfig) (*Transport, error) {
	family := []string{"network"}
	if _, ok := conf.Options["tls"]; ok {
		family = append(family, "tls")
	}

	return &Transport{
		FQDN:    conf.Host,
		Port:    conf.Port,
		Scheme:  conf.Options["scheme"],
		Family:  family,
		Options: conf.Options,
	}, nil
}

func (t *Transport) Identifier() (string, error) {
	host := t.FQDN
	if t.Port != 0 {
		host = t.FQDN + ":" + strconv.Itoa(int(t.Port))
	}

	if _, ok := t.Options["tls"]; ok {
		return "//platformid.api.mondoo.app/runtime/network/tls/" + host, nil
	} else {
		return "//platformid.api.mondoo.app/runtime/network/host/" + host, nil
	}
}

func (t *Transport) URI() string {
	if t.Port == 0 {
		return t.Scheme + "://" + t.FQDN
	}
	return t.Scheme + "://" + t.FQDN + ":" + strconv.Itoa(int(t.Port))
}

func (t *Transport) Supports(mode string) bool {
	for i := range t.Family {
		if t.Family[i] == mode {
			return true
		}
	}
	return false
}

// ----------------- other requirements vv -------------------------

func (t *Transport) RunCommand(command string) (*transports.Command, error) {
	return nil, errors.New("Network transport does not implement RunCommand")
}

func (t *Transport) FileInfo(path string) (transports.FileInfoDetails, error) {
	return transports.FileInfoDetails{}, errors.New("Network transport does not implement FileInfo")
}

func (t *Transport) FS() afero.Fs {
	return &fsutil.NoFs{}
}

func (t *Transport) Close() {}

func (t *Transport) Capabilities() transports.Capabilities {
	return transports.Capabilities{}
}

func (t *Transport) Kind() transports.Kind {
	return transports.Kind_KIND_NETWORK
}

func (t *Transport) PlatformIdDetectors() []transports.PlatformIdDetector {
	return []transports.PlatformIdDetector{}
}

func (t *Transport) Runtime() string {
	return ""
}
