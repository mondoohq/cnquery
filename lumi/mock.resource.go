// copyright: 2019, Dominik Richter and Christoph Hartmann
// author: Dominik Richter
// author: Christoph Hartmann

package lumi

import (
	"errors"
)

// MockResource helps mocking resources with fields with static data
type MockResource struct {
	StaticFields   map[string]*Field
	StaticResource *Resource
}

// LumiResource provides static resource information
func (m MockResource) LumiResource() *Resource {
	return m.StaticResource
}

// Fields lists all fields of the mock resource
func (m MockResource) Fields() []*Field {
	res := []*Field{}
	for _, f := range m.StaticFields {
		res = append(res, f)
	}
	return res
}

// Field retrieves the current value of a field
func (m MockResource) Field(name string) (interface{}, error) {
	f, ok := m.StaticFields[name]
	if !ok {
		return nil, errors.New("cannot find field " + name)
	}
	return f, nil
}

// Register a field and all its callbacks
func (m MockResource) Register(field string) error {
	_, ok := m.StaticFields[field]
	if !ok {
		return errors.New("cannot find field " + field)
	}
	return m.StaticResource.Runtime.Observers.Trigger(m.LumiResource().FieldUID(field))
}

// Compute a field. For mock, all fields are always computed
func (m MockResource) Compute(field string) error {
	_, ok := m.StaticFields[field]
	if !ok {
		return errors.New("cannot find field " + field)
	}
	return nil
}

// Validate has nothing to do in mock, everything is valid.
func (m MockResource) Validate() error {
	return nil
}
