package k8s

import (
	"go.mondoo.io/mondoo/motor/platform"
)

func (t *Transport) Identifier() (string, error) {
	return t.connector.Identifier()
}

func (t *Transport) Name() (string, error) {
	return t.connector.Name()
}

func (t *Transport) PlatformInfo() *platform.Platform {
	return t.connector.PlatformInfo()
}
