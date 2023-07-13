package opcua

import (
	"strconv"

	"go.mondoo.com/cnquery/resources"
)

func (o *mqlOpcuaNamespace) id() (string, error) {
	id, err := o.Id()
	if err != nil {
		return "", err
	}
	s := strconv.FormatInt(id, 10)
	return "opcua.namespace/" + s, nil
}

// https://reference.opcfoundation.org/DI/v102/docs/11.2
func (o *mqlOpcua) GetNamespaces() ([]interface{}, error) {
	op, err := opcuaProvider(o.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	client := op.Client()

	namespaces := client.Namespaces()
	resList := []interface{}{}
	for i := range namespaces {
		res, err := newMqlOpcuaNamespaceResource(o.MotorRuntime, int64(i), namespaces[i])
		if err != nil {
			return nil, err
		}
		resList = append(resList, res)
	}
	return resList, nil
}

func newMqlOpcuaNamespaceResource(runtime *resources.Runtime, id int64, name string) (interface{}, error) {
	return runtime.CreateResource("opcua.namespace",
		"id", id,
		"name", name,
	)
}
