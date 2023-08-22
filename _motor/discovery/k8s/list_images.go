// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package k8s

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/docker/cli/cli/config/configfile"
	"github.com/google/go-containerregistry/pkg/name"

	"go.mondoo.com/cnquery/motor/discovery/container_registry"
	"go.mondoo.com/cnquery/motor/vault"
	"go.mondoo.com/cnquery/types"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/motor/asset"
	"go.mondoo.com/cnquery/motor/providers/k8s"
	v1 "k8s.io/api/core/v1"
)

// ListPodImages lits all container images for the pods in the cluster. Only unique container images are returned.
// Uniqueness is determined based on the container digests.
func ListPodImages(p k8s.KubernetesProvider, nsFilter NamespaceFilterOpts, od *k8s.PlatformIdOwnershipDirectory) ([]*asset.Asset, error) {
	namespaces, err := p.Namespaces()
	if err != nil {
		return nil, errors.Wrap(err, "could not list kubernetes namespaces")
	}

	// Grab the unique container images in the cluster.
	runningImages := make(map[string]ContainerImage)
	credsStore := NewCredsStore(p)
	for i := range namespaces {
		namespace := namespaces[i]
		skip, err := skipNamespace(namespace, nsFilter)
		if err != nil {
			log.Error().Err(err).Str("namespace", namespace.Name).Msg("error checking whether namespace should be included or excluded")
			return nil, err
		}
		if skip {
			log.Debug().Str("namespace", namespace.Name).Msg("namespace not included")
			continue
		}

		pods, err := p.Pods(namespace)
		if err != nil {
			return nil, errors.Wrap(err, "failed to list pods")
		}

		for j := range pods {
			od.Add(pods[j])
			podImages := UniqueImagesForPod(*pods[j], credsStore)
			runningImages = types.MergeMaps(runningImages, podImages)
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

	return types.MapValuesToSlice(assets), nil
}

func newPodImageAsset(i ContainerImage) (*asset.Asset, error) {
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
