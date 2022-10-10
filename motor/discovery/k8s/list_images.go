package k8s

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/docker/cli/cli/config/configfile"
	"github.com/google/go-containerregistry/pkg/name"

	"go.mondoo.com/cnquery/motor/discovery/container_registry"
	"go.mondoo.com/cnquery/motor/vault"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/motor/asset"
	"go.mondoo.com/cnquery/motor/providers/k8s"
	v1 "k8s.io/api/core/v1"
)

const dockerPullablePrefix = "docker-pullable://"

// ListPodImages lits all container images for the pods in the cluster. Only unique container images are returned.
// Uniqueness is determined based on the container digests.
func ListPodImages(p k8s.KubernetesProvider, namespaceFilter []string, od *k8s.PlatformIdOwnershipDirectory) ([]*asset.Asset, error) {
	namespaces, err := p.Namespaces()
	if err != nil {
		return nil, errors.Wrap(err, "could not list kubernetes namespaces")
	}

	// Grab the unique container images in the cluster.
	runningImages := make(map[string]containerImage)
	credsStore := NewCredsStore(p)
	for i := range namespaces {
		namespace := namespaces[i]
		if !isIncluded(namespace.Name, namespaceFilter) {
			log.Debug().Str("namespace", namespace.Name).Strs("filter", namespaceFilter).Msg("namespace not included")
			continue
		}

		pods, err := p.Pods(namespace)
		if err != nil {
			return nil, errors.Wrap(err, "failed to list pods")
		}

		for j := range pods {
			od.Add(pods[j])
			podImages := uniqueImagesForPod(*pods[j], credsStore)
			runningImages = mergeMaps(runningImages, podImages)
		}
	}

	// Convert the container images to assets.
	assets := make(map[string]*asset.Asset)
	for _, i := range runningImages {
		a, err := newPodImageAsset(i)
		if err != nil {
			log.Error().Err(err).Msg("failed to convert container image to asset")
			continue
		}

		// It is still possible to have unique images at this point. There might be
		// multiple image tags that actually point to the same digest. If we are scanning
		// a manifest, where there is no container status, we can only know that the 2 images
		// are identical after we resolve them with the container registry.
		assets[a.Labels["docker.io/digest"]] = a
		log.Debug().Str("name", a.Name).Str("image", a.Connections[0].Host).Msg("resolved pod")
	}

	return mapValuesToSlice(assets), nil
}

// uniqueImagesForPod returns the unique container images for a pod. Images are compared based on their digest
// if that is available in the pod status. If there is no pod status set, the container image tag is used.
func uniqueImagesForPod(pod v1.Pod, credsStore *credsStore) map[string]containerImage {
	imagesSet := make(map[string]containerImage)

	pullSecrets := make([]v1.Secret, 0, len(pod.Spec.ImagePullSecrets))
	for _, ps := range pod.Spec.ImagePullSecrets {
		s, err := credsStore.Get(pod.Namespace, ps.Name) // TODO: figure out if we want to do anything with the error here
		if err == nil {
			pullSecrets = append(pullSecrets, *s)
		}
	}

	// it is best to read the image from the container status since it is resolved
	// and more accurate, for static file scan we also need to fall-back to pure spec
	// since the status will not be set
	imagesSet = mergeMaps(imagesSet, resolveUniqueContainerImagesFromStatus(pod.Status.InitContainerStatuses, pullSecrets))

	// fall-back to spec
	if len(pod.Spec.InitContainers) > 0 && len(pod.Status.InitContainerStatuses) == 0 {
		imagesSet = mergeMaps(imagesSet, resolveUniqueContainerImages(pod.Spec.InitContainers, pullSecrets))
	}

	imagesSet = mergeMaps(imagesSet, resolveUniqueContainerImagesFromStatus(pod.Status.ContainerStatuses, pullSecrets))

	// fall-back to spec
	if len(pod.Spec.Containers) > 0 && len(pod.Status.ContainerStatuses) == 0 {
		imagesSet = mergeMaps(imagesSet, resolveUniqueContainerImages(pod.Spec.Containers, pullSecrets))
	}
	return imagesSet
}

type containerImage struct {
	image         string
	resolvedImage string
	pullSecrets   []v1.Secret
}

func resolveUniqueContainerImages(cs []v1.Container, ps []v1.Secret) map[string]containerImage {
	imagesSet := make(map[string]containerImage)
	for _, c := range cs {
		imagesSet[c.Image] = containerImage{image: c.Image, resolvedImage: c.Image, pullSecrets: ps}
	}
	return imagesSet
}

func resolveUniqueContainerImagesFromStatus(cs []v1.ContainerStatus, ps []v1.Secret) map[string]containerImage {
	imagesSet := make(map[string]containerImage)
	for _, c := range cs {
		image, resolvedImage := resolveContainerImageFromStatus(c, ps)
		imagesSet[resolvedImage] = containerImage{image: image, resolvedImage: resolvedImage, pullSecrets: ps}
	}
	return imagesSet
}

func resolveContainerImageFromStatus(containerStatus v1.ContainerStatus, ps []v1.Secret) (string, string) {
	image := containerStatus.Image
	resolvedImage := containerStatus.ImageID
	if strings.HasPrefix(resolvedImage, dockerPullablePrefix) {
		resolvedImage = strings.TrimPrefix(resolvedImage, dockerPullablePrefix)
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

func newPodImageAsset(i containerImage) (*asset.Asset, error) {
	ccresolver := container_registry.NewContainerRegistryResolver()

	ref, err := name.ParseReference(i.resolvedImage, name.WeakValidation)
	if err != nil {
		return nil, err
	}

	a, err := ccresolver.GetImage(ref, nil)
	// If there was an error getting the image, try to resolve it using image pull secrets.
	// It might be that the container is coming from a private repo.
	if err != nil {
		for _, secret := range i.pullSecrets {
			if cfg, ok := secret.Data[v1.DockerConfigJsonKey]; ok {
				creds, err := toCredential(cfg)
				if err != nil {
					continue
				}

				a, err = ccresolver.GetImage(ref, creds)
				if err == nil {
					break
				}
			}
		}
	}

	// If at this point we still have no asset it means that neither public scan worked, nor
	// a scan using pull secrets.
	if a == nil {
		return nil, fmt.Errorf("could not resolve image %s. %v", i.resolvedImage, err)
	}

	// parse image name to extract tags
	tagName := ""
	if len(i.image) > 0 {
		tag, err := name.NewTag(i.image, name.WeakValidation)
		if err == nil {
			tagName = tag.Name()
		}
	}
	if a.Labels == nil {
		a.Labels = map[string]string{}
	}
	a.Labels["docker.io/tags"] = tagName
	return a, nil
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

// mapValuesToSlice returns a slice with the values of the map
func mapValuesToSlice[K comparable, V any](m map[K]V) []V {
	var slice []V
	for _, v := range m {
		slice = append(slice, v)
	}
	return slice
}

// mergeMaps merges 2 maps. If there are duplicate keys the values from m2 will override
// the values from m1.
func mergeMaps[K comparable, V any](m1 map[K]V, m2 map[K]V) map[K]V {
	for k, v := range m2 {
		m1[k] = v
	}
	return m1
}

func toCredential(cfg []byte) ([]*vault.Credential, error) {
	cf := configfile.ConfigFile{}
	if err := json.Unmarshal(cfg, &cf); err != nil {
		return nil, err
	}

	var creds []*vault.Credential
	for _, v := range cf.AuthConfigs {
		c := &vault.Credential{
			User: v.Username,
		}

		if v.Password != "" {
			c.Type = vault.CredentialType_password
			c.Secret = []byte(v.Password)
		} else if v.RegistryToken != "" {
			c.Type = vault.CredentialType_bearer
			c.Secret = []byte(v.RegistryToken)
		}
		creds = append(creds, c)
	}
	return creds, nil
}
