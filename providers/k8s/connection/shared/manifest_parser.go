// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package shared

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/providers/k8s/connection/shared/resources"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	k8sRuntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
)

type ManifestParser struct {
	Objects            []runtime.Object
	namespace          string
	selectedResourceID string
}

func NewManifestParser(manifest []byte, namespace, selectedResourceID string) (ManifestParser, error) {
	objs, err := load(manifest)
	if err != nil {
		return ManifestParser{}, errors.Wrap(err, "could not query resource objects")
	}
	log.Debug().Msgf("found %d resource objects", len(objs))
	return ManifestParser{
		Objects:            objs,
		namespace:          namespace,
		selectedResourceID: selectedResourceID,
	}, nil
}

func (t *ManifestParser) SupportedResourceTypes() (*resources.ApiResourceIndex, error) {
	return resources.NewApiResourceIndex(), nil
}

func (t *ManifestParser) Nodes() ([]v1.Node, error) {
	return []v1.Node{}, nil
}

func (t *ManifestParser) Namespace(name string) (*v1.Namespace, error) {
	nss, err := t.Namespaces()
	if err != nil {
		return nil, err
	}

	for i := range nss {
		if nss[i].Name == name {
			return &nss[i], nil
		}
	}
	return nil, fmt.Errorf("namespace %s not found", name)
}

// Namespaces iterates over all file-based manifests and extracts all namespaces used
func (t *ManifestParser) Namespaces() ([]v1.Namespace, error) {
	namespaceMap := map[string]struct{}{}
	for i := range t.Objects {
		res := t.Objects[i]
		o, err := meta.Accessor(res)
		if err == nil {
			ns := o.GetNamespace()
			// There are types of resources that do not have meta data. Instead of erroring
			// skip them.
			namespaceMap[ns] = struct{}{}
		}
	}

	var nss []v1.Namespace

	// NOTE: this only does the minimal required for our current implementation
	// going forward we may need a bit more information
	for k := range namespaceMap {
		nss = append(nss, v1.Namespace{
			TypeMeta: metav1.TypeMeta{
				Kind: "Namespace",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: k,
			},
		})
	}

	return nss, nil
}

func (t *ManifestParser) Pod(namespace string, name string) (*v1.Pod, error) {
	result, err := t.Resources("pods.v1.", name, namespace)
	if err != nil {
		return nil, err
	}

	if len(result.Resources) > 1 {
		return nil, errors.New("multiple pods found")
	}
	if len(result.Resources) == 0 {
		return nil, fmt.Errorf("pod %s not found", name)
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

func (t *ManifestParser) Pods(namespace v1.Namespace) ([]*v1.Pod, error) {
	result, err := t.Resources("pods.v1.", "", namespace.GetName())
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

	return sliceToPtrSlice(pods), nil
}

func (t *ManifestParser) Deployment(namespace string, name string) (*appsv1.Deployment, error) {
	result, err := t.Resources("deployments.v1.apps", name, namespace)
	if err != nil {
		return nil, err
	}

	if len(result.Resources) > 1 {
		return nil, errors.New("multiple deployments found")
	}
	if len(result.Resources) == 0 {
		return nil, fmt.Errorf("deployment %s not found", name)
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

func (t *ManifestParser) Deployments(namespace v1.Namespace) ([]*appsv1.Deployment, error) {
	result, err := t.Resources("deployments.v1.apps", "", namespace.GetName())
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

	return sliceToPtrSlice(deployments), nil
}

func (t *ManifestParser) resourceIndex() (*resources.ApiResourceIndex, error) {
	// find root nodes
	resList, err := resources.CachedServerResources()
	if err != nil {
		return nil, err
	}

	resTypes, err := resources.ResourceIndex(resList)
	if err != nil {
		return nil, err
	}

	// We have a static list of API resources for the manifest provider. Here we dynamically add
	// API resources for every Unstructured object we encounter.
	for _, o := range t.Objects {
		if unstr, ok := o.(*unstructured.Unstructured); ok {
			gvk := unstr.GetObjectKind().GroupVersionKind()

			// Only add the API resource if it wasn't added already.
			if _, err := resTypes.Lookup(strings.ToLower(gvk.GroupKind().String())); err != nil {
				apiRes := resources.ApiResource{
					GroupVersion: unstr.GroupVersionKind().GroupVersion(),
					Resource: metav1.APIResource{
						// The k8s API doesn't add just 's'. For kinds that end on 's' the suffix is 'es' but
						// that doesn't change anything for us. That's why only the basic logic is implemented.
						Name:         strings.ToLower(gvk.Kind) + "s",
						SingularName: strings.ToLower(gvk.Kind),
						Verbs:        []string{"list"},
						Kind:         gvk.Kind,
						Version:      gvk.Version,
						Group:        gvk.Group,
					},
				}
				resTypes.Add(apiRes)
			}
		}
	}

	return resTypes, nil
}

func (t *ManifestParser) Resources(kind string, name string, namespace string) (*ResourceResult, error) {
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

	res, err := resources.FilterResource(resType, t.Objects, name, namespace)

	return &ResourceResult{
		Name:         name,
		Kind:         kind,
		ResourceType: resType,
		Resources:    res,
		Namespace:    ns,
		AllNs:        allNs,
	}, err
}

func (t *ManifestParser) CronJob(namespace string, name string) (*batchv1.CronJob, error) {
	result, err := t.Resources("cronjobs.v1.batch", name, namespace)
	if err != nil {
		return nil, err
	}

	if len(result.Resources) > 1 {
		return nil, errors.New("multiple cronjobs found")
	}
	if len(result.Resources) == 0 {
		return nil, fmt.Errorf("cronjob %s not found", name)
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

func (t *ManifestParser) CronJobs(namespace v1.Namespace) ([]*batchv1.CronJob, error) {
	result, err := t.Resources("cronjobs.v1.batch", "", namespace.GetName())
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

	return sliceToPtrSlice(cronJobs), nil
}

func (t *ManifestParser) StatefulSet(namespace string, name string) (*appsv1.StatefulSet, error) {
	result, err := t.Resources("statefulsets.v1.apps", name, namespace)
	if err != nil {
		return nil, err
	}

	if len(result.Resources) > 1 {
		return nil, errors.New("multiple statefulsets found")
	}
	if len(result.Resources) == 0 {
		return nil, fmt.Errorf("statefulset %s not found", name)
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

func (t *ManifestParser) StatefulSets(namespace v1.Namespace) ([]*appsv1.StatefulSet, error) {
	result, err := t.Resources("statefulsets.v1.apps", "", namespace.GetName())
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

	return sliceToPtrSlice(statefulSets), nil
}

func (t *ManifestParser) Job(namespace string, name string) (*batchv1.Job, error) {
	result, err := t.Resources("jobs.v1.batch", name, namespace)
	if err != nil {
		return nil, err
	}

	if len(result.Resources) > 1 {
		return nil, errors.New("multiple jobs found")
	}
	if len(result.Resources) == 0 {
		return nil, fmt.Errorf("job %s not found", name)
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

func (t *ManifestParser) Jobs(namespace v1.Namespace) ([]*batchv1.Job, error) {
	result, err := t.Resources("jobs.v1.batch", "", namespace.GetName())
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

	return sliceToPtrSlice(jobs), nil
}

func (t *ManifestParser) ReplicaSet(namespace string, name string) (*appsv1.ReplicaSet, error) {
	result, err := t.Resources("replicasets.v1.apps", name, namespace)
	if err != nil {
		return nil, err
	}

	if len(result.Resources) > 1 {
		return nil, errors.New("multiple replicasets found")
	}
	if len(result.Resources) == 0 {
		return nil, fmt.Errorf("replicaset %s not found", name)
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

func (t *ManifestParser) ReplicaSets(namespace v1.Namespace) ([]*appsv1.ReplicaSet, error) {
	result, err := t.Resources("replicasets.v1.apps", "", namespace.GetName())
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

	return sliceToPtrSlice(replicaSets), nil
}

func (t *ManifestParser) DaemonSet(namespace string, name string) (*appsv1.DaemonSet, error) {
	result, err := t.Resources("daemonsets.v1.apps", name, namespace)
	if err != nil {
		return nil, err
	}

	if len(result.Resources) > 1 {
		return nil, errors.New("multiple daemonsets found")
	}
	if len(result.Resources) == 0 {
		return nil, fmt.Errorf("daemonset %s not found", name)
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

func (t *ManifestParser) DaemonSets(namespace v1.Namespace) ([]*appsv1.DaemonSet, error) {
	// iterate over all resources and extract the daemonsets

	result, err := t.Resources("daemonsets.v1.apps", "", namespace.GetName())
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

	return sliceToPtrSlice(daemonsets), nil
}

func (t *ManifestParser) Ingress(namespace, name string) (*networkingv1.Ingress, error) {
	result, err := t.Resources("ingresses.v1.networking.k8s.io", name, namespace)
	if err != nil {
		return nil, err
	}

	if len(result.Resources) > 1 {
		return nil, errors.New("multiple ingresses found")
	}
	if len(result.Resources) == 0 {
		return nil, fmt.Errorf("ingress %s not found", name)
	}
	foundIngress, ok := result.Resources[0].(*networkingv1.Ingress)
	if !ok {
		return nil, errors.New("could not convert k8s resource to ingress")
	}

	if foundIngress.Name == "" {
		return nil, errors.New("ingress not found")
	}
	return foundIngress, nil
}

func (t *ManifestParser) Ingresses(namespace v1.Namespace) ([]*networkingv1.Ingress, error) {
	result, err := t.Resources("ingresses.v1.networking.k8s.io", "", namespace.GetName())
	if err != nil {
		return nil, err
	}

	var ingresses []networkingv1.Ingress
	for i := range result.Resources {
		r := result.Resources[i]

		ingress, ok := r.(*networkingv1.Ingress)
		if !ok {
			log.Error().Err(err).Msg("could not convert k8s resource to ingress")
			return nil, err
		}
		ingresses = append(ingresses, *ingress)
	}

	return sliceToPtrSlice(ingresses), nil
}

func (t *ManifestParser) Secret(namespace, name string) (*v1.Secret, error) {
	result, err := t.Resources("secrets.v1.", name, namespace)
	if err != nil {
		return nil, err
	}

	if len(result.Resources) > 1 {
		return nil, errors.New("multiple secrets found")
	}
	if len(result.Resources) == 0 {
		return nil, fmt.Errorf("secret %s not found", name)
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

	resList, err := resources.CachedServerResources()
	if err != nil {
		return nil, err
	}

	resTypes, err := resources.ResourceIndex(resList)
	if err != nil {
		return nil, err
	}

	// Every unstructured object here is an object that we couldn't match to an actual type.
	// Such objects we treat as custom resources and should end up in the k8s.customresources list.
	// To do that we need to have a CRD for every kind that couldn't be matched to a type. Here we create
	// the related CRD for every type that needs it.
	addedCrds := make(map[string]struct{})
	for _, o := range res {
		if unstr, ok := o.(*unstructured.Unstructured); ok {
			gvk := unstr.GetObjectKind().GroupVersionKind()
			if _, err := resTypes.Lookup(gvk.Kind); err != nil {
				// Only add the CRD once.
				crdName := strings.ToLower(fmt.Sprintf("%s.%s", gvk.Kind, gvk.Group))
				if _, ok := addedCrds[crdName]; ok {
					continue
				}

				addedCrds[crdName] = struct{}{}
				res = append(
					res,
					&apiextensionsv1.CustomResourceDefinition{TypeMeta: metav1.TypeMeta{Kind: "CustomResourceDefinition"}, ObjectMeta: metav1.ObjectMeta{Name: crdName}})
			}
		}
	}

	return res, nil
}

func ProjectNameFromPath(file string) string {
	// if it is a local file (which may not be true)
	name := ""
	fi, err := os.Stat(file)
	if err == nil {
		if fi.IsDir() && fi.Name() != "." {
			name = "directory " + fi.Name()
		} else if fi.IsDir() {
			name = fi.Name()
		} else {
			name = filepath.Base(fi.Name())
			extension := filepath.Ext(name)
			name = strings.TrimSuffix(name, extension)
		}
	} else {
		// it is not a local file, so we try to be a bit smart
		name = path.Base(file)
		extension := path.Ext(name)
		name = strings.TrimSuffix(name, extension)
	}

	// if the path is . we read the current directory
	if name == "." {
		abspath, err := filepath.Abs(name)
		if err == nil {
			name = ProjectNameFromPath(abspath)
		}
	}

	return name
}

func LoadManifestFile(manifestFile string) ([]byte, error) {
	log.Debug().Str("filename", manifestFile).Msg("loading manifest file")
	var input io.Reader

	// return all resources from manifest
	filenames := []string{}

	fi, err := os.Stat(manifestFile)
	if err != nil {
		return nil, err
	}

	if fi.IsDir() {
		yamlDecoder := yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)
		filepath.WalkDir(manifestFile, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			// only load yaml files for now
			if !d.IsDir() {
				ext := filepath.Ext(path)
				if ext != ".yaml" && ext != ".yml" {
					log.Debug().Str("file", path).Msg("ignore file, no .yaml or .yml ending")
					return nil
				}
				// check whether this is valid k8s yaml
				content, err := os.ReadFile(path)
				if err != nil {
					log.Debug().Str("file", path).Err(err).Msg("ignore file, could not read file")
					return nil
				}
				// At this point, we do not care about specific schemes, just whether the file is a valid k8s yaml
				_, _, err = yamlDecoder.Decode(content, nil, nil)
				if err != nil {
					// the err contains the file content, which is not useful in the output
					errorString := ""
					if len(err.Error()) > 40 {
						errorString = err.Error()[:40] + "..."
					} else {
						errorString = err.Error()
					}
					log.Debug().Str("file", path).Str("error", errorString).Msg("ignore file, no valid kubernetes yaml")
					return nil
				}
				log.Debug().Str("file", path).Msg("add file to manifest loading")
				filenames = append(filenames, path)
			}

			return nil
		})
	} else {
		filenames = append(filenames, manifestFile)
	}

	input, err = resources.MergeManifestFiles(filenames)
	if err != nil {
		return nil, err
	}

	return io.ReadAll(input)
}
