package k8s

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/motor/providers"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAssetData(t *testing.T) {
	// Seed CronJobs
	cronjob := batchv1.CronJob{
		TypeMeta: metav1.TypeMeta{
			Kind:       "CronJob",
			APIVersion: "batch/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "nginx",
			Namespace: "test123",
			UID:       "123",
		},
		Spec: batchv1.CronJobSpec{
			Schedule: "*/1 * * * *",
			JobTemplate: batchv1.JobTemplateSpec{
				Spec: batchv1.JobSpec{
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
		},
	}

	clusterIdentifier := "//platformid.api.mondoo.app/runtime/k8s/uid/e26043bb-8669-48a2-b684-b1e132198cdc"

	tc := &providers.Config{}

	asset, err := createAssetFromObject(&cronjob, "k8s-cluster", tc, clusterIdentifier)
	require.NoError(t, err)

	assert.Equal(t, "batch/v1", asset.Platform.Version)
	assert.Equal(t, "k8s-cronjob", asset.Platform.Name)
	assert.ElementsMatch(t, []string{"k8s", "k8s-workload"}, asset.Platform.Family)
	assert.Equal(t, "test123", asset.Labels["namespace"])
}

func TestAssetNodeData(t *testing.T) {
	// Seed CronJobs
	node := corev1.Node{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Node",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "minikube",
			UID:  "123",
		},
		Spec: corev1.NodeSpec{},
	}

	clusterIdentifier := "//platformid.api.mondoo.app/runtime/k8s/uid/e26043bb-8669-48a2-b684-b1e132198cdc"

	tc := &providers.Config{}

	asset, err := createAssetFromObject(&node, "k8s-cluster", tc, clusterIdentifier)
	require.NoError(t, err)

	assert.Equal(t, "v1", asset.Platform.Version)
	assert.Equal(t, "k8s-node", asset.Platform.Name)
	assert.ElementsMatch(t, []string{"k8s"}, asset.Platform.Family)
}
