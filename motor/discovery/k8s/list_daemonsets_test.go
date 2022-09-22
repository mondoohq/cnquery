package k8s

import (
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/motor/providers"
	"go.mondoo.com/cnquery/motor/providers/k8s"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	appsv1 "k8s.io/api/apps/v1"
)

func TestListDaemonsets(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	p := k8s.NewMockKubernetesProvider(mockCtrl)

	// Seed namespaces
	nss := []corev1.Namespace{
		{ObjectMeta: metav1.ObjectMeta{Name: "default"}},
	}
	p.EXPECT().Namespaces().Return(nss, nil)
	// called for each DaemonSet
	p.EXPECT().Runtime().Return("k8s-cluster")
	p.EXPECT().Runtime().Return("k8s-cluster")

	// pretend daemon set owned by deployment
	parent := appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "nginx-deployment",
			Namespace: nss[0].Name,
			UID:       "000",
		},
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{Image: "nginx:1.22.0-alpine"}},
				},
			},
		},
	}

	// Seed DaemonSets
	daemonsets := []appsv1.DaemonSet{
		{
			TypeMeta: metav1.TypeMeta{
				Kind:       "DaemonSet",
				APIVersion: "apps/v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "nginx",
				Namespace: nss[0].Name,
				UID:       "123",
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: parent.APIVersion,
						Kind:       parent.Kind,
						Name:       parent.Name,
						UID:        parent.UID,
					},
				},
			},
			Spec: appsv1.DaemonSetSpec{
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{{Image: "nginx:1.22.0-alpine"}},
					},
				},
			},
		},
		{
			TypeMeta: metav1.TypeMeta{
				Kind:       "DaemonSet",
				APIVersion: "apps/v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "nginx2",
				Namespace: nss[0].Name,
				UID:       "456",
			},
			Spec: appsv1.DaemonSetSpec{
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{{Image: "nginx:1.22.0-alpine"}},
					},
				},
			},
		},
	}

	p.EXPECT().DaemonSets(nss[0]).Return(daemonsets, nil)

	expectedAssetNames := []string{
		nss[0].Name + "/" + daemonsets[0].Name,
		nss[0].Name + "/" + daemonsets[1].Name,
	}

	clusterIdentifier := "//platformid.api.mondoo.app/runtime/k8s/uid/e26043bb-8669-48a2-b684-b1e132198cdc"

	expectedAssetPlatformIds := []string{
		clusterIdentifier + "/namespace/" + nss[0].Name + "/daemonsets/name/" + daemonsets[0].Name,
		clusterIdentifier + "/namespace/" + nss[0].Name + "/daemonsets/name/" + daemonsets[1].Name,
	}

	pCfg := &providers.Config{}
	ownershipDir := k8s.NewEmptyPlatformIdOwnershipDirectory(clusterIdentifier)
	assets, err := ListDaemonSets(p, pCfg, clusterIdentifier, nil, make(map[string][]K8sResourceIdentifier), ownershipDir)
	require.NoError(t, err)
	require.Equal(t, []string{k8s.NewPlatformWorkloadId(clusterIdentifier,
		strings.ToLower(parent.Kind),
		parent.Namespace,
		parent.Name)},
		ownershipDir.OwnedBy(expectedAssetPlatformIds[0]))

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
	assert.Equal(t, "apps/v1", assets[0].Platform.Version)
	assert.Equal(t, "k8s-daemonset", assets[0].Platform.Name)
	assert.ElementsMatch(t, []string{"k8s", "k8s-workload"}, assets[0].Platform.Family)
	assert.Equal(t, nss[0].Name, assets[0].Labels["k8s.mondoo.com/namespace"])
}

func TestListDaemonsets_Filter(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	p := k8s.NewMockKubernetesProvider(mockCtrl)

	// called for each DaemonSet
	p.EXPECT().Runtime().Return("k8s-cluster")

	// Seed DaemonSets
	daemonsets := []appsv1.DaemonSet{
		{
			TypeMeta: metav1.TypeMeta{
				Kind:       "DaemonSet",
				APIVersion: "apps/v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "nginx",
				Namespace: "default",
				UID:       "123",
			},
			Spec: appsv1.DaemonSetSpec{
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{{Image: "nginx:1.22.0-alpine"}},
					},
				},
			},
		},
		{
			TypeMeta: metav1.TypeMeta{
				Kind:       "DaemonSet",
				APIVersion: "apps/v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "nginx2",
				Namespace: "default",
				UID:       "456",
			},
			Spec: appsv1.DaemonSetSpec{
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{{Image: "nginx:1.22.0-alpine"}},
					},
				},
			},
		},
	}

	p.EXPECT().DaemonSet(daemonsets[0].Namespace, daemonsets[0].Name).Return(&daemonsets[0], nil)

	expectedAssetNames := []string{
		daemonsets[0].Namespace + "/" + daemonsets[0].Name,
	}

	clusterIdentifier := "//platformid.api.mondoo.app/runtime/k8s/uid/e26043bb-8669-48a2-b684-b1e132198cdc"

	expectedAssetPlatformIds := []string{
		clusterIdentifier + "/namespace/" + daemonsets[0].Namespace + "/daemonsets/name/" + daemonsets[0].Name,
	}

	pCfg := &providers.Config{}
	ownershipDir := k8s.NewEmptyPlatformIdOwnershipDirectory(clusterIdentifier)
	resFilter := map[string][]K8sResourceIdentifier{
		"daemonset": {
			{Type: "daemonset", Name: daemonsets[0].Name, Namespace: daemonsets[0].Namespace},
		},
	}
	assets, err := ListDaemonSets(p, pCfg, clusterIdentifier, nil, resFilter, ownershipDir)
	require.NoError(t, err)

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
	assert.Equal(t, "apps/v1", assets[0].Platform.Version)
	assert.Equal(t, "k8s-daemonset", assets[0].Platform.Name)
	assert.ElementsMatch(t, []string{"k8s", "k8s-workload"}, assets[0].Platform.Family)
	assert.Equal(t, daemonsets[0].Namespace, assets[0].Labels["k8s.mondoo.com/namespace"])
}
