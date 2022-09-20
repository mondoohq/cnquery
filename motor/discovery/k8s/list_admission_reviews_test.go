package k8s

import (
	"bytes"
	"io"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/motor/providers"
	"go.mondoo.com/cnquery/motor/providers/k8s"

	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/client-go/kubernetes/scheme"
)

func TestListAdmissionReviews(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	p := k8s.NewMockKubernetesProvider(mockCtrl)
	// called for each AdmissionReview
	p.EXPECT().Runtime().Return("k8s-cluster")

	pod := corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "nginx",
			Namespace: "default",
			UID:       "123",
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{Image: "nginx:1.22.0-alpine"}},
		},
	}
	s := json.NewYAMLSerializer(json.DefaultMetaFactory, scheme.Scheme, scheme.Scheme)

	var b bytes.Buffer
	foo := io.Writer(&b)
	err := s.Encode(&pod, foo)
	require.NoError(t, err)
	// Seed AdmissionReviews
	admissionReviews := []admissionv1.AdmissionReview{
		{
			TypeMeta: metav1.TypeMeta{
				Kind:       "AdmissionReview",
				APIVersion: "admission.k8s.io/v1",
			},
			Request: &admissionv1.AdmissionRequest{
				Name:      "nginx",
				Namespace: "default",
				UID:       "123",
				Object:    runtime.RawExtension{Raw: b.Bytes()},
			},
		},
	}

	p.EXPECT().AdmissionReviews().Return(admissionReviews, nil)

	expectedAssetNames := []string{
		pod.Namespace + "/" + pod.Name,
	}

	clusterIdentifier := "//platformid.api.mondoo.app/runtime/k8s/uid/e26043bb-8669-48a2-b684-b1e132198cdc"

	expectedAssetPlatformIds := []string{
		clusterIdentifier + "/namespace/" + pod.Namespace + "/admissionreviews/name/" + pod.Name,
	}

	pCfg := &providers.Config{}
	ownershipDir := k8s.NewEmptyPlatformIdOwnershipDirectory(clusterIdentifier)
	assets, err := ListAdmissionReviews(p, pCfg, clusterIdentifier, ownershipDir)
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
	assert.Equal(t, "admission.k8s.io/v1", assets[0].Platform.Version)
	assert.Equal(t, "k8s-admission", assets[0].Platform.Name)
	assert.ElementsMatch(t, []string{"k8s", "k8s-workload"}, assets[0].Platform.Family)
	assert.Equal(t, pod.Namespace, assets[0].Labels["namespace"])
}
