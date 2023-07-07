package opcua

import (
	"context"

	"github.com/gopcua/opcua/id"
	"github.com/gopcua/opcua/ua"
	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/resources/packs/core"
)

func (o *mqlOpcuaServer) init(args *resources.Args) (*resources.Args, OpcuaServer, error) {
	op, err := opcuaProvider(o.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, nil, err
	}
	client := op.Client()

	ctx := context.Background()

	n := client.Node(ua.NewNumericNodeID(0, id.Server))
	ndef, err := fetchNodeInfo(ctx, n)
	if err != nil {
		return nil, nil, err
	}

	// create server resource
	serverNode, err := newMqlOpcuaNodeResource(o.MotorRuntime, ndef)
	if err != nil {
		return nil, nil, err
	}
	(*args)["node"] = serverNode

	// server status variable of server
	v, err := client.Node(ua.NewNumericNodeID(0, id.Server_ServerStatus)).Value()
	switch {
	case err != nil:
		return nil, nil, err
	case v == nil:
		(*args)["buildInfo"] = nil
		(*args)["currentTime"] = nil
		(*args)["startTime"] = nil
		(*args)["state"] = ""
	default:
		res := v.Value()
		extensionObject := res.(*ua.ExtensionObject)
		serverStatus := extensionObject.Value.(*ua.ServerStatusDataType)

		buildInfo, _ := core.JsonToDict(serverStatus.BuildInfo)
		(*args)["buildInfo"] = buildInfo
		(*args)["currentTime"] = &serverStatus.CurrentTime
		(*args)["startTime"] = &serverStatus.StartTime
		(*args)["state"] = serverStatus.State.String()
	}

	return args, nil, nil
}

func (o *mqlOpcuaServer) id() (string, error) {
	node, err := o.Node()
	if err != nil {
		return "", err
	}

	id, err := node.Id()
	if err != nil {
		return "", err
	}

	return "opcua.server/" + id, nil
}
