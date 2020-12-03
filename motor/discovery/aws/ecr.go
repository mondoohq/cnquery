package aws

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"go.mondoo.io/mondoo/motor/asset"
	"go.mondoo.io/mondoo/motor/platform"
	"go.mondoo.io/mondoo/motor/transports"
)

func NewEcrImages(cfg aws.Config) (*EcrImages, error) {
	return &EcrImages{config: cfg}, nil
}

type EcrImages struct {
	config aws.Config
}

var aws_ecr_registry_pattern = "https://%s.dkr.ecr.%s.amazonaws.com"

func (k *EcrImages) Name() string {
	return "AWS ECR Discover"
}

func (a *EcrImages) List() ([]*asset.Asset, error) {
	ctx := context.Background()
	svc := ecr.New(a.config)

	reqRepo := svc.DescribeRepositoriesRequest(&ecr.DescribeRepositoriesInput{})
	repoResp, err := reqRepo.Send(ctx)
	if err != nil {
		return nil, err
	}
	imgs := []*asset.Asset{}
	for i := range repoResp.Repositories {
		repoName := repoResp.Repositories[i].RepositoryName
		reqImages := svc.DescribeImagesRequest(&ecr.DescribeImagesInput{
			RepositoryName: repoName,
		})
		imageResp, err := reqImages.Send(ctx)
		if err != nil {
			return nil, err
		}

		for i := range imageResp.ImageDetails {
			registryURL := fmt.Sprintf(aws_ecr_registry_pattern, *imageResp.ImageDetails[i].RegistryId, a.config.Region)
			repoURL := registryURL + "/" + *imageResp.ImageDetails[i].RepositoryName
			digest := *imageResp.ImageDetails[i].ImageDigest

			asset := &asset.Asset{
				PlatformIDs: []string{MondooContainerImageID(digest)},
				// Name:         strings.Join(dImg.RepoTags, ","),
				Platform: &platform.Platform{
					Kind:    transports.Kind_KIND_CONTAINER_IMAGE,
					Runtime: transports.RUNTIME_AWS_ECR,
				},
				Connections: []*transports.TransportConfig{
					&transports.TransportConfig{
						Backend: transports.TransportBackend_CONNECTION_CONTAINER_REGISTRY,
						Host:    registryURL,
					},
				},
				State:  asset.State_STATE_ONLINE,
				Labels: make(map[string]string),
			}

			// store digest
			asset.Labels["docker.io/digest"] = digest

			// store repo tags
			imageTags := []string{}
			for j := range imageResp.ImageDetails[i].ImageTags {
				imageTags = append(imageTags, repoURL+":"+imageResp.ImageDetails[i].ImageTags[j])
			}
			asset.Labels["docker.io/tags"] = strings.Join(imageTags, ",")

			// store repo digest
			repoDigests := []string{repoURL + "@" + digest}
			asset.Labels["docker.io/repo-digests"] = strings.Join(repoDigests, ",")

			imgs = append(imgs, asset)
		}
	}

	return imgs, nil
}

// combine with docker image MondooContainerImageID
func MondooContainerImageID(id string) string {
	id = strings.Replace(id, "sha256:", "", -1)
	return "//platformid.api.mondoo.app/runtime/docker/images/" + id
}
