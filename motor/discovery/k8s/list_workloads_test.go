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
	cronjobs := []*batchv1.CronJob{
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
	assets, err := ListCronJobs(p, pCfg, clusterIdentifier, NamespaceFilterOpts{}, make(map[string][]K8sResourceIdentifier), ownershipDir)
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
	cronjobs := []*batchv1.CronJob{
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

	p.EXPECT().CronJob(cronjobs[0].Namespace, cronjobs[0].Name).Return(cronjobs[0], nil)

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
	assets, err := ListCronJobs(p, pCfg, clusterIdentifier, NamespaceFilterOpts{}, resFilter, ownershipDir)
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
	daemonsets := []*appsv1.DaemonSet{
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
	assets, err := ListDaemonSets(p, pCfg, clusterIdentifier, NamespaceFilterOpts{}, make(map[string][]K8sResourceIdentifier), ownershipDir)
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
	daemonsets := []*appsv1.DaemonSet{
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

	p.EXPECT().DaemonSet(daemonsets[0].Namespace, daemonsets[0].Name).Return(daemonsets[0], nil)

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
	assets, err := ListDaemonSets(p, pCfg, clusterIdentifier, NamespaceFilterOpts{}, resFilter, ownershipDir)
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

func TestListDeployments(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	p := k8s.NewMockKubernetesProvider(mockCtrl)

	// Seed namespaces
	nss := []corev1.Namespace{
		{ObjectMeta: metav1.ObjectMeta{Name: "default"}},
	}
	p.EXPECT().Namespaces().Return(nss, nil)
	// called for each Deployment
	p.EXPECT().Runtime().Return("k8s-cluster")
	p.EXPECT().Runtime().Return("k8s-cluster")

	// pretend the deployment is owned by something
	parent := appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "nginx-deployment-deployment",
			Namespace: nss[0].Name,
			UID:       "000",
		},
	}

	// Seed Deployments
	deployments := []*appsv1.Deployment{
		{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Deployment",
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
			Spec: appsv1.DeploymentSpec{
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{{Image: "nginx:1.22.0-alpine"}},
					},
				},
			},
		},
		{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Deployment",
				APIVersion: "apps/v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "nginx2",
				Namespace: nss[0].Name,
				UID:       "456",
			},
			Spec: appsv1.DeploymentSpec{
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{{Image: "nginx:1.22.0-alpine"}},
					},
				},
			},
		},
	}

	p.EXPECT().Deployments(nss[0]).Return(deployments, nil)

	expectedAssetNames := []string{
		nss[0].Name + "/" + deployments[0].Name,
		nss[0].Name + "/" + deployments[1].Name,
	}

	clusterIdentifier := "//platformid.api.mondoo.app/runtime/k8s/uid/e26043bb-8669-48a2-b684-b1e132198cdc"

	expectedAssetPlatformIds := []string{
		clusterIdentifier + "/namespace/" + nss[0].Name + "/deployments/name/" + deployments[0].Name,
		clusterIdentifier + "/namespace/" + nss[0].Name + "/deployments/name/" + deployments[1].Name,
	}

	pCfg := &providers.Config{}
	ownershipDir := k8s.NewEmptyPlatformIdOwnershipDirectory(clusterIdentifier)
	assets, err := ListDeployments(p, pCfg, clusterIdentifier, NamespaceFilterOpts{}, make(map[string][]K8sResourceIdentifier), ownershipDir)
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
	assert.Equal(t, "k8s-deployment", assets[0].Platform.Name)
	assert.ElementsMatch(t, []string{"k8s", "k8s-workload"}, assets[0].Platform.Family)
	assert.Equal(t, nss[0].Name, assets[0].Labels["k8s.mondoo.com/namespace"])
}

func TestListDeployments_Filter(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	p := k8s.NewMockKubernetesProvider(mockCtrl)

	// called for each Deployment
	p.EXPECT().Runtime().Return("k8s-cluster")

	// Seed Deployments
	deployments := []*appsv1.Deployment{
		{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Deployment",
				APIVersion: "apps/v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "nginx",
				Namespace: "default",
				UID:       "123",
			},
			Spec: appsv1.DeploymentSpec{
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{{Image: "nginx:1.22.0-alpine"}},
					},
				},
			},
		},
		{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Deployment",
				APIVersion: "apps/v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "nginx2",
				Namespace: "default",
				UID:       "456",
			},
			Spec: appsv1.DeploymentSpec{
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{{Image: "nginx:1.22.0-alpine"}},
					},
				},
			},
		},
	}

	p.EXPECT().Deployment(deployments[0].Namespace, deployments[0].Name).Return(deployments[0], nil)

	expectedAssetNames := []string{
		deployments[0].Namespace + "/" + deployments[0].Name,
	}

	clusterIdentifier := "//platformid.api.mondoo.app/runtime/k8s/uid/e26043bb-8669-48a2-b684-b1e132198cdc"

	expectedAssetPlatformIds := []string{
		clusterIdentifier + "/namespace/" + deployments[0].Namespace + "/deployments/name/" + deployments[0].Name,
	}

	pCfg := &providers.Config{}
	ownershipDir := k8s.NewEmptyPlatformIdOwnershipDirectory(clusterIdentifier)
	resFilter := map[string][]K8sResourceIdentifier{
		"deployment": {
			{Type: "deployment", Name: deployments[0].Name, Namespace: deployments[0].Namespace},
		},
	}
	assets, err := ListDeployments(p, pCfg, clusterIdentifier, NamespaceFilterOpts{}, resFilter, ownershipDir)
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
	assert.Equal(t, "k8s-deployment", assets[0].Platform.Name)
	assert.ElementsMatch(t, []string{"k8s", "k8s-workload"}, assets[0].Platform.Family)
	assert.Equal(t, deployments[0].Namespace, assets[0].Labels["k8s.mondoo.com/namespace"])
}

func TestListJobs(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	p := k8s.NewMockKubernetesProvider(mockCtrl)

	// Seed namespaces
	nss := []corev1.Namespace{
		{ObjectMeta: metav1.ObjectMeta{Name: "default"}},
	}
	p.EXPECT().Namespaces().Return(nss, nil)
	// called for each Job
	p.EXPECT().Runtime().Return("k8s-cluster")
	p.EXPECT().Runtime().Return("k8s-cluster")

	// pretend the job has a parent
	parent := appsv1.ReplicaSet{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ReplicaSet",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "nginx-replicaset",
			Namespace: nss[0].Name,
			UID:       "000",
		},
	}

	// Seed Jobs
	jobs := []*batchv1.Job{
		{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Job",
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
		{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Job",
				APIVersion: "batch/v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "nginx2",
				Namespace: nss[0].Name,
				UID:       "456",
			},
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
	}

	p.EXPECT().Jobs(nss[0]).Return(jobs, nil)

	expectedAssetNames := []string{
		nss[0].Name + "/" + jobs[0].Name,
		nss[0].Name + "/" + jobs[1].Name,
	}

	clusterIdentifier := "//platformid.api.mondoo.app/runtime/k8s/uid/e26043bb-8669-48a2-b684-b1e132198cdc"

	expectedAssetPlatformIds := []string{
		clusterIdentifier + "/namespace/" + nss[0].Name + "/jobs/name/" + jobs[0].Name,
		clusterIdentifier + "/namespace/" + nss[0].Name + "/jobs/name/" + jobs[1].Name,
	}

	pCfg := &providers.Config{}
	ownershipDir := k8s.NewEmptyPlatformIdOwnershipDirectory(clusterIdentifier)
	ownershipDir.Add(&parent)
	assets, err := ListJobs(p, pCfg, clusterIdentifier, NamespaceFilterOpts{}, make(map[string][]K8sResourceIdentifier), ownershipDir)
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
	assert.Equal(t, "k8s-job", assets[0].Platform.Name)
	assert.ElementsMatch(t, []string{"k8s", "k8s-workload"}, assets[0].Platform.Family)
	assert.Equal(t, nss[0].Name, assets[0].Labels["k8s.mondoo.com/namespace"])
}

func TestListJobs_Filter(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	p := k8s.NewMockKubernetesProvider(mockCtrl)

	// called for each Job
	p.EXPECT().Runtime().Return("k8s-cluster")

	// Seed Jobs
	jobs := []*batchv1.Job{
		{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Job",
				APIVersion: "batch/v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "nginx",
				Namespace: "default",
				UID:       "123",
			},
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
		{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Job",
				APIVersion: "batch/v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "nginx2",
				Namespace: "default",
				UID:       "456",
			},
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
	}

	p.EXPECT().Job(jobs[0].Namespace, jobs[0].Name).Return(jobs[0], nil)

	expectedAssetNames := []string{
		jobs[0].Namespace + "/" + jobs[0].Name,
	}

	clusterIdentifier := "//platformid.api.mondoo.app/runtime/k8s/uid/e26043bb-8669-48a2-b684-b1e132198cdc"

	expectedAssetPlatformIds := []string{
		clusterIdentifier + "/namespace/" + jobs[0].Namespace + "/jobs/name/" + jobs[0].Name,
	}

	pCfg := &providers.Config{}
	ownershipDir := k8s.NewEmptyPlatformIdOwnershipDirectory(clusterIdentifier)
	resFilter := map[string][]K8sResourceIdentifier{
		"job": {
			{Type: "job", Name: jobs[0].Name, Namespace: jobs[0].Namespace},
		},
	}
	assets, err := ListJobs(p, pCfg, clusterIdentifier, NamespaceFilterOpts{}, resFilter, ownershipDir)
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
	assert.Equal(t, "k8s-job", assets[0].Platform.Name)
	assert.ElementsMatch(t, []string{"k8s", "k8s-workload"}, assets[0].Platform.Family)
	assert.Equal(t, jobs[0].Namespace, assets[0].Labels["k8s.mondoo.com/namespace"])
}

func TestListPods(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	p := k8s.NewMockKubernetesProvider(mockCtrl)

	// Seed namespaces
	nss := []corev1.Namespace{
		{ObjectMeta: metav1.ObjectMeta{Name: "default"}},
	}
	p.EXPECT().Namespaces().Return(nss, nil)
	// called for each Pod
	p.EXPECT().Runtime().Return("k8s-cluster")
	p.EXPECT().Runtime().Return("k8s-cluster")

	parent := appsv1.ReplicaSet{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ReplicaSet",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "nginx-replicaset",
			Namespace: nss[0].Name,
			UID:       "000",
		},
	}

	// Seed Pods
	pods := []*corev1.Pod{
		{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Pod",
				APIVersion: "v1",
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
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{{Image: "nginx:1.22.0-alpine"}},
			},
		},
		{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Pod",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "nginx2",
				Namespace: nss[0].Name,
				UID:       "456",
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{{Image: "nginx:1.22.0-alpine"}},
			},
		},
	}

	p.EXPECT().Pods(nss[0]).Return(pods, nil)

	expectedAssetNames := []string{
		nss[0].Name + "/" + pods[0].Name,
		nss[0].Name + "/" + pods[1].Name,
	}

	clusterIdentifier := "//platformid.api.mondoo.app/runtime/k8s/uid/e26043bb-8669-48a2-b684-b1e132198cdc"

	expectedAssetPlatformIds := []string{
		clusterIdentifier + "/namespace/" + nss[0].Name + "/pods/name/" + pods[0].Name,
		clusterIdentifier + "/namespace/" + nss[0].Name + "/pods/name/" + pods[1].Name,
	}

	pCfg := &providers.Config{}
	ownershipDir := k8s.NewEmptyPlatformIdOwnershipDirectory(clusterIdentifier)
	assets, err := ListPods(p, pCfg, clusterIdentifier, NamespaceFilterOpts{}, make(map[string][]K8sResourceIdentifier), ownershipDir)
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
	assert.Equal(t, "v1", assets[0].Platform.Version)
	assert.Equal(t, "k8s-pod", assets[0].Platform.Name)
	assert.ElementsMatch(t, []string{"k8s", "k8s-workload"}, assets[0].Platform.Family)
	assert.Equal(t, nss[0].Name, assets[0].Labels["k8s.mondoo.com/namespace"])
}

func TestListPods_Filter(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	p := k8s.NewMockKubernetesProvider(mockCtrl)

	// called for each Pod
	p.EXPECT().Runtime().Return("k8s-cluster")

	// Seed Pods
	pods := []*corev1.Pod{
		{
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
		},
		{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Pod",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "nginx2",
				Namespace: "default",
				UID:       "456",
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{{Image: "nginx:1.22.0-alpine"}},
			},
		},
	}

	p.EXPECT().Pod(pods[0].Namespace, pods[0].Name).Return(pods[0], nil)

	expectedAssetNames := []string{
		pods[0].Namespace + "/" + pods[0].Name,
	}

	clusterIdentifier := "//platformid.api.mondoo.app/runtime/k8s/uid/e26043bb-8669-48a2-b684-b1e132198cdc"

	expectedAssetPlatformIds := []string{
		clusterIdentifier + "/namespace/" + pods[0].Namespace + "/pods/name/" + pods[0].Name,
	}

	pCfg := &providers.Config{}
	ownershipDir := k8s.NewEmptyPlatformIdOwnershipDirectory(clusterIdentifier)
	resFilter := map[string][]K8sResourceIdentifier{
		"pod": {
			{Type: "pod", Name: pods[0].Name, Namespace: pods[0].Namespace},
		},
	}
	assets, err := ListPods(p, pCfg, clusterIdentifier, NamespaceFilterOpts{}, resFilter, ownershipDir)
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
	assert.Equal(t, "v1", assets[0].Platform.Version)
	assert.Equal(t, "k8s-pod", assets[0].Platform.Name)
	assert.ElementsMatch(t, []string{"k8s", "k8s-workload"}, assets[0].Platform.Family)
	assert.Equal(t, pods[0].Namespace, assets[0].Labels["k8s.mondoo.com/namespace"])
}

func TestListReplicaSets(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	p := k8s.NewMockKubernetesProvider(mockCtrl)

	// Seed namespaces
	nss := []corev1.Namespace{
		{ObjectMeta: metav1.ObjectMeta{Name: "default"}},
	}
	p.EXPECT().Namespaces().Return(nss, nil)
	// called for each ReplicaSet
	p.EXPECT().Runtime().Return("k8s-cluster")
	p.EXPECT().Runtime().Return("k8s-cluster")
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
	// Seed ReplicaSets
	replicaSets := []*appsv1.ReplicaSet{
		{
			TypeMeta: metav1.TypeMeta{
				Kind:       "ReplicaSet",
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
			Spec: appsv1.ReplicaSetSpec{
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{{Image: "nginx:1.22.0-alpine"}},
					},
				},
			},
		},
		{
			TypeMeta: metav1.TypeMeta{
				Kind:       "ReplicaSet",
				APIVersion: "apps/v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "nginx2",
				Namespace: nss[0].Name,
				UID:       "456",
			},
			Spec: appsv1.ReplicaSetSpec{
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{{Image: "nginx:1.22.0-alpine"}},
					},
				},
			},
		},
	}

	p.EXPECT().ReplicaSets(nss[0]).Return(replicaSets, nil)

	expectedAssetNames := []string{
		nss[0].Name + "/" + replicaSets[0].Name,
		nss[0].Name + "/" + replicaSets[1].Name,
	}

	clusterIdentifier := "//platformid.api.mondoo.app/runtime/k8s/uid/e26043bb-8669-48a2-b684-b1e132198cdc"

	expectedAssetPlatformIds := []string{
		clusterIdentifier + "/namespace/" + nss[0].Name + "/replicasets/name/" + replicaSets[0].Name,
		clusterIdentifier + "/namespace/" + nss[0].Name + "/replicasets/name/" + replicaSets[1].Name,
	}

	pCfg := &providers.Config{}
	ownershipDir := k8s.NewEmptyPlatformIdOwnershipDirectory(clusterIdentifier)
	assets, err := ListReplicaSets(p, pCfg, clusterIdentifier, NamespaceFilterOpts{}, make(map[string][]K8sResourceIdentifier), ownershipDir)
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
	assert.Equal(t, "k8s-replicaset", assets[0].Platform.Name)
	assert.ElementsMatch(t, []string{"k8s", "k8s-workload"}, assets[0].Platform.Family)
	assert.Equal(t, nss[0].Name, assets[0].Labels["k8s.mondoo.com/namespace"])
}

func TestListReplicaSets_Filter(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	p := k8s.NewMockKubernetesProvider(mockCtrl)

	// called for each ReplicaSet
	p.EXPECT().Runtime().Return("k8s-cluster")

	// Seed ReplicaSets
	replicaSets := []*appsv1.ReplicaSet{
		{
			TypeMeta: metav1.TypeMeta{
				Kind:       "ReplicaSet",
				APIVersion: "apps/v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "nginx",
				Namespace: "default",
				UID:       "123",
			},
			Spec: appsv1.ReplicaSetSpec{
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{{Image: "nginx:1.22.0-alpine"}},
					},
				},
			},
		},
		{
			TypeMeta: metav1.TypeMeta{
				Kind:       "ReplicaSet",
				APIVersion: "apps/v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "nginx2",
				Namespace: "default",
				UID:       "456",
			},
			Spec: appsv1.ReplicaSetSpec{
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{{Image: "nginx:1.22.0-alpine"}},
					},
				},
			},
		},
	}

	p.EXPECT().ReplicaSet(replicaSets[0].Namespace, replicaSets[0].Name).Return(replicaSets[0], nil)

	expectedAssetNames := []string{
		replicaSets[0].Namespace + "/" + replicaSets[0].Name,
	}

	clusterIdentifier := "//platformid.api.mondoo.app/runtime/k8s/uid/e26043bb-8669-48a2-b684-b1e132198cdc"

	expectedAssetPlatformIds := []string{
		clusterIdentifier + "/namespace/" + replicaSets[0].Namespace + "/replicasets/name/" + replicaSets[0].Name,
	}

	pCfg := &providers.Config{}
	ownershipDir := k8s.NewEmptyPlatformIdOwnershipDirectory(clusterIdentifier)
	resFilter := map[string][]K8sResourceIdentifier{
		"replicaset": {
			{Type: "replicaset", Name: replicaSets[0].Name, Namespace: replicaSets[0].Namespace},
		},
	}
	assets, err := ListReplicaSets(p, pCfg, clusterIdentifier, NamespaceFilterOpts{}, resFilter, ownershipDir)
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
	assert.Equal(t, "k8s-replicaset", assets[0].Platform.Name)
	assert.ElementsMatch(t, []string{"k8s", "k8s-workload"}, assets[0].Platform.Family)
	assert.Equal(t, replicaSets[0].Namespace, assets[0].Labels["k8s.mondoo.com/namespace"])
}

func TestListStatefulSets(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	p := k8s.NewMockKubernetesProvider(mockCtrl)

	// Seed namespaces
	nss := []corev1.Namespace{
		{ObjectMeta: metav1.ObjectMeta{Name: "default"}},
	}
	p.EXPECT().Namespaces().Return(nss, nil)
	// called for each StatefulSet
	p.EXPECT().Runtime().Return("k8s-cluster")
	p.EXPECT().Runtime().Return("k8s-cluster")

	// pretend stateful set owned by deployment
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

	// Seed StatefulSets
	statefulsets := []*appsv1.StatefulSet{
		{
			TypeMeta: metav1.TypeMeta{
				Kind:       "StatefulSet",
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
			TypeMeta: metav1.TypeMeta{
				Kind:       "StatefulSet",
				APIVersion: "apps/v1",
			},
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

	p.EXPECT().StatefulSets(nss[0]).Return(statefulsets, nil)

	expectedAssetNames := []string{
		nss[0].Name + "/" + statefulsets[0].Name,
		nss[0].Name + "/" + statefulsets[1].Name,
	}

	clusterIdentifier := "//platformid.api.mondoo.app/runtime/k8s/uid/e26043bb-8669-48a2-b684-b1e132198cdc"

	expectedAssetPlatformIds := []string{
		clusterIdentifier + "/namespace/" + nss[0].Name + "/statefulsets/name/" + statefulsets[0].Name,
		clusterIdentifier + "/namespace/" + nss[0].Name + "/statefulsets/name/" + statefulsets[1].Name,
	}

	pCfg := &providers.Config{}
	ownershipDir := k8s.NewEmptyPlatformIdOwnershipDirectory(clusterIdentifier)
	assets, err := ListStatefulSets(p, pCfg, clusterIdentifier, NamespaceFilterOpts{}, make(map[string][]K8sResourceIdentifier), ownershipDir)
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
	assert.Equal(t, "k8s-statefulset", assets[0].Platform.Name)
	assert.ElementsMatch(t, []string{"k8s", "k8s-workload"}, assets[0].Platform.Family)
	assert.Equal(t, nss[0].Name, assets[0].Labels["k8s.mondoo.com/namespace"])
}

func TestListStatefulSets_Filter(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	p := k8s.NewMockKubernetesProvider(mockCtrl)

	// called for each StatefulSet
	p.EXPECT().Runtime().Return("k8s-cluster")

	// Seed StatefulSets
	statefulsets := []*appsv1.StatefulSet{
		{
			TypeMeta: metav1.TypeMeta{
				Kind:       "StatefulSet",
				APIVersion: "apps/v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "nginx",
				Namespace: "default",
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
			TypeMeta: metav1.TypeMeta{
				Kind:       "StatefulSet",
				APIVersion: "apps/v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "nginx2",
				Namespace: "default",
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

	p.EXPECT().StatefulSet(statefulsets[0].Namespace, statefulsets[0].Name).Return(statefulsets[0], nil)

	expectedAssetNames := []string{
		statefulsets[0].Namespace + "/" + statefulsets[0].Name,
	}

	clusterIdentifier := "//platformid.api.mondoo.app/runtime/k8s/uid/e26043bb-8669-48a2-b684-b1e132198cdc"

	expectedAssetPlatformIds := []string{
		clusterIdentifier + "/namespace/" + statefulsets[0].Namespace + "/statefulsets/name/" + statefulsets[0].Name,
	}

	pCfg := &providers.Config{}
	ownershipDir := k8s.NewEmptyPlatformIdOwnershipDirectory(clusterIdentifier)
	resFilter := map[string][]K8sResourceIdentifier{
		"statefulset": {
			{Type: "statefulset", Name: statefulsets[0].Name, Namespace: statefulsets[0].Namespace},
		},
	}
	assets, err := ListStatefulSets(p, pCfg, clusterIdentifier, NamespaceFilterOpts{}, resFilter, ownershipDir)
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
	assert.Equal(t, "k8s-statefulset", assets[0].Platform.Name)
	assert.ElementsMatch(t, []string{"k8s", "k8s-workload"}, assets[0].Platform.Family)
	assert.Equal(t, statefulsets[0].Namespace, assets[0].Labels["k8s.mondoo.com/namespace"])
}

func TestListFiltering(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	p := k8s.NewMockKubernetesProvider(mockCtrl)

	// Seed namespaces
	nss := []corev1.Namespace{
		{ObjectMeta: metav1.ObjectMeta{Name: "default"}},
		{ObjectMeta: metav1.ObjectMeta{Name: "kube-system"}},
		{ObjectMeta: metav1.ObjectMeta{Name: "kube-system-alternative"}},
	}
	p.EXPECT().Namespaces().Return(nss, nil).AnyTimes()

	// Seed pods
	defaultNamespacePods := []*corev1.Pod{
		{
			TypeMeta:   metav1.TypeMeta{Kind: "Pod", APIVersion: "v1"},
			ObjectMeta: metav1.ObjectMeta{Name: "nginx", Namespace: nss[0].Name},
		},
		{
			TypeMeta:   metav1.TypeMeta{Kind: "Pod", APIVersion: "v1"},
			ObjectMeta: metav1.ObjectMeta{Name: "nginx2", Namespace: nss[0].Name},
		},
	}

	kubeSystemPods := []*corev1.Pod{
		{
			TypeMeta:   metav1.TypeMeta{Kind: "Pod", APIVersion: "v1"},
			ObjectMeta: metav1.ObjectMeta{Name: "kube-proxy", Namespace: nss[1].Name},
		},
	}

	otherNamespacePods := []*corev1.Pod{
		{
			TypeMeta:   metav1.TypeMeta{Kind: "Pod", APIVersion: "v1"},
			ObjectMeta: metav1.ObjectMeta{Name: "some-workload", Namespace: nss[2].Name},
		},
		{
			TypeMeta:   metav1.TypeMeta{Kind: "Pod", APIVersion: "v1"},
			ObjectMeta: metav1.ObjectMeta{Name: "some-workload2", Namespace: nss[2].Name},
		},
		{
			TypeMeta:   metav1.TypeMeta{Kind: "Pod", APIVersion: "v1"},
			ObjectMeta: metav1.ObjectMeta{Name: "some-workload3", Namespace: nss[2].Name},
		},
	}
	p.EXPECT().Pods(nss[0]).Return(defaultNamespacePods, nil).AnyTimes()
	p.EXPECT().Pods(nss[1]).Return(kubeSystemPods, nil).AnyTimes()
	p.EXPECT().Pods(nss[2]).Return(otherNamespacePods, nil).AnyTimes()
	p.EXPECT().Runtime().Return("k8s-cluster").AnyTimes()

	clusterIdentifier := "//platformid.api.mondoo.app/runtime/k8s/uid/e26043bb-8669-48a2-b684-b1e132198cdc"
	ownershipDir := k8s.NewEmptyPlatformIdOwnershipDirectory(clusterIdentifier)
	pCfg := &providers.Config{}

	// List with no filtering
	assets, err := ListPods(p, pCfg, clusterIdentifier, NamespaceFilterOpts{}, make(map[string][]K8sResourceIdentifier), ownershipDir)
	require.NoError(t, err)
	assert.Equal(t, 6, len(assets), "expected all Pods to be found when no filter specified")

	// List only 'kube-system'
	assets, err = ListPods(p, pCfg, clusterIdentifier, NamespaceFilterOpts{include: []string{nss[1].Name}}, make(map[string][]K8sResourceIdentifier), ownershipDir)
	require.NoError(t, err)
	assert.Equal(t, 1, len(assets), "expected only 1 Pod to be returned")

	// List 'kube-system' and 'other-namespace'
	assets, err = ListPods(p, pCfg, clusterIdentifier, NamespaceFilterOpts{include: []string{nss[1].Name, nss[2].Name}}, make(map[string][]K8sResourceIdentifier), ownershipDir)
	require.NoError(t, err)
	assert.Equal(t, 4, len(assets), "expected only 4 Pods to be returned")

	// Exclude kube-system
	assets, err = ListPods(p, pCfg, clusterIdentifier, NamespaceFilterOpts{exclude: []string{nss[1].Name}}, make(map[string][]K8sResourceIdentifier), ownershipDir)
	require.NoError(t, err)
	assert.Equal(t, 5, len(assets), "expected only 5 Pods to be returned")

	// Include and exclude list should behave like only include list
	assets, err = ListPods(p, pCfg, clusterIdentifier, NamespaceFilterOpts{include: []string{nss[1].Name}, exclude: []string{nss[1].Name}}, make(map[string][]K8sResourceIdentifier), ownershipDir)
	require.NoError(t, err)
	assert.Equal(t, 1, len(assets), "expected only 1 Pod to be returned")

	// List w/glob 'kube*'
	assets, err = ListPods(p, pCfg, clusterIdentifier, NamespaceFilterOpts{include: []string{"kube*"}}, make(map[string][]K8sResourceIdentifier), ownershipDir)
	require.NoError(t, err)
	assert.Equal(t, 4, len(assets), "expected 4 Pods to be returned from matched Namspaces")

	// List w/glob '*alt*'
	assets, err = ListPods(p, pCfg, clusterIdentifier, NamespaceFilterOpts{include: []string{"*alt*"}}, make(map[string][]K8sResourceIdentifier), ownershipDir)
	require.NoError(t, err)
	assert.Equal(t, 3, len(assets), "expected 3 Pods to be returned from matched Namspaces")

	// Exclude w/glob '*default*'
	assets, err = ListPods(p, pCfg, clusterIdentifier, NamespaceFilterOpts{exclude: []string{"*default*"}}, make(map[string][]K8sResourceIdentifier), ownershipDir)
	require.NoError(t, err)
	assert.Equal(t, 4, len(assets), "expected 4 Pods to be returned from non-excluded Namspaces")
}
