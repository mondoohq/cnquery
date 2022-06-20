package k8s

import (
	"go.mondoo.io/mondoo/motor/platform"
)

func (t *transport) Identifier() (string, error) {
	return t.connector.Identifier()
}

func (t *transport) Name() (string, error) {
	return t.connector.Name()
}

func (t *transport) PlatformInfo() *platform.Platform {
	return t.connector.PlatformInfo()
}
