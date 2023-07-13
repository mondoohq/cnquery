package resources

import (
	"sort"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
)

// ResourceOwnershipDirectory indexes resources and their owner relationships
type ResourceOwnershipDirectory struct {
	index     map[types.UID]runtime.Object
	ownership map[types.UID][]types.UID
}

// NewResourceOwnershipDirectory optimizes the Unstructures for quick access of ownerships
func NewResourceOwnershipDirectory(objs []runtime.Object) ResourceOwnershipDirectory {
	v := ResourceOwnershipDirectory{
		index:     make(map[types.UID]runtime.Object),
		ownership: make(map[types.UID][]types.UID),
	}
	for _, obj := range objs {
		o, err := meta.Accessor(obj)
		if err != nil {
			klog.V(1).Infof("could not access object attributes: %v", err)
			continue
		}
		uid := o.GetUID()
		v.index[uid] = obj
		for _, ownerRef := range o.GetOwnerReferences() {
			if v.ownership[ownerRef.UID] == nil {
				v.ownership[ownerRef.UID] = []types.UID{}
			}
			v.ownership[ownerRef.UID] = append(v.ownership[ownerRef.UID], uid)
		}
	}
	return v
}

// GetResource finds resource by ID
func (od ResourceOwnershipDirectory) GetResource(id types.UID) runtime.Object {
	return od.index[id]
}

// OwnedBy returns resources that own the specified resource id and sorts by Kind, then by Name, then by Namespace
func (od ResourceOwnershipDirectory) OwnedBy(id types.UID) []runtime.Object {
	var out []runtime.Object
	for i := range od.ownership[id] {
		out = append(out, od.GetResource(od.ownership[id][i]))
	}
	sort.Sort(ByKindNameNamespace(out))
	return out
}
