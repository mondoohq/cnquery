// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"strings"

	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/types"
	v1 "k8s.io/api/core/v1"
)

const DockerPullablePrefix = "docker-pullable://"

type ContainerImage struct {
	image         string
	resolvedImage string
	pullSecrets   []v1.Secret
}

func ResolveUniqueContainerImages(cs []v1.Container, ps []v1.Secret) map[string]ContainerImage {
	imagesSet := make(map[string]ContainerImage)
	for _, c := range cs {
		imagesSet[c.Image] = ContainerImage{image: c.Image, resolvedImage: c.Image, pullSecrets: ps}
	}
	return imagesSet
}

func ResolveUniqueContainerImagesFromStatus(cs []v1.ContainerStatus, ps []v1.Secret) map[string]ContainerImage {
	imagesSet := make(map[string]ContainerImage)
	for _, c := range cs {
		image, resolvedImage := ResolveContainerImageFromStatus(c)
		imagesSet[resolvedImage] = ContainerImage{image: image, resolvedImage: resolvedImage, pullSecrets: ps}
	}
	return imagesSet
}

func ResolveContainerImageFromStatus(containerStatus v1.ContainerStatus) (string, string) {
	image := containerStatus.Image
	resolvedImage := containerStatus.ImageID
	resolvedImage = strings.TrimPrefix(resolvedImage, DockerPullablePrefix)

	// stopped pods may not include the resolved image
	// pods with imagePullPolicy: Never do not have a proper ImageId value as it contains only the
	// sha but not the repository. If we use that value, it will cause issues later because we will
	// eventually try to pull an image by providing just the sha without a repo.
	if len(resolvedImage) == 0 || !strings.Contains(resolvedImage, "@") {
		resolvedImage = containerStatus.Image
	}

	return image, resolvedImage
}

// UniqueImagesForPod returns the unique container images for a pod. Images are compared based on their digest
// if that is available in the pod status. If there is no pod status set, the container image tag is used.
func UniqueImagesForPod(pod v1.Pod, runtime *plugin.Runtime) map[string]ContainerImage {
	imagesSet := make(map[string]ContainerImage)

	pullSecrets := make([]v1.Secret, 0, len(pod.Spec.ImagePullSecrets))
	for _, ps := range pod.Spec.ImagePullSecrets {
		s, err := GetSecret(runtime, pod.Namespace, ps.Name) // TODO: figure out if we want to do anything with the error here
		if err == nil {
			pullSecrets = append(pullSecrets, *s)
		}
	}

	// it is best to read the image from the container status since it is resolved
	// and more accurate, for static file scan we also need to fall-back to pure spec
	// since the status will not be set
	imagesSet = types.MergeMaps(imagesSet, ResolveUniqueContainerImagesFromStatus(pod.Status.InitContainerStatuses, pullSecrets))

	// fall-back to spec
	if len(pod.Spec.InitContainers) > 0 && len(pod.Status.InitContainerStatuses) == 0 {
		imagesSet = types.MergeMaps(imagesSet, ResolveUniqueContainerImages(pod.Spec.InitContainers, pullSecrets))
	}

	imagesSet = types.MergeMaps(imagesSet, ResolveUniqueContainerImagesFromStatus(pod.Status.ContainerStatuses, pullSecrets))

	// fall-back to spec
	if len(pod.Spec.Containers) > 0 && len(pod.Status.ContainerStatuses) == 0 {
		imagesSet = types.MergeMaps(imagesSet, ResolveUniqueContainerImages(pod.Spec.Containers, pullSecrets))
	}
	return imagesSet
}

func GetSecret(runtime *plugin.Runtime, namespace, name string) (*v1.Secret, error) {
	s, err := NewResource(runtime, "k8s.secret", map[string]*llx.RawData{
		"namespace": llx.StringData(namespace),
		"name":      llx.StringData(name),
	})
	if err != nil {
		return nil, err
	}

	return s.(*mqlK8sSecret).obj, nil
}
