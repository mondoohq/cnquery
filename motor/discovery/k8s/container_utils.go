package k8s

import (
	"strings"

	v1 "k8s.io/api/core/v1"
)

const DockerPullablePrefix = "docker-pullable://"

type containerImage struct {
	image         string
	resolvedImage string
	pullSecrets   []v1.Secret
}

func ResolveUniqueContainerImages(cs []v1.Container, ps []v1.Secret) map[string]containerImage {
	imagesSet := make(map[string]containerImage)
	for _, c := range cs {
		imagesSet[c.Image] = containerImage{image: c.Image, resolvedImage: c.Image, pullSecrets: ps}
	}
	return imagesSet
}

func ResolveUniqueContainerImagesFromStatus(cs []v1.ContainerStatus, ps []v1.Secret) map[string]containerImage {
	imagesSet := make(map[string]containerImage)
	for _, c := range cs {
		image, resolvedImage := ResolveContainerImageFromStatus(c)
		imagesSet[resolvedImage] = containerImage{image: image, resolvedImage: resolvedImage, pullSecrets: ps}
	}
	return imagesSet
}

func ResolveContainerImageFromStatus(containerStatus v1.ContainerStatus) (string, string) {
	image := containerStatus.Image
	resolvedImage := containerStatus.ImageID
	if strings.HasPrefix(resolvedImage, DockerPullablePrefix) {
		resolvedImage = strings.TrimPrefix(resolvedImage, DockerPullablePrefix)
	}

	// stopped pods may not include the resolved image
	// pods with imagePullPolicy: Never do not have a proper ImageId value as it contains only the
	// sha but not the repository. If we use that value, it will cause issues later because we will
	// eventually try to pull an image by providing just the sha without a repo.
	if len(resolvedImage) == 0 || !strings.Contains(resolvedImage, "@") {
		resolvedImage = containerStatus.Image
	}

	return image, resolvedImage
}
