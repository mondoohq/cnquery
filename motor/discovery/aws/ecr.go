package aws

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/aws/aws-sdk-go-v2/service/ecr/types"
	"github.com/aws/aws-sdk-go-v2/service/ecrpublic"
	publicecrtypes "github.com/aws/aws-sdk-go-v2/service/ecrpublic/types"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/motor/asset"
	"go.mondoo.com/cnquery/motor/motorid/containerid"
	"go.mondoo.com/cnquery/motor/platform"
	"go.mondoo.com/cnquery/motor/providers"
	"go.mondoo.com/cnquery/resources/library/jobpool"
)

func NewEcrDiscovery(cfg aws.Config) (*EcrImages, error) {
	return &EcrImages{config: cfg}, nil
}

type EcrImages struct {
	config  aws.Config
	profile string
}

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
	tasks := make([]*jobpool.Job, 0)
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
			assets := []*asset.Asset{}

			repoResp, err := svc.DescribeRepositories(ctx, &ecr.DescribeRepositoriesInput{})
			if err != nil {
				return assets, nil
			}
			for i := range repoResp.Repositories {
				assets = append(assets, ecrRepoToAsset(repoResp.Repositories[i], region, a.profile))
				repoName := repoResp.Repositories[i].RepositoryName
				repoUrl := *repoResp.Repositories[i].RepositoryUri
				imageResp, err := svc.DescribeImages(ctx, &ecr.DescribeImagesInput{
					RepositoryName: repoName,
				})
				if err != nil {
					log.Error().Err(err).Str("repo", repoUrl).Msg("cannot describe images")
					continue
				}
				for i := range imageResp.ImageDetails {
					assets = append(assets, ecrImageToAsset(imageResp.ImageDetails[i], region, repoUrl, a.profile))
				}
			}
			publicEcrSvc := ecrpublic.NewFromConfig(clonedConfig)

			publicRepoResp, err := publicEcrSvc.DescribeRepositories(ctx, &ecrpublic.DescribeRepositoriesInput{})
			if err != nil {
				return assets, nil
			}
			for i := range publicRepoResp.Repositories {
				assets = append(assets, publicEcrRepoToAsset(publicRepoResp.Repositories[i], region, a.profile))
				repoName := publicRepoResp.Repositories[i].RepositoryName
				repoUrl := publicRepoResp.Repositories[i].RepositoryUri
				imageResp, err := publicEcrSvc.DescribeImages(ctx, &ecrpublic.DescribeImagesInput{
					RepositoryName: repoName,
				})
				if err != nil {
					log.Error().Err(err).Str("repo", *repoUrl).Msg("cannot describe images")
					continue
				}
				for i := range imageResp.ImageDetails {
					assets = append(assets, publicEcrImageToAsset(imageResp.ImageDetails[i], region, *repoUrl, a.profile))
				}
			}
			return jobpool.JobResult(assets), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func MondooImageRegistryID(id string) string {
	return "//platformid.api.mondoo.app/runtime/docker/registry/" + id
}

func publicEcrRepoToAsset(repo publicecrtypes.Repository, region string, profile string) *asset.Asset {
	asset := &asset.Asset{
		PlatformIds: []string{MondooImageRegistryID(*repo.RegistryId)},
		Name:        *repo.RepositoryName,
		Platform: &platform.Platform{
			Kind:    providers.Kind_KIND_CONTAINER_IMAGE,
			Runtime: providers.RUNTIME_AWS_ECR,
		},
		Connections: []*providers.Config{
			{
				Backend: providers.ProviderType_CONTAINER_REGISTRY,
				Host:    *repo.RepositoryUri,
				Options: map[string]string{
					"region":  region,
					"profile": profile,
				},
			},
		},
		State:  asset.State_STATE_ONLINE,
		Labels: make(map[string]string),
	}
	return asset
}

func ecrRepoToAsset(repo types.Repository, region string, profile string) *asset.Asset {
	asset := &asset.Asset{
		PlatformIds: []string{MondooImageRegistryID(*repo.RegistryId)},
		Name:        *repo.RepositoryName,
		Platform: &platform.Platform{
			Kind:    providers.Kind_KIND_CONTAINER_IMAGE,
			Runtime: providers.RUNTIME_AWS_ECR,
		},
		Connections: []*providers.Config{
			{
				Backend: providers.ProviderType_CONTAINER_REGISTRY,
				Host:    *repo.RepositoryUri,
				Options: map[string]string{
					"region":  region,
					"profile": profile,
				},
			},
		},
		State:  asset.State_STATE_ONLINE,
		Labels: make(map[string]string),
	}
	return asset
}

type EcrImageInfo struct {
	digest      string
	tags        []string
	repoDigests []string
	repoUrl     string
	region      string
	repoName    string
}

func ecrToAsset(i EcrImageInfo, profile string) *asset.Asset {
	a := &asset.Asset{
		PlatformIds: []string{containerid.MondooContainerImageID(i.digest)},
		Name:        i.repoName + "@" + i.digest,
		Platform: &platform.Platform{
			Kind:    providers.Kind_KIND_CONTAINER_IMAGE,
			Runtime: providers.RUNTIME_AWS_ECR,
		},
		Connections: []*providers.Config{},
		State:       asset.State_STATE_ONLINE,
		Labels:      make(map[string]string),
	}
	for _, tag := range i.tags {
		a.Connections = append(a.Connections, &providers.Config{
			Backend: providers.ProviderType_CONTAINER_REGISTRY,
			Host:    i.repoUrl + ":" + tag,
			Options: map[string]string{
				"region":  i.region,
				"profile": profile,
			},
		})
	}
	// store digest
	a.Labels[fmt.Sprintf("ecr.%s.amazonaws.com/digest", i.region)] = i.digest

	// store repo tags
	imageTags := []string{}
	for j := range i.tags {
		imageTags = append(imageTags, i.repoUrl+":"+i.tags[j])
	}
	a.Labels[fmt.Sprintf("ecr.%s.amazonaws.com/tags", i.region)] = strings.Join(imageTags, ",")

	// store repo digest
	repoDigests := []string{i.repoUrl + "@" + i.digest}
	a.Labels[fmt.Sprintf("ecr.%s.amazonaws.com/repo-digests", i.region)] = strings.Join(repoDigests, ",")
	return a
}

func publicEcrImageToAsset(image publicecrtypes.ImageDetail, region string, repoUrl string, profile string) *asset.Asset {
	return ecrToAsset(EcrImageInfo{
		digest:      *image.ImageDigest,
		tags:        image.ImageTags,
		repoDigests: []string{repoUrl + "@" + *image.ImageDigest},
		repoUrl:     repoUrl,
		region:      region,
		repoName:    *image.RepositoryName,
	}, profile)
}

func ecrImageToAsset(image types.ImageDetail, region string, repoUrl string, profile string) *asset.Asset {
	return ecrToAsset(EcrImageInfo{
		digest:      *image.ImageDigest,
		tags:        image.ImageTags,
		repoDigests: []string{repoUrl + "@" + *image.ImageDigest},
		repoUrl:     repoUrl,
		region:      region,
		repoName:    *image.RepositoryName,
	}, profile)
}
