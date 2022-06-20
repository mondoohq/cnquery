package k8s

import (
	"go.mondoo.io/mondoo/motor/transports/k8s/resources"
	"k8s.io/apimachinery/pkg/version"
)

func (t *transport) Resources(kind string, name string) (*ResourceResult, error) {
	return t.connector.Resources(kind, name)
}

func (t *transport) ServerVersion() *version.Info {
	return t.connector.ServerVersion()
}

func (t *transport) SupportedResourceTypes() (*resources.ApiResourceIndex, error) {
	return t.connector.SupportedResourceTypes()
}
