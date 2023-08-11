package resources

import (
	"sync"

	"go.mondoo.com/cnquery/providers-sdk/v1/util/convert"
)

type mqlK8sInternal struct {
	lock        sync.Mutex
	nodesByName map[string]*mqlK8sNode
}

func (k *mqlK8s) serverVersion() (interface{}, error) {
	kt, err := k8sProvider(k.MqlRuntime.Connection)
	if err != nil {
		return nil, err
	}

	return convert.JsonToDict(kt.ServerVersion())
}
