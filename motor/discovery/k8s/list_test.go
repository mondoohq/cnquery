package k8s

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.io/mondoo/motor/transports/k8s/fake"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestListPodImage(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	transport := fake.NewMockTransport(mockCtrl)

	// Seed namespaces
	nss := []corev1.Namespace{
		{ObjectMeta: metav1.ObjectMeta{Name: "default"}},
		{ObjectMeta: metav1.ObjectMeta{Name: "kube-system"}},
	}
	transport.EXPECT().Namespaces().Return(nss, nil)

	// Seed pods
	pods1 := []corev1.Pod{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "nginx", Namespace: nss[0].Name},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{{Image: "nginx:1.22.0-alpine"}},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "nginx2", Namespace: nss[0].Name},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{{Image: "nginx:1.22.0-alpine"}},
			},
		},
	}

	pods2 := []corev1.Pod{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "kube-proxy", Namespace: nss[1].Name},
			Spec: corev1.PodSpec{
				InitContainers: []corev1.Container{{Image: "k8s.gcr.io/kube-proxy:v1.23.3"}},
				Containers:     []corev1.Container{{Image: "k8s.gcr.io/kube-proxy:v1.23.3"}},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "kube-scheduler", Namespace: nss[1].Name},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{{Image: "k8s.gcr.io/kube-scheduler:v1.23.3"}},
			},
		},
	}
	transport.EXPECT().Pods(nss[0]).Return(pods1, nil)
	transport.EXPECT().Pods(nss[1]).Return(pods2, nil)

	// nginx's tags seem to change digests, so resolve it to figure out what is the correct one
	ref, err := name.ParseReference("nginx:1.22.0-alpine", name.WeakValidation)
	require.NoError(t, err)
	desc, err := remote.Get(ref)
	require.NoError(t, err)

	imgDigest := desc.Digest.String()
	repoName := ref.Context().Name()
	imageUrl := repoName + "@" + imgDigest

	expectedAssetNames := []string{
		imageUrl,
		"k8s.gcr.io/kube-scheduler@sha256:32308abe86f7415611ca86ee79dd0a73e74ebecb2f9e3eb85fc3a8e62f03d0e7",
		"k8s.gcr.io/kube-proxy@sha256:def87f007b49d50693aed83d4703d0e56c69ae286154b1c7a20cd1b3a320cf7c",
	}

	assets, err := ListPodImages(transport, nil)
	assert.NoError(t, err)

	var assetNames []string
	for _, a := range assets {
		assetNames = append(assetNames, a.Name)
	}

	assert.ElementsMatch(t, expectedAssetNames, assetNames)
}

func TestListPodImage_FromStatus(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	transport := fake.NewMockTransport(mockCtrl)

	// Seed namespaces
	nss := []corev1.Namespace{
		{ObjectMeta: metav1.ObjectMeta{Name: "default"}},
		{ObjectMeta: metav1.ObjectMeta{Name: "kube-system"}},
	}
	transport.EXPECT().Namespaces().Return(nss, nil)

	// Seed pods
	pods1 := []corev1.Pod{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "nginx", Namespace: nss[0].Name},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{{Image: "nginx:1.22.0-alpine"}},
			},
			Status: corev1.PodStatus{
				ContainerStatuses: []corev1.ContainerStatus{
					{
						Image:   "nginx:1.22.0-alpine",
						ImageID: "docker-pullable://nginx@sha256:f335d7436887b39393409261603fb248e0c385ec18997d866dd44f7e9b621096",
					},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "nginx2", Namespace: nss[0].Name},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{{Image: "nginx:1.22.0-alpine"}},
			},
			Status: corev1.PodStatus{
				ContainerStatuses: []corev1.ContainerStatus{
					{
						Image:   "nginx:1.22.0-alpine",
						ImageID: "docker-pullable://nginx@sha256:f335d7436887b39393409261603fb248e0c385ec18997d866dd44f7e9b621096",
					},
				},
			},
		},
	}

	pods2 := []corev1.Pod{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "kube-proxy", Namespace: nss[1].Name},
			Spec: corev1.PodSpec{
				InitContainers: []corev1.Container{{Image: "k8s.gcr.io/kube-proxy:v1.23.3"}},
				Containers:     []corev1.Container{{Image: "k8s.gcr.io/kube-proxy:v1.23.3"}},
			},
			Status: corev1.PodStatus{
				InitContainerStatuses: []corev1.ContainerStatus{
					{
						Image:   "k8s.gcr.io/kube-proxy:v1.23.3",
						ImageID: "docker-pullable://k8s.gcr.io/kube-proxy@sha256:def87f007b49d50693aed83d4703d0e56c69ae286154b1c7a20cd1b3a320cf7c",
					},
				},
				ContainerStatuses: []corev1.ContainerStatus{
					{
						Image:   "k8s.gcr.io/kube-proxy:v1.23.3",
						ImageID: "docker-pullable://k8s.gcr.io/kube-proxy@sha256:def87f007b49d50693aed83d4703d0e56c69ae286154b1c7a20cd1b3a320cf7c",
					},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "kube-scheduler", Namespace: nss[1].Name},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{{Image: "k8s.gcr.io/kube-scheduler:v1.23.3"}},
			},
			Status: corev1.PodStatus{
				ContainerStatuses: []corev1.ContainerStatus{
					{
						Image:   "k8s.gcr.io/kube-scheduler:v1.23.3",
						ImageID: "docker-pullable://k8s.gcr.io/kube-scheduler@sha256:32308abe86f7415611ca86ee79dd0a73e74ebecb2f9e3eb85fc3a8e62f03d0e7",
					},
				},
			},
		},
	}
	transport.EXPECT().Pods(nss[0]).Return(pods1, nil)
	transport.EXPECT().Pods(nss[1]).Return(pods2, nil)

	expectedAssetNames := []string{
		"index.docker.io/library/nginx@sha256:f335d7436887b39393409261603fb248e0c385ec18997d866dd44f7e9b621096",
		"k8s.gcr.io/kube-scheduler@sha256:32308abe86f7415611ca86ee79dd0a73e74ebecb2f9e3eb85fc3a8e62f03d0e7",
		"k8s.gcr.io/kube-proxy@sha256:def87f007b49d50693aed83d4703d0e56c69ae286154b1c7a20cd1b3a320cf7c",
	}

	assets, err := ListPodImages(transport, nil)
	assert.NoError(t, err)

	var assetNames []string
	for _, a := range assets {
		assetNames = append(assetNames, a.Name)
	}

	assert.ElementsMatch(t, expectedAssetNames, assetNames)
}
