package k8s

import (
	"bytes"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/motor/providers/k8s/resources"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sRuntime "k8s.io/apimachinery/pkg/runtime"
)

type manifestParser struct {
	objects            []runtime.Object
	namespace          string
	selectedResourceID string
}

func newManifestParser(manifest []byte, namespace, selectedResourceID string) (manifestParser, error) {
	objs, err := load(manifest)
	if err != nil {
		return manifestParser{}, errors.Wrap(err, "could not query resource objects")
	}
	log.Debug().Msgf("found %d resource objects", len(objs))
	return manifestParser{
		objects:            objs,
		namespace:          namespace,
		selectedResourceID: selectedResourceID,
	}, nil
}

func (t *manifestParser) SupportedResourceTypes() (*resources.ApiResourceIndex, error) {
	return resources.NewApiResourceIndex(), nil
}

func (t *manifestParser) Nodes() ([]v1.Node, error) {
	return []v1.Node{}, nil
}

// Namespaces iterates over all file-based manifests and extracts all namespaces used
func (t *manifestParser) Namespaces() ([]v1.Namespace, error) {
	namespaceMap := map[string]struct{}{}
	for i := range t.objects {
		res := t.objects[i]
		o, err := meta.Accessor(res)
		if err != nil {
			return nil, err
		}

		namespaceMap[o.GetNamespace()] = struct{}{}
	}

	var nss []v1.Namespace

	// NOTE: this only does the minimal required for our current implementation
	// going forward we may need a bit more information
	for k := range namespaceMap {
		nss = append(nss, v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: k,
			},
		})
	}

	return nss, nil
}

func (t *manifestParser) Pod(namespace string, name string) (*v1.Pod, error) {
	result, err := t.Resources("pods.v1.", name, namespace)
	if err != nil {
		return nil, err
	}

	if len(result.Resources) > 1 {
		return nil, errors.New("multiple pods found")
	}
	foundPod, ok := result.Resources[0].(*v1.Pod)
	if !ok {
		return nil, errors.New("could not convert k8s resource to pod")
	}

	if foundPod.Name == "" {
		return nil, errors.New("pod not found")
	}
	return foundPod, nil
}

func (t *manifestParser) Pods(namespace v1.Namespace) ([]v1.Pod, error) {
	result, err := t.Resources("pods.v1.", "", namespace.GetNamespace())
	if err != nil {
		return nil, err
	}

	var pods []v1.Pod
	for i := range result.Resources {
		r := result.Resources[i]

		pod, ok := r.(*v1.Pod)
		if !ok {
			log.Warn().Msg("could not convert k8s resource to pod")
			continue
		}
		pods = append(pods, *pod)
	}

	return pods, nil
}

func (t *manifestParser) Deployment(namespace string, name string) (*appsv1.Deployment, error) {
	result, err := t.Resources("deployments.appsv1.", name, namespace)
	if err != nil {
		return nil, err
	}

	if len(result.Resources) > 1 {
		return nil, errors.New("multiple deployments found")
	}
	foundDeployment, ok := result.Resources[0].(*appsv1.Deployment)
	if !ok {
		return nil, errors.New("could not convert k8s resource to deployment")
	}

	if foundDeployment.Name == "" {
		return nil, errors.New("deployment not found")
	}
	return foundDeployment, nil
}

func (t *manifestParser) Deployments(namespace v1.Namespace) ([]appsv1.Deployment, error) {
	result, err := t.Resources("deployments.v1.apps", "", namespace.GetNamespace())
	if err != nil {
		return nil, err
	}

	var deployments []appsv1.Deployment
	for i := range result.Resources {
		r := result.Resources[i]

		deployment, ok := r.(*appsv1.Deployment)
		if !ok {
			log.Error().Err(err).Msg("could not convert k8s resource to deployment")
			return nil, err
		}
		deployments = append(deployments, *deployment)
	}

	return deployments, nil
}

func (t *manifestParser) resourceIndex() (*resources.ApiResourceIndex, error) {
	// find root nodes
	resList, err := resources.CachedServerResources()
	if err != nil {
		return nil, err
	}

	resTypes, err := resources.ResourceIndex(resList)
	if err != nil {
		return nil, err
	}
	return resTypes, nil
}

func (t *manifestParser) Resources(kind string, name string, namespace string) (*ResourceResult, error) {
	var ns string
	if namespace == "" {
		ns = t.namespace
	} else {
		ns = namespace
	}
	allNs := false
	if ns == "" {
		allNs = true
	}

	resTypes, err := t.resourceIndex()
	if err != nil {
		return nil, err
	}

	resType, err := resTypes.Lookup(kind)
	if err != nil {
		return nil, err
	}

	resources, err := resources.FilterResource(resType, t.objects, name, namespace)

	return &ResourceResult{
		Name:         name,
		Kind:         kind,
		ResourceType: resType,
		Resources:    resources,
		Namespace:    ns,
		AllNs:        allNs,
	}, err
}

func (t *manifestParser) CronJob(namespace string, name string) (*batchv1.CronJob, error) {
	result, err := t.Resources("cronjobs.v1.batch", name, namespace)
	if err != nil {
		return nil, err
	}

	if len(result.Resources) > 1 {
		return nil, errors.New("multiple cronjobs found")
	}
	foundCronJob, ok := result.Resources[0].(*batchv1.CronJob)
	if !ok {
		return nil, errors.New("could not convert k8s resource to cronjob")
	}

	if foundCronJob.Name == "" {
		return nil, errors.New("cronjob not found")
	}
	return foundCronJob, nil
}

func (t *manifestParser) CronJobs(namespace v1.Namespace) ([]batchv1.CronJob, error) {
	result, err := t.Resources("cronjobs.v1.batch", "", namespace.GetNamespace())
	if err != nil {
		return nil, err
	}

	var cronJobs []batchv1.CronJob
	for i := range result.Resources {
		r := result.Resources[i]

		cronJob, ok := r.(*batchv1.CronJob)
		if !ok {
			log.Warn().Msg("could not convert k8s resource to cronjob")
			continue
		}
		cronJobs = append(cronJobs, *cronJob)
	}

	return cronJobs, nil
}

func (t *manifestParser) StatefulSet(namespace string, name string) (*appsv1.StatefulSet, error) {
	result, err := t.Resources("statefulsets.v1.apps", name, namespace)
	if err != nil {
		return nil, err
	}

	if len(result.Resources) > 1 {
		return nil, errors.New("multiple statefulsets found")
	}
	foundStatefulSet, ok := result.Resources[0].(*appsv1.StatefulSet)
	if !ok {
		return nil, errors.New("could not convert k8s resource to statefulset")
	}

	if foundStatefulSet.Name == "" {
		return nil, errors.New("statefulset not found")
	}
	return foundStatefulSet, nil
}

func (t *manifestParser) StatefulSets(namespace v1.Namespace) ([]appsv1.StatefulSet, error) {
	result, err := t.Resources("statefulsets.v1.apps", "", namespace.GetNamespace())
	if err != nil {
		return nil, err
	}

	var statefulSets []appsv1.StatefulSet
	for i := range result.Resources {
		r := result.Resources[i]

		statefulSet, ok := r.(*appsv1.StatefulSet)
		if !ok {
			log.Warn().Msg("could not convert k8s resource to statefulset")
			continue
		}
		statefulSets = append(statefulSets, *statefulSet)
	}

	return statefulSets, nil
}

func (t *manifestParser) Job(namespace string, name string) (*batchv1.Job, error) {
	result, err := t.Resources("jobs.v1.batch", name, namespace)
	if err != nil {
		return nil, err
	}

	if len(result.Resources) > 1 {
		return nil, errors.New("multiple jobs found")
	}
	foundJob, ok := result.Resources[0].(*batchv1.Job)
	if !ok {
		return nil, errors.New("could not convert k8s resource to job")
	}

	if foundJob.Name == "" {
		return nil, errors.New("job not found")
	}
	return foundJob, nil
}

func (t *manifestParser) Jobs(namespace v1.Namespace) ([]batchv1.Job, error) {
	result, err := t.Resources("jobs.v1.batch", "", namespace.GetNamespace())
	if err != nil {
		return nil, err
	}

	var jobs []batchv1.Job
	for i := range result.Resources {
		r := result.Resources[i]

		job, ok := r.(*batchv1.Job)
		if !ok {
			log.Warn().Msg("could not convert k8s resource to job")
			continue
		}
		jobs = append(jobs, *job)
	}

	return jobs, nil
}

func (t *manifestParser) ReplicaSet(namespace string, name string) (*appsv1.ReplicaSet, error) {
	result, err := t.Resources("replicasets.v1.apps", name, namespace)
	if err != nil {
		return nil, err
	}

	if len(result.Resources) > 1 {
		return nil, errors.New("multiple replicasets found")
	}
	foundReplicaSet, ok := result.Resources[0].(*appsv1.ReplicaSet)
	if !ok {
		return nil, errors.New("could not convert k8s resource to replicaset")
	}

	if foundReplicaSet.Name == "" {
		return nil, errors.New("replicaset not found")
	}
	return foundReplicaSet, nil
}

func (t *manifestParser) ReplicaSets(namespace v1.Namespace) ([]appsv1.ReplicaSet, error) {
	result, err := t.Resources("replicasets.v1.apps", "", namespace.GetNamespace())
	if err != nil {
		return nil, err
	}

	var replicaSets []appsv1.ReplicaSet
	for i := range result.Resources {
		r := result.Resources[i]

		replicaSet, ok := r.(*appsv1.ReplicaSet)
		if !ok {
			log.Warn().Msg("could not convert k8s resource to replicaset")
			continue
		}
		replicaSets = append(replicaSets, *replicaSet)
	}

	return replicaSets, nil
}

func (t *manifestParser) DaemonSet(namespace string, name string) (*appsv1.DaemonSet, error) {
	result, err := t.Resources("daemonsets.appsv1.", name, namespace)
	if err != nil {
		return nil, err
	}

	if len(result.Resources) > 1 {
		return nil, errors.New("multiple daemonsets found")
	}
	foundDaemonSet, ok := result.Resources[0].(*appsv1.DaemonSet)
	if !ok {
		return nil, errors.New("could not convert k8s resource to daemonset")
	}

	if foundDaemonSet.Name == "" {
		return nil, errors.New("daemonset not found")
	}
	return foundDaemonSet, nil
}

func (t *manifestParser) DaemonSets(namespace v1.Namespace) ([]appsv1.DaemonSet, error) {
	// iterate over all resources and extract the daemonsets

	result, err := t.Resources("daemonsets.v1.apps", "", namespace.GetNamespace())
	if err != nil {
		return nil, err
	}

	var daemonsets []appsv1.DaemonSet
	for i := range result.Resources {
		r := result.Resources[i]

		daemonset, ok := r.(*appsv1.DaemonSet)
		if !ok {
			log.Error().Err(err).Msg("could not convert k8s resource to daemonset")
			return nil, err
		}
		daemonsets = append(daemonsets, *daemonset)
	}

	return daemonsets, nil
}

func (t *manifestParser) Secret(namespace, name string) (*v1.Secret, error) {
	result, err := t.Resources("secrets.v1.", name, namespace)
	if err != nil {
		return nil, err
	}

	if len(result.Resources) > 1 {
		return nil, errors.New("multiple secrets found")
	}
	foundSecret, ok := result.Resources[0].(*v1.Secret)
	if !ok {
		return nil, errors.New("could not convert k8s resource to secret")
	}

	if foundSecret.Name == "" {
		return nil, errors.New("secret not found")
	}
	return foundSecret, nil
}

func load(manifest []byte) ([]k8sRuntime.Object, error) {
	res := []k8sRuntime.Object{}
	if len(manifest) > 0 {
		objects, err := resources.ResourcesFromManifest(bytes.NewReader(manifest))
		if err != nil {
			return nil, err
		}
		res = append(res, objects...)
	}

	return res, nil
}
