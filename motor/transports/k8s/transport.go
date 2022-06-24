package k8s

import (
	"errors"

	platform "go.mondoo.io/mondoo/motor/platform"
	"go.mondoo.io/mondoo/motor/transports"
	"go.mondoo.io/mondoo/motor/transports/k8s/resources"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/version"
)

const (
	OPTION_MANIFEST  = "path"
	OPTION_NAMESPACE = "namespace"
)

//go:generate mockgen -source=./transport.go -destination=./fake/transport_generated.go -package=fake

type Transport interface {
	transports.Transport
	transports.TransportPlatformIdentifier
	Name() (string, error)
	PlatformInfo() *platform.Platform

	// Resources returns the resources that match the provided kind and name. If not kind and name
	// are provided, then all cluster resources are returned.
	Resources(kind string, name string) (*ResourceResult, error)
	ServerVersion() *version.Info
	SupportedResourceTypes() (*resources.ApiResourceIndex, error)

	Identifier() (string, error)
	Namespaces() ([]v1.Namespace, error)
	Pods(namespace v1.Namespace) ([]v1.Pod, error)
}

type ClusterInfo struct {
	Name string
}

type ResourceResult struct {
	Name         string
	Kind         string
	ResourceType *resources.ApiResource // resource type that matched kind

	// Resources the resources that match the name, kind and namespace
	Resources []runtime.Object
	Namespace string
	AllNs     bool
}

// New initializes the k8s transport and loads a configuration.
// Supported options are:
// - namespace: limits the resources to a specific namespace
// - path: use a manifest file instead of live API
func New(tc *transports.TransportConfig) (Transport, error) {
	if tc.Backend != transports.TransportBackend_CONNECTION_K8S {
		return nil, errors.New("backend is not supported for k8s transport")
	}

	manifestFile, manifestDefined := tc.Options[OPTION_MANIFEST]
	if manifestDefined {
		return newManifestTransport(WithManifestFile(manifestFile), WithNamespace(tc.Options[OPTION_NAMESPACE])), nil
	}

	return newApiTransport(tc.Options[OPTION_NAMESPACE])
}
