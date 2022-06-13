package k8s

import (
	"strings"

	"github.com/google/go-containerregistry/pkg/name"

	"go.mondoo.io/mondoo/motor/discovery/container_registry"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/motor/asset"
	"go.mondoo.io/mondoo/motor/transports/k8s"
	v1 "k8s.io/api/core/v1"
)

const dockerPullablePrefix = "docker-pullable://"

func ListPodImages(transport *k8s.Transport, namespaceFilter []string) ([]*asset.Asset, error) {
	namespaces, err := transport.Connector().Namespaces()
	if err != nil {
		return nil, errors.Wrap(err, "could not list kubernetes namespaces")
	}

	runningImages := []*asset.Asset{}
	for i := range namespaces.Items {
		namespace := namespaces.Items[i]
		if !isIncluded(namespace.Name, namespaceFilter) {
			log.Info().Str("namespace", namespace.Name).Strs("filter", namespaceFilter).Msg("namespace not included")
			continue
		}

		pods, err := transport.Connector().Pods(namespace)
		if err != nil {
			return nil, errors.Wrap(err, "failed to list pods")
		}

		for j := range pods.Items {
			assets, err := resolvePodAssets(pods.Items[j])
			if err != nil {
				return nil, err
			}
			runningImages = append(runningImages, assets...)
		}
	}

	return runningImages, nil
}

func isIncluded(value string, included []string) bool {
	if len(included) == 0 {
		return true
	}

	for _, ex := range included {
		if strings.EqualFold(ex, value) {
			return true
		}
	}

	return false
}

func resolvePodAssets(pod v1.Pod) ([]*asset.Asset, error) {
	resolved := []*asset.Asset{}

	// it is best to read the image from the container status since it is resolved
	// and more accurate, for static file scan we also need to fall-back to pure spec
	// since the status will not be set
	for ics := range pod.Status.InitContainerStatuses {
		containerStatus := pod.Status.InitContainerStatuses[ics]
		image, resolvedImage := resolveContainerImageFromStatus(containerStatus)
		a, err := newPodImageAsset(image, resolvedImage)
		if err != nil {
			return nil, err
		}
		resolved = append(resolved, a)
	}

	// fall-back to spec
	if len(pod.Spec.InitContainers) > 0 && len(pod.Status.InitContainerStatuses) == 0 {
		for i := range pod.Spec.InitContainers {
			initContainer := pod.Spec.InitContainers[i]
			image, resolvedImage := resolveContainerImage(initContainer)
			a, err := newPodImageAsset(image, resolvedImage)
			if err != nil {
				return nil, err
			}
			resolved = append(resolved, a)
		}
	}

	for cs := range pod.Status.ContainerStatuses {
		containerStatus := pod.Status.ContainerStatuses[cs]
		image, resolvedImage := resolveContainerImageFromStatus(containerStatus)
		a, err := newPodImageAsset(image, resolvedImage)
		if err != nil {
			return nil, err
		}
		resolved = append(resolved, a)
	}

	// fall-back to spec
	if len(pod.Spec.Containers) > 0 && len(pod.Status.ContainerStatuses) == 0 {
		for i := range pod.Spec.Containers {
			container := pod.Spec.Containers[i]
			image, resolvedImage := resolveContainerImage(container)
			a, err := newPodImageAsset(image, resolvedImage)
			if err != nil {
				return nil, err
			}
			resolved = append(resolved, a)
		}
	}

	return resolved, nil
}

func resolveContainerImage(container v1.Container) (string, string) {
	image := container.Image
	return image, image
}

func resolveContainerImageFromStatus(containerStatus v1.ContainerStatus) (string, string) {
	image := containerStatus.Image
	resolvedImage := containerStatus.ImageID
	if strings.HasPrefix(resolvedImage, dockerPullablePrefix) {
		resolvedImage = strings.TrimPrefix(resolvedImage, dockerPullablePrefix)
	}

	// stopped pods may not include the resolved image
	if len(resolvedImage) == 0 {
		resolvedImage = containerStatus.Image
	}

	return image, resolvedImage
}

func newPodImageAsset(image string, resolvedImage string) (*asset.Asset, error) {
	ccresolver := container_registry.NewContainerRegistryResolver()

	ref, err := name.ParseReference(resolvedImage, name.WeakValidation)
	if err != nil {
		return nil, err
	}
	a, err := ccresolver.GetImage(ref, nil)
	if err != nil {
		return nil, err
	}

	// parse image name to extract tags
	tagName := ""
	if len(image) > 0 {
		tag, err := name.NewTag(image, name.WeakValidation)
		if err == nil {
			tagName = tag.TagStr()
		}
	}
	if a.Labels == nil {
		a.Labels = map[string]string{}
	}
	a.Labels["docker.io/tags"] = tagName
	return a, nil
}
