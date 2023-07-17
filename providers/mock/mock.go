package mock

import (
	"os"

	"github.com/BurntSushi/toml"
	"github.com/pkg/errors"
	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/providers"
	"go.mondoo.com/cnquery/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/providers-sdk/v1/resources"
)

// Unlike other providers, we are currently building this into the core
// of the providers library. In that, it is similar to the core providers.
// Both are optional and both will be removed from being built-in in the
// future, at least from some of the builds.
//
// The reason for this decision is that we want to use it for testing and
// recording/playback in all scenarios. Because core needs to support
// parsers at the moment anyway, we get the benefit of having those
// libraries anyway. So the overhead of this additional loader is very
// small.

type Mock struct {
	Inventory map[string]Resources
	Providers []string
	schema    llx.Schema
}

type Resources map[string]Resource

type Resource struct {
	Fields map[string]plugin.DataRes
}

func NewFromTomlFile(path string) (*Mock, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return NewFromToml(data)
}

func loadRawDataRes(raw interface{}) (plugin.DataRes, error) {
	switch v := raw.(type) {
	case string:
		return plugin.DataRes{Data: llx.StringPrimitive(v)}, nil
	case int64:
		return plugin.DataRes{Data: llx.IntPrimitive(v)}, nil
	case bool:
		return plugin.DataRes{Data: llx.BoolPrimitive(v)}, nil
	default:
		return plugin.DataRes{}, errors.New("failed to load value")
	}
}

func loadRawFields(resource *Resource, fields map[string]interface{}) error {
	var err error
	for field, raw := range fields {
		resource.Fields[field], err = loadRawDataRes(raw)
		if err != nil {
			return err
		}
	}
	return nil
}

func loadMqlInfo(raw interface{}, m *Mock) error {
	info, ok := raw.(map[string]interface{})
	if !ok {
		return errors.New("mql info is not a map")
	}

	if p, ok := info["providers"]; ok {
		list := p.([]interface{})
		for _, v := range list {
			m.Providers = append(m.Providers, v.(string))
		}
	}

	return nil
}

func New() *Mock {
	return &Mock{
		Inventory: map[string]Resources{},
		Providers: []string{},
		schema:    emptySchema{},
	}
}

func NewFromToml(raw []byte) (*Mock, error) {
	var tmp interface{}
	err := toml.Unmarshal(raw, &tmp)
	if err != nil {
		return nil, err
	}

	res := Mock{
		Inventory: map[string]Resources{},
		schema:    providers.DefaultRuntime().Schema(),
	}
	err = nil

	rawResources, ok := tmp.(map[string]interface{})
	if !ok {
		return nil, errors.New("incorrect structure of recording TOML (outer layer should be resources)")
	}

	if mqlInfo, ok := rawResources["mql"]; ok {
		loadMqlInfo(mqlInfo, &res)
		delete(rawResources, "mql")
	}

	for name, v := range rawResources {
		resources := Resources{}
		res.Inventory[name] = resources

		rawList, ok := v.(map[string]interface{})
		if !ok {
			return nil, errors.New("incorrect structure of recording TOML (" + name + " resources should be followed by IDs)")
		}

		for id, vv := range rawList {
			resource := Resource{
				Fields: map[string]plugin.DataRes{},
			}

			rawFields, ok := vv.(map[string]interface{})
			if !ok {
				return nil, errors.New("incorrect structure of recording TOML (resource " + name + " (id: " + id + ") should have fields set)")
			}

			if err = loadRawFields(&resource, rawFields); err != nil {
				return nil, err
			}

			resources[id] = resource
		}
	}

	return &res, err
}

func (m *Mock) Unregister(watcherUID string) error {
	// nothing will change, so nothing to watch or unregister
	return nil
}

func (m *Mock) CreateResource(name string, args map[string]*llx.Primitive) (llx.Resource, error) {
	resourceCache, ok := m.Inventory[name]
	if !ok {
		return nil, errors.New("resource '" + name + "' is not in recording")
	}

	// FIXME: we currently have no way of generating the ID that we need to get the right resource,
	// until we have a solid way to (1) connect the right provider and (2) use it to generate the ID on the fly.
	//
	// For now, we are just using a few hardcoded workaround...

	switch name {
	case "command":
		rid, ok := args["command"]
		if !ok {
			return nil, errors.New("cannot find '" + name + "' ID in recording")
		}

		id := string(rid.Value)
		_, ok = resourceCache[id]
		if !ok {
			return nil, errors.New("cannot find " + name + " '" + id + "' in recording")
		}

		return &llx.MockResource{Name: name, ID: id}, nil

	case "file":
		fid, ok := args["path"]
		if !ok {
			return nil, errors.New("cannot find '" + name + "' ID in recording")
		}

		id := string(fid.Value)
		_, ok = resourceCache[id]
		if !ok {
			return nil, errors.New("cannot find " + name + " '" + id + "' in recording")
		}

		return &llx.MockResource{Name: name, ID: id}, nil

	default:
		// for all static resources
		if _, ok := resourceCache[""]; ok {
			return &llx.MockResource{Name: name, ID: ""}, nil
		}
	}

	return nil, errors.New("cannot create resource '" + name + "' from recording yet")
}

func (m *Mock) CreateResourceWithID(name string, id string, args map[string]*llx.Primitive) (llx.Resource, error) {
	resourceCache, ok := m.Inventory[name]
	if !ok {
		return nil, errors.New("resource '" + name + "' is not in recording")
	}

	_, ok = resourceCache[id]
	if !ok {
		return nil, errors.New("cannot find " + name + " '" + id + "' in recording")
	}

	return &llx.MockResource{Name: name, ID: id}, nil
}

func (m *Mock) WatchAndUpdate(resource llx.Resource, field string, watcherUID string, callback func(res interface{}, err error)) error {
	name := resource.MqlName()
	resourceCache, ok := m.Inventory[name]
	if !ok {
		return errors.New("resource '" + name + "' is not in recording")
	}

	id := resource.MqlID()
	x, ok := resourceCache[id]
	if !ok {
		return errors.New("cannot find " + name + " '" + id + "' in recording")
	}

	f, ok := x.Fields[field]
	if !ok {
		return errors.New("cannot find field '" + field + "' in resource " + name + " (id: " + id + ")")
	}

	if f.Error != "" {
		callback(nil, errors.New(f.Error))
	} else {
		callback(f.Data.RawData().Value, nil)
	}

	// nothing will change, so nothing to watch or unregister
	return nil
}

func (m *Mock) Resource(name string) (*resources.ResourceInfo, bool) {
	panic("not sure how to get resource info from mock yet...")
	return nil, false
}

func (m *Mock) Schema() llx.Schema {
	return m.schema
}

func (m *Mock) Close() {}

type emptySchema struct{}

func (e emptySchema) Lookup(resource string) *resources.ResourceInfo {
	return nil
}

func (e emptySchema) AllResources() map[string]*resources.ResourceInfo {
	return nil
}
