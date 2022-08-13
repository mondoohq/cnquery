package k8s

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"go.mondoo.io/mondoo/motor/platform"
	"go.mondoo.io/mondoo/motor/providers"
	"go.mondoo.io/mondoo/motor/providers/k8s"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestListStatefulSets(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	transport := k8s.NewMockKubernetesProvider(mockCtrl)

	statefulSetPlatform := &platform.Platform{
		Name:    "k8s-statefulset",
		Title:   "Kubernetes StatefulSet",
		Family:  []string{"k8s", "k8s-workload"},
		Kind:    providers.Kind_KIND_K8S_OBJECT,
		Runtime: providers.RUNTIME_KUBERNETES_CLUSTER,
	}

	// Seed namespaces
	nss := []corev1.Namespace{
		{ObjectMeta: metav1.ObjectMeta{Name: "default"}},
	}
	transport.EXPECT().Namespaces().Return(nss, nil)
	// called for each StatefulSet
	transport.EXPECT().PlatformInfo().Return(statefulSetPlatform)
	transport.EXPECT().PlatformInfo().Return(statefulSetPlatform)

	// Seed StatefulSets
	statefulsets := []appsv1.StatefulSet{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "nginx",
				Namespace: nss[0].Name,
				UID:       "123",
			},
			Spec: appsv1.StatefulSetSpec{
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"app": "nginx",
						},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{{Image: "nginx:1.22.0-alpine"}},
					},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "nginx2",
				Namespace: nss[0].Name,
				UID:       "456",
			},
			Spec: appsv1.StatefulSetSpec{
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"app": "nginx",
						},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{{Image: "nginx:1.22.0-alpine"}},
					},
				},
			},
		},
	}

	transport.EXPECT().StatefulSets(nss[0]).Return(statefulsets, nil)

	expectedAssetNames := []string{
		nss[0].Name + "/" + statefulsets[0].Name,
		nss[0].Name + "/" + statefulsets[1].Name,
	}

	clusterIdentifier := "//platformid.api.mondoo.app/runtime/k8s/uid/e26043bb-8669-48a2-b684-b1e132198cdc"

	expectedAssetPlatformIds := []string{
		clusterIdentifier + "/namespace/" + nss[0].Name + "/statefulsets/name/" + statefulsets[0].Name,
		clusterIdentifier + "/namespace/" + nss[0].Name + "/statefulsets/name/" + statefulsets[1].Name,
	}

	tc := &providers.TransportConfig{}
	assets, err := ListStatefulSets(transport, tc, clusterIdentifier, nil)
	assert.NoError(t, err)

	var assetNames []string
	for _, a := range assets {
		assetNames = append(assetNames, a.Name)
	}

	var assetPlatformIds []string
	for _, a := range assets {
		assetPlatformIds = append(assetPlatformIds, a.PlatformIds[0])
	}

	assert.ElementsMatch(t, expectedAssetNames, assetNames)
	assert.ElementsMatch(t, expectedAssetPlatformIds, assetPlatformIds)
}
