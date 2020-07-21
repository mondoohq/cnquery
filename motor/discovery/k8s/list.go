package k8s

import (
	"strings"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/pkg/errors"
	"go.mondoo.io/mondoo/motor/discovery/docker"
	"go.mondoo.io/mondoo/motor/motorapi"
	"go.mondoo.io/mondoo/motor/runtime"
	"go.mondoo.io/mondoo/nexus/assets"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
)

const dockerPullablePrefix = "docker-pullable://"

type PodContainerImage struct {
	Image         string
	ResolvedImage string
	Namespace     string
	Pod           string
	InitContainer *string
	Container     *string
}

func ListPodImages(context string, namespaceFilter []string, podFilter []string) ([]*assets.Asset, error) {
	var configFlags *genericclioptions.ConfigFlags
	configFlags = genericclioptions.NewConfigFlags(false)

	if len(context) > 0 {
		configFlags.Context = &context
	}

	config, err := configFlags.ToRESTConfig()
	if err != nil {
		return nil, errors.Wrap(err, "could not read kubectl config")
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, errors.Wrap(err, "could not create kubernetes clientset")
	}

	namespaces, err := clientset.CoreV1().Namespaces().List(metav1.ListOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "could not list kubernetes namespaces")
	}

	runningImages := []*assets.Asset{}
	for i := range namespaces.Items {
		namespace := namespaces.Items[i]
		if !isIncluded(namespace.Name, namespaceFilter) {
			continue
		}

		pods, err := clientset.CoreV1().Pods(namespace.Name).List(metav1.ListOptions{})
		if err != nil {
			return nil, errors.Wrap(err, "failed to list pods")
		}

		for j := range pods.Items {
			pod := pods.Items[j]

			if !isIncluded(pod.Name, podFilter) {
				continue
			}

			for ics := range pod.Status.InitContainerStatuses {
				containerStatus := pod.Status.InitContainerStatuses[ics]
				runningImages = append(runningImages, toAsset(pod, containerStatus))
			}

			for cs := range pod.Status.ContainerStatuses {
				containerStatus := pod.Status.ContainerStatuses[cs]
				runningImages = append(runningImages, toAsset(pod, containerStatus))
			}
		}
	}

	return runningImages, nil
}

func isIncluded(value string, included []string) bool {
	if len(included) == 0 {
		return true
	}

	for _, ex := range included {
		if strings.ToLower(ex) == strings.ToLower(value) {
			return true
		}
	}

	return false
}

// TODO: should we ignore pods with CreateContainerError
func toAsset(pod v1.Pod, status v1.ContainerStatus) *assets.Asset {
	resolvedImage := status.ImageID
	if strings.HasPrefix(resolvedImage, dockerPullablePrefix) {
		resolvedImage = strings.TrimPrefix(resolvedImage, dockerPullablePrefix)
	}

	connection := resolvedImage
	parentRef := ""

	// stopped pods may not include the resolved image
	if len(resolvedImage) == 0 {
		connection = status.Image
	}

	// parse resolved image to extract the digest
	if len(resolvedImage) > 0 {
		digest, err := name.NewDigest(resolvedImage, name.WeakValidation)
		if err == nil {
			parentRef = docker.MondooContainerImageID(digest.DigestStr())
		}
	}

	// parse image name to extract tags
	tagName := ""
	if len(status.Image) > 0 {
		tag, err := name.NewTag(resolvedImage, name.WeakValidation)
		if err == nil {
			tagName = tag.TagStr()
		}
	}

	asset := &assets.Asset{
		Name: pod.Name,

		ReferenceIDs:      []string{MondooKubernetesPodID(string(pod.UID))},
		ParentReferenceID: parentRef,

		Platform: &assets.Platform{
			Kind:    assets.Kind_KIND_CONTAINER,
			Runtime: runtime.RUNTIME_KUBERNETES,
		},

		Connections: []*motorapi.TransportConfig{
			&motorapi.TransportConfig{
				Backend: motorapi.TransportBackend_CONNECTION_DOCKER_IMAGE,
				Host:    connection,
			},
		},
		State:  mapPodStatus(pod.Status),
		Labels: make(map[string]string),
	}

	for key := range pod.Annotations {
		asset.Labels[key] = pod.Annotations[key]
	}

	// fetch k8s specific metadata
	asset.Labels["k8s.mondoo.app/name"] = pod.Name
	asset.Labels["k8s.mondoo.app/namespace"] = pod.Namespace
	asset.Labels["k8s.mondoo.app/cluster-name"] = pod.ClusterName
	asset.Labels["k8s.mondoo.app/status/name"] = status.Name
	asset.Labels["k8s.mondoo.app/status/image"] = status.Image
	asset.Labels["docker.io/tags"] = tagName
	return asset
}

func mapPodStatus(status v1.PodStatus) assets.State {
	switch status.Phase {
	case v1.PodPending:
		return assets.State_STATE_PENDING
	case v1.PodFailed:
		return assets.State_STATE_ERROR
	case v1.PodRunning:
		return assets.State_STATE_RUNNING
	case v1.PodSucceeded:
		return assets.State_STATE_PENDING
	case v1.PodUnknown:
		return assets.State_STATE_UNKNOWN
	default:
		return assets.State_STATE_UNKNOWN
	}
}

// TODO: find a method to uniquely identify a kubernetes cluster
// see https://github.com/kubernetes/kubernetes/issues/77487, kubesystem uid
func MondooKubernetesPodID(podId string) string {
	return "//platformid.api.mondoo.app/runtime/kubernetes/pod/" + podId
}
