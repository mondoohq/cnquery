// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"strconv"

	"go.mondoo.com/cnquery/v10/llx"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v10/providers/opcua/connection"
)

func (o *mqlOpcuaNamespace) id() (string, error) {
	if o.Id.Error != nil {
		return "", o.Id.Error
	}
	s := strconv.FormatInt(o.Id.Data, 10)
	return "opcua.namespace/" + s, nil
}

// https://reference.opcfoundation.org/DI/v102/docs/11.2
func (o *mqlOpcua) namespaces() ([]interface{}, error) {
	conn := o.MqlRuntime.Connection.(*connection.OpcuaConnection)
	client := conn.Client()

	namespaces := client.Namespaces()
	resList := []interface{}{}
	for i := range namespaces {
		res, err := newMqlOpcuaNamespaceResource(o.MqlRuntime, int64(i), namespaces[i])
		if err != nil {
			return nil, err
		}
		resList = append(resList, res)
	}
	return resList, nil
}

func newMqlOpcuaNamespaceResource(runtime *plugin.Runtime, id int64, name string) (interface{}, error) {
	return CreateResource(runtime, "opcua.namespace", map[string]*llx.RawData{
		"id":   llx.IntData(id),
		"name": llx.StringData(name),
	})
}
