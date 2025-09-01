// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package explorer

import "go.mondoo.com/cnquery/v12/llx"

// PropsResolver helps look up properties between local query properties
// and possible bundle/pack properties
type PropsResolver struct {
	QueryCache  map[string]PropertyRef
	GlobalCache map[string]PropertyRef
	Query       *Mquery
}

// Available lists only immediately available query properties
// (i.e. doesn't look at bundle/pack properties)
func (p *PropsResolver) Available() map[string]*llx.Primitive {
	res := make(map[string]*llx.Primitive, len(p.QueryCache))
	for k, v := range p.QueryCache {
		res[k] = &llx.Primitive{Type: v.Property.Type}
	}
	return res
}

// All lists all possible properties, including bundle/pack props
func (p *PropsResolver) All() map[string]*llx.Primitive {
	// note: this allocates possibly too much space, because we don't need as much
	// but the call is in errors and auto-complete, so it's rarer
	res := make(map[string]*llx.Primitive, len(p.QueryCache)+len(p.GlobalCache))
	for k, v := range p.GlobalCache {
		res[k] = &llx.Primitive{Type: v.Property.Type}
	}
	for k, v := range p.QueryCache {
		res[k] = &llx.Primitive{Type: v.Property.Type}
	}
	return res
}

// Try to get a named property. If it isn't available on the query, try
// to look it up in the bundle/pack. If it is available there, embed it into the
// query properties.
func (p *PropsResolver) Get(name string) *llx.Primitive {
	// Property lookup happens on 2 layers:
	// 1. the property is part of the query definition, we are ll set
	// 2. the property is part of the pack definition, then we have to add it to the query
	ref, ok := p.QueryCache[name]
	if ok {
		return &llx.Primitive{Type: ref.Property.Type}
	}

	ref, ok = p.GlobalCache[name]
	if !ok {
		return nil
	}
	p.Query.Props = append(p.Query.Props, ref.Property)
	return &llx.Primitive{Type: ref.Property.Type}
}
