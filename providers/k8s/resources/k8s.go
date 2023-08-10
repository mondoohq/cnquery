package resources

import "sync"

type mqlK8sInternal struct {
	lock        sync.Mutex
	nodesByName map[string]*mqlK8sNode
}
