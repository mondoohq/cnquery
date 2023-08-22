// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package opcua

import (
	"context"
	"fmt"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/errors"
	"github.com/gopcua/opcua/id"
	"github.com/gopcua/opcua/ua"
	"go.mondoo.com/cnquery/resources"
)

type nodeMeta struct {
	NodeID      *ua.NodeID
	NodeClass   ua.NodeClass
	BrowseName  string
	Description string
	AccessLevel ua.AccessLevelType
	Path        string
	DataType    string
	Writable    bool
	Unit        string
	Scale       string
	Min         string
	Max         string
	Components  []*opcua.Node
	Organizes   []*opcua.Node
	Properties  []*opcua.Node
}

func fetchNodeInfo(ctx context.Context, n *opcua.Node) (*nodeMeta, error) {
	attrs, err := n.AttributesWithContext(ctx, ua.AttributeIDNodeClass, ua.AttributeIDBrowseName, ua.AttributeIDDescription, ua.AttributeIDAccessLevel, ua.AttributeIDDataType)
	if err != nil {
		return nil, err
	}

	def := nodeMeta{
		NodeID: n.ID,
	}

	switch err := attrs[0].Status; err {
	case ua.StatusOK:
		def.NodeClass = ua.NodeClass(attrs[0].Value.Int())
	default:
		return nil, err
	}

	switch err := attrs[0].Status; err {
	case ua.StatusOK:
		def.NodeClass = ua.NodeClass(attrs[0].Value.Int())
	default:
		return nil, err
	}

	switch err := attrs[1].Status; err {
	case ua.StatusOK:
		def.BrowseName = attrs[1].Value.String()
	default:
		return nil, err
	}

	switch err := attrs[2].Status; err {
	case ua.StatusOK:
		def.Description = attrs[2].Value.String()
	case ua.StatusBadAttributeIDInvalid:
		// ignore
	default:
		return nil, err
	}

	switch err := attrs[3].Status; err {
	case ua.StatusOK:
		def.AccessLevel = ua.AccessLevelType(attrs[3].Value.Int())
		def.Writable = def.AccessLevel&ua.AccessLevelTypeCurrentWrite == ua.AccessLevelTypeCurrentWrite
	case ua.StatusBadAttributeIDInvalid:
		// ignore
	default:
		return nil, err
	}

	switch err := attrs[4].Status; err {
	case ua.StatusOK:
		switch v := attrs[4].Value.NodeID().IntID(); v {
		case id.DateTime:
			def.DataType = "time.Time"
		case id.Boolean:
			def.DataType = "bool"
		case id.SByte:
			def.DataType = "int8"
		case id.Int16:
			def.DataType = "int16"
		case id.Int32:
			def.DataType = "int32"
		case id.Byte:
			def.DataType = "byte"
		case id.UInt16:
			def.DataType = "uint16"
		case id.UInt32:
			def.DataType = "uint32"
		case id.UtcTime:
			def.DataType = "time.Time"
		case id.String:
			def.DataType = "string"
		case id.Float:
			def.DataType = "float32"
		case id.Double:
			def.DataType = "float64"
		default:
			def.DataType = attrs[4].Value.NodeID().String()
		}
	case ua.StatusBadAttributeIDInvalid:
		// ignore
	default:
		return nil, err
	}

	// TODO: set path

	fetchReference := func(refType uint32) ([]*opcua.Node, error) {
		return n.ReferencedNodesWithContext(ctx, refType, ua.BrowseDirectionForward, ua.NodeClassAll, true)
	}

	if componentRefs, err := fetchReference(id.HasComponent); err != nil {
		return nil, err
	} else {
		def.Components = append(def.Components, componentRefs...)
	}

	if componentRefs, err := fetchReference(id.Organizes); err != nil {
		return nil, err
	} else {
		def.Organizes = append(def.Organizes, componentRefs...)
	}

	if componentRefs, err := fetchReference(id.HasProperty); err != nil {
		return nil, err
	} else {
		def.Properties = append(def.Properties, componentRefs...)
	}

	return &def, nil
}

func newMqlOpcuaNodeResource(runtime *resources.Runtime, ndef *nodeMeta) (interface{}, error) {
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

func (o *mqlOpcuaNode) id() (string, error) {
	id, err := o.Id()
	if err != nil {
		return "", err
	}
	return "opcua.node/" + id, nil
}

func (o *mqlOpcuaNode) GetNamespace() (interface{}, error) {
	res, ok := o.Cache.Load("_object")
	if !ok {
		return nil, errors.New("could not fetch properties")
	}

	if res.Error != nil {
		return nil, res.Error
	}
	nodeDef, ok := res.Data.(*nodeMeta)
	if !ok {
		return nil, fmt.Errorf("\"opcua\" failed to cast field \"node\" to the right type: %#v", res)
	}

	obj, err := o.MotorRuntime.CreateResource("opcua")
	if err != nil {
		return nil, err
	}
	mqlOpcua := obj.(Opcua)

	namespaces, err := mqlOpcua.Namespaces()
	if err != nil {
		return nil, err
	}

	entry := namespaces[nodeDef.NodeID.Namespace()]
	return entry, nil
}

func (o *mqlOpcuaNode) GetProperties() ([]interface{}, error) {
	res, ok := o.Cache.Load("_object")
	if !ok {
		return nil, errors.New("could not fetch properties")
	}
	if res.Error != nil {
		return nil, res.Error
	}
	nodeDef, ok := res.Data.(*nodeMeta)
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
	nodeDef, ok := res.Data.(*nodeMeta)
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
	nodeDef, ok := res.Data.(*nodeMeta)
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
