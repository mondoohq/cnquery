package k8s

import (
	"strings"

	"github.com/rs/zerolog/log"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
)

type KubernetesObjectInfo struct {
	Kind              string
	Name              string
	Namespace         string
	ClusterIdentifier string
}

// PlatformIdOwnershipDirectory indexes kubernetes object relationships by their constructed platform id
type PlatformIdOwnershipDirectory struct {
	// a map that contains the owners of each resolved k8s object
	ownedBy map[string][]string
	// a map that contains the k8s data for each object, both directly resolved and ones resolved via owner references
	metadataMap       map[string]KubernetesObjectInfo
	clusterIdentifier string
}

func NewKubernetesObjectInfo(clusterIdentifier, workloadType, namespace, name string) KubernetesObjectInfo {
	return KubernetesObjectInfo{
		Kind:              workloadType,
		Name:              name,
		Namespace:         namespace,
		ClusterIdentifier: clusterIdentifier,
	}
}

func NewEmptyPlatformIdOwnershipDirectory(clusterIdentifier string) *PlatformIdOwnershipDirectory {
	return &PlatformIdOwnershipDirectory{
		ownedBy:           make(map[string][]string),
		metadataMap:       make(map[string]KubernetesObjectInfo),
		clusterIdentifier: clusterIdentifier,
	}
}

func (od *PlatformIdOwnershipDirectory) Add(obj runtime.Object) {
	k8sMeta, err := meta.Accessor(obj)
	if err != nil {
		log.Error().Err(err).Msg("could not access object meta attributes")
		return
	}
	objType, err := meta.TypeAccessor(obj)
	if err != nil {
		log.Error().Err(err).Msg("could not access object type attributes")
		return
	}

	objPlatformId := NewPlatformWorkloadId(od.clusterIdentifier, strings.ToLower(objType.GetKind()), k8sMeta.GetNamespace(), k8sMeta.GetName())
	objMeta := NewKubernetesObjectInfo(od.clusterIdentifier, objType.GetKind(), k8sMeta.GetNamespace(), k8sMeta.GetName())

	od.metadataMap[objPlatformId] = objMeta
	for _, ownerRef := range k8sMeta.GetOwnerReferences() {
		ownerPlatformId := NewPlatformWorkloadId(od.clusterIdentifier, strings.ToLower(ownerRef.Kind), k8sMeta.GetNamespace(), ownerRef.Name)
		ownerMeta := NewKubernetesObjectInfo(od.clusterIdentifier, ownerRef.Kind, k8sMeta.GetNamespace(), ownerRef.Name)
		od.metadataMap[ownerPlatformId] = ownerMeta

		if _, ok := od.ownedBy[objPlatformId]; !ok {
			od.ownedBy[objPlatformId] = []string{}
		}
		od.ownedBy[objPlatformId] = append(od.ownedBy[objPlatformId], ownerPlatformId)
	}
}

// OwnedBy returns platform identifiers that own the specified platform id
func (od *PlatformIdOwnershipDirectory) OwnedBy(id string) []string {
	return od.ownedBy[id]
}

func (od *PlatformIdOwnershipDirectory) GetKubernetesObjectData(id string) (KubernetesObjectInfo, bool) {
	entry, ok := od.metadataMap[id]
	return entry, ok
}
