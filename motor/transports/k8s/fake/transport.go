package fake

import (
	"errors"

	gomock "github.com/golang/mock/gomock"
	"github.com/spf13/afero"
	platform "go.mondoo.io/mondoo/motor/platform"
	"go.mondoo.io/mondoo/motor/transports"
	"go.mondoo.io/mondoo/motor/transports/fsutil"
	k8s "go.mondoo.io/mondoo/motor/transports/k8s"
	resources "go.mondoo.io/mondoo/motor/transports/k8s/resources"
	version "k8s.io/apimachinery/pkg/version"
)

var _ k8s.Transport = (*FakeTransport)(nil)

type FakeTransport struct {
	MockConnector *MockConnector
}

func NewFakeTransport(ctrl *gomock.Controller) *FakeTransport {
	return &FakeTransport{MockConnector: NewMockConnector(ctrl)}
}

func (t *FakeTransport) Identifier() (string, error) {
	return t.MockConnector.Identifier()
}

func (t *FakeTransport) Name() (string, error) {
	return t.MockConnector.Name()
}

func (t *FakeTransport) PlatformInfo() *platform.Platform {
	return t.MockConnector.PlatformInfo()
}

func (t *FakeTransport) Resources(kind string, name string) (*k8s.ResourceResult, error) {
	return t.MockConnector.Resources(kind, name)
}

func (t *FakeTransport) ServerVersion() *version.Info {
	return t.MockConnector.ServerVersion()
}

func (t *FakeTransport) SupportedResourceTypes() (*resources.ApiResourceIndex, error) {
	return t.MockConnector.SupportedResourceTypes()
}

func (t *FakeTransport) Connector() k8s.Connector {
	return t.MockConnector
}

func (t *FakeTransport) RunCommand(command string) (*transports.Command, error) {
	return nil, errors.New("k8s does not implement RunCommand")
}

func (t *FakeTransport) FileInfo(path string) (transports.FileInfoDetails, error) {
	return transports.FileInfoDetails{}, errors.New("k8s does not implement FileInfo")
}

func (t *FakeTransport) FS() afero.Fs {
	return &fsutil.NoFs{}
}

func (t *FakeTransport) Close() {}

func (t *FakeTransport) Capabilities() transports.Capabilities {
	return transports.Capabilities{}
}

func (t *FakeTransport) Options() map[string]string {
	return nil
}

func (t *FakeTransport) Kind() transports.Kind {
	return transports.Kind_KIND_API
}

func (t *FakeTransport) Runtime() string {
	return transports.RUNTIME_KUBERNETES
}

func (t *FakeTransport) PlatformIdDetectors() []transports.PlatformIdDetector {
	return []transports.PlatformIdDetector{
		transports.TransportPlatformIdentifierDetector,
	}
}
