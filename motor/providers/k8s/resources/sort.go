package resources

import (
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
)

// ByKindNameNamespace sorts objects by 'Kind', 'Name' & 'Namespace'
type ByKindNameNamespace []runtime.Object

func (s ByKindNameNamespace) Len() int {
	return len(s)
}

func (s ByKindNameNamespace) Less(i, j int) bool {
	a, err := meta.Accessor(s[i])
	if err != nil {
		return true
	}
	at, err := meta.TypeAccessor(s[i])
	if err != nil {
		return true
	}
	b, err := meta.Accessor(s[j])
	if err != nil {
		return false
	}
	bt, err := meta.TypeAccessor(s[j])
	if err != nil {
		return false
	}

	if at.GetKind() != bt.GetKind() {
		return at.GetKind() < bt.GetKind()
	}
	if a.GetName() != b.GetName() {
		return a.GetName() < b.GetName()
	}
	return a.GetNamespace() < b.GetNamespace()
}

func (s ByKindNameNamespace) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
