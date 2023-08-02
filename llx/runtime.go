package llx

import "go.mondoo.com/cnquery/providers-sdk/v1/resources"

type Runtime interface {
	AssetMRN() string
	Unregister(watcherUID string) error
	CreateResource(name string, args map[string]*Primitive) (Resource, error)
	CreateResourceWithID(name string, id string, args map[string]*Primitive) (Resource, error)
	WatchAndUpdate(resource Resource, field string, watcherUID string, callback func(res interface{}, err error)) error
	Schema() Schema
	Close()
}

type Schema interface {
	Lookup(resource string) *resources.ResourceInfo
	AllResources() map[string]*resources.ResourceInfo
}
