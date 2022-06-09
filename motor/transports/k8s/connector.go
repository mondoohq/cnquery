package k8s

import (
	"go.mondoo.io/mondoo/motor/platform"
	"go.mondoo.io/mondoo/motor/transports/k8s/resources"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/version"
)

type Connector interface {
	Name() (string, error)
	Identifier() (string, error)
	Resources(kind string, name string) (*ResourceResult, error)
	PlatformInfo() *platform.Platform
	ServerVersion() *version.Info
	SupportedResourceTypes() (*resources.ApiResourceIndex, error)
	Namespaces() (*v1.NamespaceList, error)
	Pods(namespace v1.Namespace) (*v1.PodList, error)
}

type ResourceResult struct {
	Name          string
	Kind          string
	ResourceType  *resources.ApiResource // resource type that matched kind
	AllResources  []runtime.Object
	RootResources []runtime.Object
	Namespace     string
	AllNs         bool
}
