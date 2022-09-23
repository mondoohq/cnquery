// copyright: 2019, Dominik Richter and Christoph Hartmann
// author: Dominik Richter
// author: Christoph Hartmann

package resources

import (
	"encoding/json"
	"errors"
	fmt "fmt"
	"sort"
	"strings"

	"go.mondoo.com/cnquery/types"
)

// Args for initializing resources
type Args map[string]interface{}

type FieldFilter struct { // TODO: tbd
}

// Registry of all initialized resources
type Registry struct {
	Resources map[string]*ResourceCls
}

// NewRegistry creates a new instance of the resource registry and cache
func NewRegistry() *Registry {
	return &Registry{
		Resources: make(map[string]*ResourceCls),
	}
}

// Add all resources from another registry to this registry
func (ctx *Registry) Add(r *Registry) {
	for k, v := range r.Resources {
		ctx.Resources[k] = v
	}
}

// Clone creates a shallow copy of this registry, which means you can add/remove
// resources, but don't mess with their underlying configuration
func (ctx *Registry) Clone() *Registry {
	res := make(map[string]*ResourceCls, len(ctx.Resources))
	for k, v := range ctx.Resources {
		res[k] = v
	}
	return &Registry{res}
}

// LoadJson loads a set of resource definitions from JSON into the registry
func (ctx *Registry) LoadJson(raw []byte) error {
	schema := Schema{}
	if err := json.Unmarshal(raw, &schema); err != nil {
		return errors.New("cannot load embedded core resource schema")
	}

	// since we establish the resource chain of any missing resources,
	// it is important to add things in the right order (for now)
	keys := make([]string, len(schema.Resources))
	var i int
	for k := range schema.Resources {
		keys[i] = k
		i++
	}

	sort.Strings(keys)

	for i := range keys {
		isAlias := keys[i] != schema.Resources[keys[i]].Id
		if err := ctx.AddResourceInfo(schema.Resources[keys[i]]); err != nil {
			return errors.New("failed to add resource info: " + err.Error())
		}
		if isAlias {
			info := ctx.Resources[schema.Resources[keys[i]].Id]
			ctx.Resources[keys[i]] = info
			ctx.ensureResourceChain(keys[i], info.Private, isAlias)
		}
	}

	return nil
}

// for a given resource name, make sure all parent resources exist
// e.g. sshd.config ==> make sure sshd exists
func (ctx *Registry) ensureResourceChain(name string, isPrivate, isAlias bool) {
	parts := strings.Split(name, ".")
	if len(parts) == 1 {
		return
	}
	cur := parts[0]
	for i := 0; i < len(parts)-1; i++ {
		o, ok := ctx.Resources[cur]
		// it can be that we're trying to lookup an alias that doesn't have the parent defined
		if !ok && isAlias {
			o, ok = ctx.Resources[parts[i]]
			if ok {
				ctx.Resources[cur] = o
			}
		}
		if !ok {
			o = newResourceCls(cur)
			ctx.Resources[cur] = o
			// parent resources get the visibility of their children by default
			// any public child overwrites the rest for the parent (see below)
			o.Private = isPrivate
		}

		// we may need to overwrite parent resource declaration if we realize the child is public
		if !isPrivate {
			o.Private = false
		}
		next := cur + "." + parts[i+1]

		f, ok := o.Fields[parts[i+1]]
		if !ok {
			f = &Field{
				Name:               parts[i+1],
				Type:               string(types.Resource(next)),
				IsMandatory:        false,
				IsImplicitResource: true,
				Refs:               []string{},
				IsPrivate:          isPrivate,
			}
			o.Fields[parts[i+1]] = f
		}
		// same as above: if any child is public, the field in the chain must become public
		if !isPrivate {
			f.IsPrivate = isPrivate
		}

		cur = next
	}
}

func mergeResourceInfoPartial(a *ResourceCls, b *ResourceInfo) error {
	if a.Id != b.Id {
		return fmt.Errorf("could not merge resources %s and %s",
			a.Id, b.Id)
	}

	if a.Name != b.Name {
		return fmt.Errorf("could not merge resources %s and %s because names differ",
			a.Id, b.Id)
	}

	if a.ListType != b.ListType {
		return fmt.Errorf("could not merge resources %s and %s because list type does not match",
			a.Id, b.Id)
	}

	if a.Private != b.Private {
		return fmt.Errorf("could not merge resources %s and %s because private modifier does not match",
			a.Id, b.Id)
	}

	if a.Title == "" {
		a.Title = b.Title
	}

	if a.Desc == "" {
		a.Desc = b.Desc
	}

	if a.MinMondooVersion == "" {
		a.MinMondooVersion = b.MinMondooVersion
	}

	for _, f := range b.Fields {
		if _, ok := a.Fields[f.Name]; !ok {
			a.Fields[f.Name] = f
		}
	}

	return nil
}

func (ctx *Registry) AddResourceInfo(info *ResourceInfo) error {
	name := info.Id

	// NOTE: we do not yet merge resources! So error for now.
	if r, ok := ctx.Resources[name]; ok {
		if err := mergeResourceInfoPartial(r, info); err != nil {
			return err
		}
		return nil
	}

	if info.Fields == nil {
		info.Fields = map[string]*Field{}
	}

	ctx.Resources[name] = &ResourceCls{
		ResourceInfo: *info,
	}

	ctx.ensureResourceChain(name, info.Private, false)

	return nil
}

// Add a new resource with a factory for creating an instance
func (ctx *Registry) AddFactory(name string, factory ResourceFactory) error {
	if name == "" {
		return errors.New("trying to add factory for a resource without a name")
	}

	resource, ok := ctx.Resources[name]
	if !ok {
		return errors.New("resource '" + name + "' cannot be found")
	}

	resource.Factory = factory
	return nil
}

// Names all resources
func (ctx *Registry) Names() []string {
	res := make([]string, len(ctx.Resources))
	i := 0
	for key := range ctx.Resources {
		res[i] = key
		i++
	}
	return res
}

// Fields of a resource
func (ctx *Registry) Fields(name string) (map[string]*Field, error) {
	r, ok := ctx.Resources[name]
	if !ok {
		return nil, errors.New("Failed to get fields for resource " + name + ", couldn't find a resource with that name")
	}
	return r.Fields, nil
}

// Schema of all loaded resources
func (ctx *Registry) Schema() *Schema {
	res := Schema{Resources: make(map[string]*ResourceInfo)}
	for id, i := range ctx.Resources {
		res.Resources[id] = &i.ResourceInfo
	}
	return &res
}
