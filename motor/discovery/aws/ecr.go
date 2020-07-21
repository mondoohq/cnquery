package aws

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"go.mondoo.io/mondoo/motor/motorapi"
	"go.mondoo.io/mondoo/nexus/assets"
)

func NewEcrImages(cfg aws.Config) (*EcrImages, error) {
	return &EcrImages{config: cfg}, nil
}

type EcrImages struct {
	config aws.Config
}

var aws_ecr_registry_pattern = "https://%s.dkr.ecr.%s.amazonaws.com"

func (a *EcrImages) List() ([]*assets.Asset, error) {
	ctx := context.Background()
	svc := ecr.New(a.config)

	reqRepo := svc.DescribeRepositoriesRequest(&ecr.DescribeRepositoriesInput{})
	repoResp, err := reqRepo.Send(ctx)
	if err != nil {
		return nil, err
	}
	imgs := []*assets.Asset{}
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

			asset := &assets.Asset{
				ReferenceIDs: []string{MondooContainerImageID(digest)},
				// Name:         strings.Join(dImg.RepoTags, ","),
				Platform: &assets.Platform{
					Kind:    assets.Kind_KIND_CONTAINER_IMAGE,
					Runtime: "aws ecr",
				},
				Connections: []*motorapi.TransportConfig{
					&motorapi.TransportConfig{
						Backend: motorapi.TransportBackend_CONNECTION_DOCKER_REGISTRY,
						Host:    registryURL,
					},
				},
				State:  assets.State_STATE_ONLINE,
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
