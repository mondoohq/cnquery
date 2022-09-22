package k8s

import (
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/motor/providers"
	"go.mondoo.com/cnquery/motor/providers/k8s"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestListCronJobs(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	p := k8s.NewMockKubernetesProvider(mockCtrl)

	// Seed namespaces
	nss := []corev1.Namespace{
		{ObjectMeta: metav1.ObjectMeta{Name: "default"}},
	}
	p.EXPECT().Namespaces().Return(nss, nil)
	// called for each CronJob
	p.EXPECT().Runtime().Return("k8s-cluster")
	p.EXPECT().Runtime().Return("k8s-cluster")

	// pretend cronjob owned by deployment
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

	// Seed CronJobs
	cronjobs := []batchv1.CronJob{
		{
			TypeMeta: metav1.TypeMeta{
				Kind:       "CronJob",
				APIVersion: "batch/v1",
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
		},
		{
			TypeMeta: metav1.TypeMeta{
				Kind:       "CronJob",
				APIVersion: "batch/v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "nginx2",
				Namespace: nss[0].Name,
				UID:       "456",
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
		},
	}

	p.EXPECT().CronJobs(nss[0]).Return(cronjobs, nil)

	expectedAssetNames := []string{
		nss[0].Name + "/" + cronjobs[0].Name,
		nss[0].Name + "/" + cronjobs[1].Name,
	}

	clusterIdentifier := "//platformid.api.mondoo.app/runtime/k8s/uid/e26043bb-8669-48a2-b684-b1e132198cdc"

	expectedAssetPlatformIds := []string{
		clusterIdentifier + "/namespace/" + nss[0].Name + "/cronjobs/name/" + cronjobs[0].Name,
		clusterIdentifier + "/namespace/" + nss[0].Name + "/cronjobs/name/" + cronjobs[1].Name,
	}

	pCfg := &providers.Config{}
	ownershipDir := k8s.NewEmptyPlatformIdOwnershipDirectory(clusterIdentifier)
	assets, err := ListCronJobs(p, pCfg, clusterIdentifier, nil, make(map[string][]K8sResourceIdentifier), ownershipDir)
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
	assert.Equal(t, "batch/v1", assets[0].Platform.Version)
	assert.Equal(t, "k8s-cronjob", assets[0].Platform.Name)
	assert.ElementsMatch(t, []string{"k8s", "k8s-workload"}, assets[0].Platform.Family)
	assert.Equal(t, nss[0].Name, assets[0].Labels["k8s.mondoo.com/namespace"])
}

func TestListCronJobs_Filter(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	p := k8s.NewMockKubernetesProvider(mockCtrl)

	// called for each CronJob
	p.EXPECT().Runtime().Return("k8s-cluster")

	// Seed CronJobs
	cronjobs := []batchv1.CronJob{
		{
			TypeMeta: metav1.TypeMeta{
				Kind:       "CronJob",
				APIVersion: "batch/v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "nginx",
				Namespace: "default",
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
		},
		{
			TypeMeta: metav1.TypeMeta{
				Kind:       "CronJob",
				APIVersion: "batch/v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "nginx2",
				Namespace: "default",
				UID:       "456",
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
		},
	}

	p.EXPECT().CronJob(cronjobs[0].Namespace, cronjobs[0].Name).Return(&cronjobs[0], nil)

	expectedAssetNames := []string{
		cronjobs[0].Namespace + "/" + cronjobs[0].Name,
	}

	clusterIdentifier := "//platformid.api.mondoo.app/runtime/k8s/uid/e26043bb-8669-48a2-b684-b1e132198cdc"

	expectedAssetPlatformIds := []string{
		clusterIdentifier + "/namespace/" + cronjobs[0].Namespace + "/cronjobs/name/" + cronjobs[0].Name,
	}

	pCfg := &providers.Config{}
	ownershipDir := k8s.NewEmptyPlatformIdOwnershipDirectory(clusterIdentifier)
	resFilter := map[string][]K8sResourceIdentifier{
		"cronjob": {
			{Type: "cronjob", Name: cronjobs[0].Name, Namespace: cronjobs[0].Namespace},
		},
	}
	assets, err := ListCronJobs(p, pCfg, clusterIdentifier, nil, resFilter, ownershipDir)
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
	assert.Equal(t, "batch/v1", assets[0].Platform.Version)
	assert.Equal(t, "k8s-cronjob", assets[0].Platform.Name)
	assert.ElementsMatch(t, []string{"k8s", "k8s-workload"}, assets[0].Platform.Family)
	assert.Equal(t, cronjobs[0].Namespace, assets[0].Labels["k8s.mondoo.com/namespace"])
}
