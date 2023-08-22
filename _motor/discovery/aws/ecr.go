// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package aws

import (
	"fmt"
	"strings"

	"go.mondoo.com/cnquery/motor/asset"
	"go.mondoo.com/cnquery/motor/motorid/containerid"
	"go.mondoo.com/cnquery/motor/platform"
	"go.mondoo.com/cnquery/motor/providers"
)

func NewEcrDiscovery(m *MqlDiscovery, cfg *providers.Config, account string) (*EcrImages, error) {
	return &EcrImages{mqlDiscovery: m, providerConfig: cfg.Clone(), account: account}, nil
}

type EcrImages struct {
	profile        string
	mqlDiscovery   *MqlDiscovery
	providerConfig *providers.Config
	account        string
	PassInLabels   map[string]string
}

func (ecri *EcrImages) Name() string {
	return "AWS ECR Discover"
}

func (ecri *EcrImages) List() ([]*asset.Asset, error) {
	imageAssets, err := ecrImages(ecri.mqlDiscovery, ecri.account, ecri.providerConfig)
	if err != nil {
		return nil, err
	}
	assetsWithConnecion := []*asset.Asset{}
	for i := range imageAssets {
		assetsWithConnecion = append(assetsWithConnecion, ecri.addConnectionInfoToEcrAsset(imageAssets[i], ecri.profile))
	}
	return assetsWithConnecion, nil
}

func MondooImageRegistryID(id string) string {
	return "//platformid.api.mondoo.app/runtime/docker/registry/" + id
}

func (ecri *EcrImages) addConnectionInfoToEcrAsset(image *asset.Asset, profile string) *asset.Asset {
	a := image
	digest := a.Labels[DigestLabel]
	repoUrl := a.Labels[RepoUrlLabel]
	region := a.Labels[RegionLabel]

	a.PlatformIds = []string{containerid.MondooContainerImageID(digest)}
	a.Platform = &platform.Platform{
		Kind:    providers.Kind_KIND_CONTAINER_IMAGE,
		Runtime: providers.RUNTIME_AWS_ECR,
	}
	a.State = asset.State_STATE_ONLINE

	imageTags := []string{}
	for k, v := range a.Labels {
		if k != "tag" {
			continue
		}
		imageTags = append(imageTags, v)
		a.Connections = []*providers.Config{{
			Backend: providers.ProviderType_CONTAINER_REGISTRY,
			Host:    repoUrl + ":" + v,
			Options: map[string]string{
				"region":  region,
				"profile": profile,
			},
		}}
	}

	// store digest
	a.Labels[fmt.Sprintf("ecr.%s.amazonaws.com/digest", region)] = digest
	a.Labels[fmt.Sprintf("ecr.%s.amazonaws.com/tags", region)] = strings.Join(imageTags, ",")

	// store repo digest
	repoDigests := []string{repoUrl + "@" + digest}
	a.Labels[fmt.Sprintf("ecr.%s.amazonaws.com/repo-digests", region)] = strings.Join(repoDigests, ",")

	if len(ecri.PassInLabels) > 0 {
		for k, v := range ecri.PassInLabels {
			a.Labels[k] = v
		}
	}
	return a
}
