package azure

import (
	"errors"

	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/spf13/afero"
	"go.mondoo.io/mondoo/motor/providers"
	"go.mondoo.io/mondoo/motor/providers/fsutil"
)

var (
	_ providers.Transport                   = (*Transport)(nil)
	_ providers.TransportPlatformIdentifier = (*Transport)(nil)
)

func New(tc *providers.TransportConfig) (*Transport, error) {
	if tc.Backend != providers.ProviderType_AZURE {
		return nil, errors.New("backend is not supported for azure transport")
	}

	if tc.Options == nil || len(tc.Options["subscriptionID"]) == 0 {
		return nil, errors.New("azure backend requires a subscriptionID")
	}

	if tc.Options == nil || len(tc.Options["tenantID"]) == 0 {
		return nil, errors.New("azure backend requires a tenantID")
	}

	return &Transport{
		subscriptionID: tc.Options["subscriptionID"],
		tenantID:       tc.Options["tenantID"],
		opts:           tc.Options,
	}, nil
}

type Transport struct {
	subscriptionID string
	tenantID       string
	opts           map[string]string
}

func (t *Transport) RunCommand(command string) (*providers.Command, error) {
	return nil, errors.New("azure does not implement RunCommand")
}

func (t *Transport) FileInfo(path string) (providers.FileInfoDetails, error) {
	return providers.FileInfoDetails{}, errors.New("azure does not implement FileInfo")
}

func (t *Transport) FS() afero.Fs {
	return &fsutil.NoFs{}
}

func (t *Transport) Close() {}

func (t *Transport) Capabilities() providers.Capabilities {
	return providers.Capabilities{
		providers.Capability_Azure,
	}
}

func (t *Transport) Options() map[string]string {
	return t.opts
}

func (t *Transport) Kind() providers.Kind {
	return providers.Kind_KIND_API
}

func (t *Transport) Runtime() string {
	return providers.RUNTIME_AZ
}

func (t *Transport) PlatformIdDetectors() []providers.PlatformIdDetector {
	return []providers.PlatformIdDetector{
		providers.TransportPlatformIdentifierDetector,
	}
}

func GetAuthorizer() (autorest.Authorizer, error) {
	return auth.NewAuthorizerFromCLI()
}

func (t *Transport) Authorizer() (autorest.Authorizer, error) {
	return GetAuthorizer()
}

func (t *Transport) AuthorizerWithAudience(audience string) (autorest.Authorizer, error) {
	return auth.NewAuthorizerFromCLIWithResource(audience)
}

func (t *Transport) ParseResourceID(id string) (*ResourceID, error) {
	return ParseResourceID(id)
}
