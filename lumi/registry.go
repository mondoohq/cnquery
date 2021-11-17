// copyright: 2019, Dominik Richter and Christoph Hartmann
// author: Dominik Richter
// author: Christoph Hartmann

package lumi

import (
	"errors"
	"strings"

	"go.mondoo.io/mondoo/types"
)

// Args for initializing resources
type Args map[string]interface{}

type FieldFilter struct {
	// TODO: tbd
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

// Clone creates a shallow copy of this registry, which means you can add/remove
// resources, but don't mess with their underlying configuration
func (ctx *Registry) Clone() *Registry {
	res := make(map[string]*ResourceCls, len(ctx.Resources))
	for k, v := range ctx.Resources {
		res[k] = v
	}
	return &Registry{res}
}

// for a given resource name, make sure all parent resources exist
// e.g. sshd.config ==> make sure sshd exists
func (ctx *Registry) ensureResourceChain(name string, isPrivate bool) {
	parts := strings.Split(name, ".")
	if len(parts) == 1 {
		return
	}
	cur := parts[0]
	for i := 0; i < len(parts)-1; i++ {
		o, ok := ctx.Resources[cur]
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
				Name:      parts[i+1],
				Type:      string(types.Resource(next)),
				Mandatory: false,
				Refs:      []string{},
				Private:   isPrivate,
			}
			o.Fields[parts[i+1]] = f
		}
		// same as above: if any child is public, the field in the chain must become public
		if !isPrivate {
			f.Private = isPrivate
		}

		cur = next
	}
}

// Add a new resource with a factory for creating an instance
func (ctx *Registry) Add(resource *ResourceCls) error {
	name := resource.Name
	if name == "" {
		return errors.New("trying to define a resource without a name")
	}
	_, ok := ctx.Resources[name]
	if ok {
		return errors.New("resource '" + name + "' is redefined.")
	}

	ctx.Resources[name] = resource
	ctx.ensureResourceChain(name, resource.ResourceInfo.Private)
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
