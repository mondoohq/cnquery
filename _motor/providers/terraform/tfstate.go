package terraform

import "encoding/json"

// This is designed around https://www.terraform.io/internals/json-format
// NOTE: it is very similar to the plan file format, but not exactly the same.

type State struct {
	FormatVersion    string       `json:"format_version,omitempty"`
	TerraformVersion string       `json:"terraform_version,omitempty"`
	Values           *StateValues `json:"values,omitempty"`
}

// StateValues is the representation of resolved values
type StateValues struct {
	Outputs    map[string]*Output `json:"outputs,omitempty"`
	RootModule *Module            `json:"root_module,omitempty"`
}

type Output struct {
	Sensitive bool            `json:"sensitive"`
	Value     json.RawMessage `json:"value,omitempty"`
	Type      json.RawMessage `json:"type,omitempty"`
}

// Module is the representation of a module in state. It can be the root module
// or a child module
type Module struct {
	// Address is the absolute module address, omitted for the root module
	Address      string      `json:"address,omitempty"`
	Resources    []*Resource `json:"resources,omitempty"`
	ChildModules []*Module   `json:"child_modules,omitempty"`
}

// WalkChildModules recursively walks the child modules and calls the callback
func (m *Module) WalkChildModules(walker func(m *Module)) {
	for _, child := range m.ChildModules {
		walker(child)
		child.WalkChildModules(walker)
	}
}

// Resource is the representation of a resource in the state
type Resource struct {
	// Address is the absolute resource address
	Address string `json:"address,omitempty"`

	// Mode can be "managed" or "data"
	Mode string `json:"mode,omitempty"`

	Type          string `json:"type,omitempty"`
	Name          string `json:"name,omitempty"`
	ProviderName  string `json:"provider_name"`
	SchemaVersion uint64 `json:"schema_version"`

	// AttributeValues is the JSON representation of the attribute values.
	// The structure depends on the resource type schema
	AttributeValues map[string]interface{} `json:"values,omitempty"`

	// SensitiveValues is similar to AttributeValues, but with all sensitive
	// values replaced with true
	SensitiveValues json.RawMessage `json:"sensitive_values,omitempty"`

	// DependsOn contains a list of the resource's dependencies
	DependsOn []string `json:"depends_on,omitempty"`

	// Tainted is true if the resource is tainted in terraform state
	Tainted bool `json:"tainted,omitempty"`

	// Deposed is set if the resource is deposed in terraform state
	DeposedKey string `json:"deposed_key,omitempty"`
}
