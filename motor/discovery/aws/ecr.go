package aws

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/lumi/library/jobpool"
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
	instances := []*asset.Asset{}
	poolOfJobs := jobpool.CreatePool(a.getRepositories(), 5)
	poolOfJobs.Run()

	// check for errors
	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	// get all the results
	for i := range poolOfJobs.Jobs {
		instances = append(instances, poolOfJobs.Jobs[i].Result.([]*asset.Asset)...)
	}

	return instances, nil
}

func (a *EcrImages) getRegions() ([]string, error) {
	regions := []string{}

	ec2svc := ec2.NewFromConfig(a.config)
	ctx := context.Background()

	res, err := ec2svc.DescribeRegions(ctx, &ec2.DescribeRegionsInput{})
	if err != nil {
		return regions, nil
	}
	for _, region := range res.Regions {
		regions = append(regions, *region.RegionName)
	}
	return regions, nil
}

func (a *EcrImages) getRepositories() []*jobpool.Job {
	var tasks = make([]*jobpool.Job, 0)
	ctx := context.Background()
	// user did not include a region filter, fetch em all
	regions, err := a.getRegions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}
	log.Debug().Msgf("regions being called for ecr images are: %v", regions)
	for ri := range regions {
		region := regions[ri]
		f := func() (jobpool.JobResult, error) {

			clonedConfig := a.config.Copy()
			clonedConfig.Region = region
			svc := ecr.NewFromConfig(clonedConfig)
			imgs := []*asset.Asset{}

			repoResp, err := svc.DescribeRepositories(ctx, &ecr.DescribeRepositoriesInput{})
			if err != nil {
				return imgs, nil
			}
			for i := range repoResp.Repositories {
				repoName := repoResp.Repositories[i].RepositoryName
				imageResp, err := svc.DescribeImages(ctx, &ecr.DescribeImagesInput{
					RepositoryName: repoName,
				})
				if err != nil {
					return imgs, nil
				}
				for i := range imageResp.ImageDetails {
					registryURL := fmt.Sprintf(aws_ecr_registry_pattern, *imageResp.ImageDetails[i].RegistryId, a.config.Region)
					repoURL := registryURL + "/" + *imageResp.ImageDetails[i].RepositoryName
					digest := *imageResp.ImageDetails[i].ImageDigest

					asset := &asset.Asset{
						PlatformIds: []string{MondooContainerImageID(digest)},
						// Name:         strings.Join(dImg.RepoTags, ","),
						Platform: &platform.Platform{
							Kind:    transports.Kind_KIND_CONTAINER_IMAGE,
							Runtime: transports.RUNTIME_AWS_ECR,
						},
						Connections: []*transports.TransportConfig{
							{
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
			return jobpool.JobResult(imgs), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

// combine with docker image MondooContainerImageID
func MondooContainerImageID(id string) string {
	id = strings.Replace(id, "sha256:", "", -1)
	return "//platformid.api.mondoo.app/runtime/docker/images/" + id
}
