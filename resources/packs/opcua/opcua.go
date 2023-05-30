package opcua

import (
	"context"
	"errors"

	"github.com/gopcua/opcua/id"
	"github.com/gopcua/opcua/ua"
	"go.mondoo.com/cnquery/motor/providers"
	opcua_provider "go.mondoo.com/cnquery/motor/providers/opcua"
	"go.mondoo.com/cnquery/resources/packs/opcua/info"
)

var Registry = info.Registry

func init() {
	Init(Registry)
}

func opcuaProvider(p providers.Instance) (*opcua_provider.Provider, error) {
	at, ok := p.(*opcua_provider.Provider)
	if !ok {
		return nil, errors.New("OPC UA resource is not supported on this provider")
	}
	return at, nil
}

func (o *mqlOpcua) id() (string, error) {
	return "opcua", nil
}

func (o *mqlOpcua) GetRoot() (interface{}, error) {
	op, err := opcuaProvider(o.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	client := op.Client()

	ctx := context.Background()
	n := client.Node(ua.NewNumericNodeID(0, id.RootFolder))
	ndef, err := fetchNodeInfo(ctx, n)
	if err != nil {
		return nil, err
	}
	return newMqlOpcuaNodeResource(o.MotorRuntime, ndef)
}

func resolve(ctx context.Context, meta *nodeMeta) ([]*nodeMeta, error) {
	nodeList := []*nodeMeta{}

	for i := range meta.Organizes {
		child := meta.Organizes[i]
		nInfoChild, err := fetchNodeInfo(ctx, child)
		if err != nil {
			return nil, err
		}
		nodeList = append(nodeList, nInfoChild)
		resolved, err := resolve(ctx, nInfoChild)
		if err != nil {
			return nil, err
		}
		nodeList = append(nodeList, resolved...)
	}

	for i := range meta.Properties {
		child := meta.Properties[i]
		nInfoChild, err := fetchNodeInfo(ctx, child)
		if err != nil {
			return nil, err
		}
		nodeList = append(nodeList, nInfoChild)
		resolved, err := resolve(ctx, nInfoChild)
		if err != nil {
			return nil, err
		}
		nodeList = append(nodeList, resolved...)
	}

	for i := range meta.Components {
		child := meta.Components[i]
		nInfoChild, err := fetchNodeInfo(ctx, child)
		if err != nil {
			return nil, err
		}
		nodeList = append(nodeList, nInfoChild)
		resolved, err := resolve(ctx, nInfoChild)
		if err != nil {
			return nil, err
		}
		nodeList = append(nodeList, resolved...)
	}

	return nodeList, nil
}

func (o *mqlOpcua) GetNodes() ([]interface{}, error) {
	op, err := opcuaProvider(o.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	client := op.Client()

	ctx := context.Background()
	n := client.Node(ua.NewNumericNodeID(0, id.RootFolder))

	nodeList := []*nodeMeta{}
	nInfo, err := fetchNodeInfo(ctx, n)
	if err != nil {
		return nil, err
	}
	nodeList = append(nodeList, nInfo)
	resolved, err := resolve(ctx, nInfo)
	if err != nil {
		return nil, err
	}
	nodeList = append(nodeList, resolved...)

	// convert list to interface
	res := []interface{}{}
	for i := range nodeList {
		entry, err := newMqlOpcuaNodeResource(o.MotorRuntime, nodeList[i])
		if err != nil {
			return nil, err
		}
		res = append(res, entry)
	}
	return res, nil
}
