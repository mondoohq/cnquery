package k8s

import (
	"context"
	"os"
	"path/filepath"

	"github.com/gosimple/slug"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/motor/platform"
	"go.mondoo.io/mondoo/motor/providers"
	"go.mondoo.io/mondoo/motor/providers/k8s/resources"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

func newApiProvider(namespace string, selectedResourceID string, dCache *resources.DiscoveryCache) (KubernetesProvider, error) {
	// check if the user .kube/config file exists
	// NOTE: BuildConfigFromFlags falls back to cluster loading when .kube/config string is empty
	// therefore we want to only change the kubeconfig string when the file really exists
	var kubeconfig string

	// use KUBECONFIG as default
	// https://kubernetes.io/docs/tasks/access-application-cluster/configure-access-multiple-clusters/#set-the-kubeconfig-environment-variable
	kubeconfig = os.Getenv("KUBECONFIG")

	// if no config is set, try to load the default kubeconfig path if nothing was provided
	if kubeconfig == "" {
		if home := homedir.HomeDir(); home != "" {
			kubeconfigpath := filepath.Join(home, ".kube", "config")
			if _, err := os.Stat(kubeconfigpath); err == nil {
				kubeconfig = kubeconfigpath
			}
		}
	}

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, err
	}

	// enable-client side throttling
	// avoids the cli warning: Waited for 1.000907542s due to client-side throttling, not priority and fairness
	config.QPS = 1000
	config.Burst = 1000

	// initialize api
	d, err := dCache.Get(config)
	if err != nil {
		return nil, err
	}
	log.Debug().Msg("loaded kubeconfig successfully")

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, errors.Wrap(err, "could not create kubernetes clientset")
	}

	return &apiProvider{
		namespace:          namespace,
		config:             config,
		d:                  d,
		clientset:          clientset,
		selectedResourceID: selectedResourceID,
	}, nil
}

type apiProvider struct {
	d                  *resources.Discovery
	config             *rest.Config
	namespace          string
	clientset          *kubernetes.Clientset
	selectedResourceID string
}

func (t *apiProvider) Close() {}

func (t *apiProvider) Capabilities() providers.Capabilities {
	return providers.Capabilities{}
}

func (t *apiProvider) Kind() providers.Kind {
	return providers.Kind_KIND_API
}

func (t *apiProvider) Runtime() string {
	return providers.RUNTIME_KUBERNETES_CLUSTER
}

func (t *apiProvider) PlatformIdDetectors() []providers.PlatformIdDetector {
	return []providers.PlatformIdDetector{
		providers.TransportPlatformIdentifierDetector,
	}
}

func (t *apiProvider) ID() (string, error) {
	// we use "kube-system" namespace uid as identifier for the cluster
	result, err := t.Resources("namespaces", "kube-system", "")
	if err != nil {
		return "", err
	}

	if len(result.Resources) != 1 {
		return "", errors.New("could not identify the k8s cluster")
	}

	resource := result.Resources[0]
	obj, err := meta.Accessor(resource)
	if err != nil {
		return "", err
	}

	uid := string(obj.GetUID())
	return uid, nil
}

func (t *apiProvider) PlatformIdentifier() (string, error) {
	if t.selectedResourceID != "" {
		return t.selectedResourceID, nil
	}

	uid, err := t.ID()
	if err != nil {
		return "", err
	}

	id := NewPlatformID(uid)
	if t.namespace != "" {
		id += "/namespace/" + slug.Make(t.namespace)
	}

	return id, nil
}

func (t *apiProvider) Identifier() (string, error) {
	return t.PlatformIdentifier()
}

func (t *apiProvider) Name() (string, error) {
	ci, err := t.ClusterInfo()
	if err != nil {
		return "", err
	}
	return ci.Name, nil
}

func (t *apiProvider) ClusterInfo() (ClusterInfo, error) {
	ctx := context.Background()
	res := ClusterInfo{}

	// right now we use the name of the first node to identify the cluster
	result, err := t.clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return res, err
	}

	if len(result.Items) > 0 {
		node := result.Items[0]
		res.Name = node.GetName()
	}

	return res, nil
}

func (t *apiProvider) ServerVersion() *version.Info {
	return t.d.ServerVersion
}

func (t *apiProvider) SupportedResourceTypes() (*resources.ApiResourceIndex, error) {
	return t.d.SupportedResourceTypes()
}

func (t *apiProvider) Resources(kind string, name string, namespace string) (*ResourceResult, error) {
	ctx := context.Background()
	ns := t.namespace
	allNs := false
	if len(ns) == 0 {
		allNs = true
	}

	// discover api and resources that have a list method
	resTypes, err := t.d.SupportedResourceTypes()
	if err != nil {
		return nil, err
	}
	log.Debug().Msg("completed querying resource types")

	resType, err := resTypes.Lookup(kind)
	if err != nil {
		return nil, err
	}

	log.Debug().Msgf("fetch all %s resources", kind)
	objs, err := t.d.GetKindResources(ctx, *resType, ns, allNs)
	if err != nil {
		return nil, err
	}
	log.Debug().Msgf("found %d resource objects", len(objs))

	objs, err = resources.FilterResource(resType, objs, name, namespace)
	if err != nil {
		return nil, err
	}

	return &ResourceResult{
		Name:         name,
		Kind:         kind,
		ResourceType: resType,
		Resources:    objs,
		Namespace:    ns,
		AllNs:        allNs,
	}, err
}

func (t *apiProvider) PlatformInfo() *platform.Platform {
	release := ""
	build := ""
	arch := ""

	platformData := getPlatformInfo(t.selectedResourceID, t.Runtime())
	if platformData != nil {
		return platformData
	}

	// cluster
	sv := t.ServerVersion()
	if sv != nil {
		release = sv.GitVersion
		build = sv.BuildDate
		arch = sv.Platform
	}

	return &platform.Platform{
		Name:    "kubernetes",
		Title:   "Kubernetes",
		Release: release,
		Version: release,
		Build:   build,
		Arch:    arch,
		Family:  []string{"kubernetes"},
		Kind:    providers.Kind_KIND_API,
		Runtime: providers.RUNTIME_KUBERNETES_CLUSTER,
	}
}

func (t *apiProvider) Namespaces() ([]v1.Namespace, error) {
	ctx := context.Background()
	list, err := t.clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	// needed because of https://github.com/kubernetes/client-go/issues/861
	for i := range list.Items {
		list.Items[i].SetGroupVersionKind(v1.SchemeGroupVersion.WithKind("Namespace"))
	}
	return list.Items, err
}

func (t *apiProvider) Pods(namespace v1.Namespace) ([]v1.Pod, error) {
	ctx := context.Background()
	list, err := t.clientset.CoreV1().Pods(namespace.Name).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	// needed because of https://github.com/kubernetes/client-go/issues/861
	for i := range list.Items {
		list.Items[i].SetGroupVersionKind(v1.SchemeGroupVersion.WithKind("Pod"))
	}
	return list.Items, err
}

func (t *apiProvider) Pod(namespace, name string) (*v1.Pod, error) {
	ctx := context.Background()
	pod, err := t.clientset.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	// needed because of https://github.com/kubernetes/client-go/issues/861
	pod.SetGroupVersionKind(v1.SchemeGroupVersion.WithKind("Pod"))
	return pod, err
}

func (t *apiProvider) CronJobs(namespace v1.Namespace) ([]batchv1.CronJob, error) {
	ctx := context.Background()
	list, err := t.clientset.BatchV1().CronJobs(namespace.Name).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	// needed because of https://github.com/kubernetes/client-go/issues/861
	for i := range list.Items {
		list.Items[i].SetGroupVersionKind(batchv1.SchemeGroupVersion.WithKind("CronJob"))
	}
	return list.Items, err
}

func (t *apiProvider) CronJob(namespace, name string) (*batchv1.CronJob, error) {
	ctx := context.Background()
	cronjob, err := t.clientset.BatchV1().CronJobs(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	// needed because of https://github.com/kubernetes/client-go/issues/861
	cronjob.SetGroupVersionKind(batchv1.SchemeGroupVersion.WithKind("CronJob"))
	return cronjob, err
}

func (t *apiProvider) StatefulSets(namespace v1.Namespace) ([]appsv1.StatefulSet, error) {
	ctx := context.Background()
	list, err := t.clientset.AppsV1().StatefulSets(namespace.Name).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	// needed because of https://github.com/kubernetes/client-go/issues/861
	for i := range list.Items {
		list.Items[i].SetGroupVersionKind(appsv1.SchemeGroupVersion.WithKind("StatefulSet"))
	}
	return list.Items, err
}

func (t *apiProvider) StatefulSet(namespace, name string) (*appsv1.StatefulSet, error) {
	ctx := context.Background()
	statefulset, err := t.clientset.AppsV1().StatefulSets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	// needed because of https://github.com/kubernetes/client-go/issues/861
	statefulset.SetGroupVersionKind(appsv1.SchemeGroupVersion.WithKind("StatefulSet"))
	return statefulset, err
}

func (t *apiProvider) Deployments(namespace v1.Namespace) ([]appsv1.Deployment, error) {
	ctx := context.Background()
	list, err := t.clientset.AppsV1().Deployments(namespace.Name).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	// needed because of https://github.com/kubernetes/client-go/issues/861
	for i := range list.Items {
		list.Items[i].SetGroupVersionKind(appsv1.SchemeGroupVersion.WithKind("Deployment"))
	}
	return list.Items, err
}

func (t *apiProvider) Deployment(namespace, name string) (*appsv1.Deployment, error) {
	ctx := context.Background()
	deployment, err := t.clientset.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	// needed because of https://github.com/kubernetes/client-go/issues/861
	deployment.SetGroupVersionKind(appsv1.SchemeGroupVersion.WithKind("Deployment"))
	return deployment, err
}

func (t *apiProvider) Jobs(namespace v1.Namespace) ([]batchv1.Job, error) {
	ctx := context.Background()
	list, err := t.clientset.BatchV1().Jobs(namespace.Name).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	// needed because of https://github.com/kubernetes/client-go/issues/861
	for i := range list.Items {
		list.Items[i].SetGroupVersionKind(batchv1.SchemeGroupVersion.WithKind("Job"))
	}
	return list.Items, err
}

func (t *apiProvider) Job(namespace, name string) (*batchv1.Job, error) {
	ctx := context.Background()
	job, err := t.clientset.BatchV1().Jobs(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	// needed because of https://github.com/kubernetes/client-go/issues/861
	job.SetGroupVersionKind(batchv1.SchemeGroupVersion.WithKind("Job"))
	return job, err
}

func (t *apiProvider) ReplicaSets(namespace v1.Namespace) ([]appsv1.ReplicaSet, error) {
	ctx := context.Background()
	list, err := t.clientset.AppsV1().ReplicaSets(namespace.Name).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	// needed because of https://github.com/kubernetes/client-go/issues/861
	for i := range list.Items {
		list.Items[i].SetGroupVersionKind(appsv1.SchemeGroupVersion.WithKind("ReplicaSet"))
	}
	return list.Items, err
}

func (t *apiProvider) ReplicaSet(namespace, name string) (*appsv1.ReplicaSet, error) {
	ctx := context.Background()
	replicaset, err := t.clientset.AppsV1().ReplicaSets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	// needed because of https://github.com/kubernetes/client-go/issues/861
	replicaset.SetGroupVersionKind(appsv1.SchemeGroupVersion.WithKind("ReplicaSet"))
	return replicaset, err
}

func (t *apiProvider) DaemonSets(namespace v1.Namespace) ([]appsv1.DaemonSet, error) {
	ctx := context.Background()
	list, err := t.clientset.AppsV1().DaemonSets(namespace.Name).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	// needed because of https://github.com/kubernetes/client-go/issues/861
	for i := range list.Items {
		list.Items[i].SetGroupVersionKind(appsv1.SchemeGroupVersion.WithKind("DaemonSet"))
	}
	return list.Items, err
}

func (t *apiProvider) DaemonSet(namespace, name string) (*appsv1.DaemonSet, error) {
	ctx := context.Background()
	daemonset, err := t.clientset.AppsV1().DaemonSets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	// needed because of https://github.com/kubernetes/client-go/issues/861
	daemonset.SetGroupVersionKind(appsv1.SchemeGroupVersion.WithKind("DaemonSet"))
	return daemonset, err
}

func (t *apiProvider) Secret(namespace, name string) (*v1.Secret, error) {
	ctx := context.Background()
	secret, err := t.clientset.CoreV1().Secrets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return secret, err
}
