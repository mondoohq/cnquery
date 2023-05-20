package opcua

import (
	"context"
	"errors"
	"fmt"
	"github.com/gopcua/opcua/id"
	"github.com/gopcua/opcua/ua"
	"go.mondoo.com/cnquery/motor/providers"
	opcua_provider "go.mondoo.com/cnquery/motor/providers/opcua"
	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/resources/packs/core"
	"go.mondoo.com/cnquery/resources/packs/opcua/info"
)

var Registry = info.Registry

func init() {
	Init(Registry)
}

func opcuaProvider(p providers.Instance) (*opcua_provider.Provider, error) {
	at, ok := p.(*opcua_provider.Provider)
	if !ok {
		return nil, errors.New("okta resource is not supported on this provider")
	}
	return at, nil
}

func (o *mqlOpcua) id() (string, error) {
	return "opcua", nil
}

// https://reference.opcfoundation.org/DI/v102/docs/11.2
func (o *mqlOpcua) GetNamespaces() ([]interface{}, error) {
	op, err := opcuaProvider(o.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	client := op.Client()

	namespaces := client.Namespaces()
	return core.StrSliceToInterface(namespaces), nil
}

func newMqlOpcuaNodeResource(runtime *resources.Runtime, ndef *NodeDef) (interface{}, error) {
	res, err := runtime.CreateResource("opcua.node",
		"id", ndef.NodeID.String(),
		"name", ndef.BrowseName,
		"class", ndef.NodeClass.String(),
		"description", ndef.Description,
		"writeable", ndef.Writable,
		"dataType", ndef.DataType,
		"min", ndef.Min,
		"max", ndef.Max,
		"unit", ndef.Unit,
		"accessLevel", ndef.AccessLevel.String(),
	)
	if err != nil {
		return nil, err
	}
	res.MqlResource().Cache.Store("_object", &resources.CacheEntry{
		Data: ndef,
	})
	return res, nil
}

func (o *mqlOpcua) GetRoot() (interface{}, error) {
	op, err := opcuaProvider(o.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	client := op.Client()

	n := client.Node(ua.NewNumericNodeID(0, id.RootFolder))
	ctx := context.Background()
	ndef, err := fetchNodeInfo(ctx, n)
	if err != nil {
		return nil, err
	}

	return newMqlOpcuaNodeResource(o.MotorRuntime, ndef)
}

func (o *mqlOpcuaNode) id() (string, error) {
	id, err := o.Id()
	if err != nil {
		return "", err
	}
	return "opcua.node/" + id, nil
}

func (o *mqlOpcuaNode) GetNamespace() (interface{}, error) {
	return nil, nil
}

func (o *mqlOpcuaNode) GetProperties() ([]interface{}, error) {
	res, ok := o.Cache.Load("_object")
	if !ok {
		return nil, errors.New("could not fetch properties")
	}
	if res.Error != nil {
		return nil, res.Error
	}
	nodeDef, ok := res.Data.(*NodeDef)
	if !ok {
		return nil, fmt.Errorf("\"opcua\" failed to cast field \"node\" to the right type: %#v", res)
	}

	ctx := context.Background()
	results := []interface{}{}
	for i := range nodeDef.Properties {
		def := nodeDef.Properties[i]
		n, err := fetchNodeInfo(ctx, def)
		if err != nil {
			return nil, err
		}
		r, err := newMqlOpcuaNodeResource(o.MotorRuntime, n)
		if err != nil {
			return nil, err
		}
		results = append(results, r)
	}

	return results, nil
}

func (o *mqlOpcuaNode) GetComponents() ([]interface{}, error) {
	res, ok := o.Cache.Load("_object")
	if !ok {
		return nil, errors.New("could not fetch properties")
	}
	if res.Error != nil {
		return nil, res.Error
	}
	nodeDef, ok := res.Data.(*NodeDef)
	if !ok {
		return nil, fmt.Errorf("\"opcua\" failed to cast field \"node\" to the right type: %#v", res)
	}

	ctx := context.Background()
	results := []interface{}{}
	for i := range nodeDef.Components {
		def := nodeDef.Components[i]
		n, err := fetchNodeInfo(ctx, def)
		if err != nil {
			return nil, err
		}
		r, err := newMqlOpcuaNodeResource(o.MotorRuntime, n)
		if err != nil {
			return nil, err
		}
		results = append(results, r)
	}

	return results, nil
}

func (o *mqlOpcuaNode) GetOrganizes() ([]interface{}, error) {
	res, ok := o.Cache.Load("_object")
	if !ok {
		return nil, errors.New("could not fetch properties")
	}
	if res.Error != nil {
		return nil, res.Error
	}
	nodeDef, ok := res.Data.(*NodeDef)
	if !ok {
		return nil, fmt.Errorf("\"opcua\" failed to cast field \"node\" to the right type: %#v", res)
	}

	ctx := context.Background()
	results := []interface{}{}
	for i := range nodeDef.Organizes {
		def := nodeDef.Organizes[i]
		n, err := fetchNodeInfo(ctx, def)
		if err != nil {
			return nil, err
		}
		r, err := newMqlOpcuaNodeResource(o.MotorRuntime, n)
		if err != nil {
			return nil, err
		}
		results = append(results, r)
	}

	return results, nil
}

func (o *mqlOpcuaNamespace) id() (string, error) {
	return "opcua.namespace", nil
}
