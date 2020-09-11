package ms365

import (
	"os"

	"github.com/cockroachdb/errors"

	"github.com/spf13/afero"
	"go.mondoo.io/mondoo/motor/transports"
	"go.mondoo.io/mondoo/motor/transports/fsutil"
)

func New(tc *transports.TransportConfig) (*Transport, error) {
	if tc.Backend != transports.TransportBackend_CONNECTION_MS365 {
		return nil, errors.New("backend is not supported for ms365 transport")
	}

	if len(tc.IdentityFiles) != 1 {
		return nil, errors.New("ms365 backend requires a credentials file, pass json via -i option")
	}

	var msauth *MicrosoftAuth
	if len(tc.IdentityFiles) == 1 {

		filename := tc.IdentityFiles[0]

		f, err := os.Open(filename)
		if err != nil {
			return nil, errors.Wrap(err, "could not open credentials file")
		}

		msauth, err = ParseMicrosoftAuth(f)
		if err != nil {
			return nil, errors.Wrap(err, "could not parse credentials file")
		}
	}

	if msauth == nil {
		return nil, errors.New("could not parse credentials file")
	}

	if len(msauth.TenantId) == 0 {
		return nil, errors.New("ms365 backend requires a tenantID")
	}

	return &Transport{
		tenantID:     msauth.TenantId,
		opts:         tc.Options,
		clientID:     msauth.ClientId,
		clientSecret: msauth.ClientSecret,
	}, nil
}

type Transport struct {
	tenantID     string
	clientID     string
	clientSecret string
	opts         map[string]string
}

func (t *Transport) RunCommand(command string) (*transports.Command, error) {
	return nil, errors.New("ms365 does not implement RunCommand")
}

func (t *Transport) FileInfo(path string) (transports.FileInfoDetails, error) {
	return transports.FileInfoDetails{}, errors.New("ms365 does not implement FileInfo")
}

func (t *Transport) FS() afero.Fs {
	return &fsutil.NoFs{}
}

func (t *Transport) Close() {}

func (t *Transport) Capabilities() transports.Capabilities {
	return transports.Capabilities{}
}

func (t *Transport) Options() map[string]string {
	return t.opts
}

func (t *Transport) Kind() transports.Kind {
	return transports.Kind_KIND_API
}

func (t *Transport) Runtime() string {
	return transports.RUNTIME_AZ
}
