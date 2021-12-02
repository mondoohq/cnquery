package network

import (
	"net/url"
	"strconv"
	"strings"

	"github.com/cockroachdb/errors"
	"github.com/spf13/afero"
	"go.mondoo.io/mondoo/motor/transports"
	"go.mondoo.io/mondoo/motor/transports/fsutil"
)

type Transport struct {
	FQDN   string
	Port   int
	Scheme string
	Family []string
}

func New(conf *transports.TransportConfig) (*Transport, error) {
	url, err := url.Parse(conf.Host)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse target URL")
	}

	// TODO: Processing the family here needs a bit more work. It is unclear
	// where this will evolve for now, so let's keep watching it.
	// So far we know:
	// - all of them are in the `api` family (also their kind is set this way)
	// - multiple families on one service are possible (eg: http, tls, tcp)
	res := &Transport{
		Scheme: url.Scheme,
		Family: strings.Split(url.Scheme, "+"),
	}
	// TODO: detect tcp, udp, unix
	res.Family = append(res.Family, "api")

	hostBits := strings.Split(url.Host, ":")
	switch len(hostBits) {
	case 1:
		res.FQDN = hostBits[0]
	case 2:
		res.FQDN = hostBits[0]
		res.Port, err = strconv.Atoi(hostBits[1])
		if err != nil {
			return nil, errors.Wrap(err, "failed to parse port in target URL")
		}
	default:
		return nil, errors.New("malformed target URL, host cannot be parsed")
	}

	return res, nil
}

func (t *Transport) Identifier() (string, error) {
	return t.URI(), nil
}

func (t *Transport) URI() string {
	if t.Port == 0 {
		return t.Scheme + "://" + t.FQDN
	}
	return t.Scheme + "://" + t.FQDN + ":" + strconv.Itoa(t.Port)
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
	return transports.Kind_KIND_API
}

func (t *Transport) PlatformIdDetectors() []transports.PlatformIdDetector {
	return []transports.PlatformIdDetector{}
}

func (t *Transport) Runtime() string {
	return ""
}
