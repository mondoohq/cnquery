// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/aws/aws-sdk-go-v2/service/ecrpublic"
	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v12/llx"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/cnquery/v12/providers/aws/connection"

	"go.mondoo.com/cnquery/v12/types"
)

func (a *mqlAwsEcr) id() (string, error) {
	return "aws.ecr", nil
}

func (a *mqlAwsEcrRepository) id() (string, error) {
	return a.Arn.Data, nil
}

func (a *mqlAwsEcrImage) id() (string, error) {
	id := a.RegistryId.Data
	sha := a.Digest.Data
	name := a.RepoName.Data
	return id + "/" + name + "/" + sha, nil
}

func (a *mqlAwsEcr) images() ([]any, error) {
	obj, err := CreateResource(a.MqlRuntime, ResourceAwsEcr, map[string]*llx.RawData{})
	if err != nil {
		return nil, err
	}
	ecr := obj.(*mqlAwsEcr)
	res := []any{}

	repos, err := ecr.publicRepositories()
	if err != nil {
		return nil, err
	}
	for i := range repos {
		images, err := repos[i].(*mqlAwsEcrRepository).images()
		if err != nil {
			return nil, err
		}
		res = append(res, images...)
	}
	pRepos, err := ecr.privateRepositories()
	if err != nil {
		return nil, err
	}
	for i := range pRepos {
		images, err := pRepos[i].(*mqlAwsEcrRepository).images()
		if err != nil {
			return nil, err
		}
		res = append(res, images...)
	}
	return res, nil
}

func (a *mqlAwsEcr) privateRepositories() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getPrivateRepositories(conn), 5)
	poolOfJobs.Run()

	// check for errors
	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	// get all the results
	for i := range poolOfJobs.Jobs {
		res = append(res, poolOfJobs.Jobs[i].Result.([]any)...)
	}

	return res, nil
}

func (a *mqlAwsEcr) getPrivateRepositories(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	ctx := context.Background()

	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}} // return the error
	}
	for ri := range regions {
		region := regions[ri]
		f := func() (jobpool.JobResult, error) {
			svc := conn.Ecr(region)
			res := []any{}

			paginator := ecr.NewDescribeRepositoriesPaginator(svc, &ecr.DescribeRepositoriesInput{})
			for paginator.HasMorePages() {
				repoResp, err := paginator.NextPage(ctx)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS API")
						return res, nil
					}
					return nil, err
				}
				for _, r := range repoResp.Repositories {
					imageScanOnPush := false
					if r.ImageScanningConfiguration != nil {
						imageScanOnPush = r.ImageScanningConfiguration.ScanOnPush
					}
					var encryptionType string
					if r.EncryptionConfiguration != nil {
						encryptionType = string(r.EncryptionConfiguration.EncryptionType)
					}
					mqlRepoResource, err := CreateResource(a.MqlRuntime, ResourceAwsEcrRepository,
						map[string]*llx.RawData{
							"arn":                llx.StringDataPtr(r.RepositoryArn),
							"name":               llx.StringDataPtr(r.RepositoryName),
							"uri":                llx.StringDataPtr(r.RepositoryUri),
							"registryId":         llx.StringDataPtr(r.RegistryId),
							"public":             llx.BoolData(false),
							"region":             llx.StringData(region),
							"imageScanOnPush":    llx.BoolData(imageScanOnPush),
							"imageTagMutability": llx.StringData(string(r.ImageTagMutability)),
							"encryptionType":     llx.StringData(encryptionType),
							"createdAt":          llx.TimeDataPtr(r.CreatedAt),
						})
					if err != nil {
						return nil, err
					}
					res = append(res, mqlRepoResource)
				}
			}

			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func (a *mqlAwsEcrRepository) scanningFrequency() (string, error) {
	if a.Public.Data {
		return "", nil
	}

	name := a.Name.Data
	region := a.Region.Data

	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	svc := conn.Ecr(region)
	ctx := context.Background()

	resp, err := svc.BatchGetRepositoryScanningConfiguration(ctx, &ecr.BatchGetRepositoryScanningConfigurationInput{
		RepositoryNames: []string{name},
	})
	if err != nil {
		if Is400AccessDeniedError(err) {
			return "", nil
		}
		return "", err
	}

	if len(resp.ScanningConfigurations) > 0 {
		// The API returns exactly one ScanningConfiguration per repository in the request.
		return string(resp.ScanningConfigurations[0].ScanFrequency), nil
	}

	return "", nil
}

func (a *mqlAwsEcrRepository) images() ([]any, error) {
	name := a.Name.Data
	region := a.Region.Data
	public := a.Public.Data
	uri := a.Uri.Data
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	ctx := context.Background()
	mqlres := []any{}
	if public {
		svc := conn.EcrPublic(region)
		paginator := ecrpublic.NewDescribeImagesPaginator(svc, &ecrpublic.DescribeImagesInput{RepositoryName: &name})
		for paginator.HasMorePages() {
			res, err := paginator.NextPage(ctx)
			if err != nil {
				if Is400AccessDeniedError(err) {
					log.Warn().Str("region", region).Msg("error accessing region for AWS API")
					return nil, nil
				}
				return nil, err
			}
			for _, image := range res.ImageDetails {
				if conn.Filters.Ecr.IsFilteredOutByTags(image.ImageTags) {
					log.Debug().Str("repository", name).Strs("tags", image.ImageTags).Msg("skipping ecr public image due to tag filters")
					continue
				}
				mqlImage, err := CreateResource(a.MqlRuntime, ResourceAwsEcrImage,
					map[string]*llx.RawData{
						"digest":     llx.StringDataPtr(image.ImageDigest),
						"mediaType":  llx.StringDataPtr(image.ImageManifestMediaType),
						"tags":       llx.ArrayData(toInterfaceArr(image.ImageTags), types.String),
						"registryId": llx.StringDataPtr(image.RegistryId),
						"repoName":   llx.StringData(name),
						"region":     llx.StringData(region),
						"arn":        llx.StringData(ecrImageArn(ImageInfo{Region: region, RegistryId: convert.ToValue(image.RegistryId), RepoName: name, Digest: convert.ToValue(image.ImageDigest)})),
						"uri":        llx.StringData(uri),
					})
				if err != nil {
					return nil, err
				}
				mqlres = append(mqlres, mqlImage)
			}
		}
		return mqlres, nil
	}

	// private
	svc := conn.Ecr(region)
	paginator := ecr.NewDescribeImagesPaginator(svc, &ecr.DescribeImagesInput{RepositoryName: &name})
	for paginator.HasMorePages() {
		res, err := paginator.NextPage(ctx)
		if err != nil {
			if Is400AccessDeniedError(err) {
				log.Warn().Str("region", region).Msg("error accessing region for AWS API")
				return nil, nil
			}
			return nil, err
		}
		for _, image := range res.ImageDetails {
			if conn.Filters.Ecr.IsFilteredOutByTags(image.ImageTags) {
				log.Debug().Str("repository", name).Strs("tags", image.ImageTags).Msg("skipping ecr private image due to tag filters")
				continue
			}
			mqlImage, err := CreateResource(a.MqlRuntime, ResourceAwsEcrImage,
				map[string]*llx.RawData{
					"arn":                  llx.StringData(ecrImageArn(ImageInfo{Region: region, RegistryId: convert.ToValue(image.RegistryId), RepoName: name, Digest: convert.ToValue(image.ImageDigest)})),
					"digest":               llx.StringDataPtr(image.ImageDigest),
					"lastRecordedPullTime": llx.TimeDataPtr(image.LastRecordedPullTime),
					"mediaType":            llx.StringDataPtr(image.ImageManifestMediaType),
					"pushedAt":             llx.TimeDataPtr(image.ImagePushedAt),
					"region":               llx.StringData(region),
					"registryId":           llx.StringDataPtr(image.RegistryId),
					"repoName":             llx.StringData(name),
					"sizeInBytes":          llx.IntDataPtr(image.ImageSizeInBytes),
					"tags":                 llx.ArrayData(toInterfaceArr(image.ImageTags), types.String),
					"uri":                  llx.StringData(uri),
				})
			if err != nil {
				return nil, err
			}
			mqlres = append(mqlres, mqlImage)
		}
	}
	return mqlres, nil
}

type ImageInfo struct {
	Region     string
	RepoName   string
	Digest     string
	RegistryId string
}

func ecrImageArn(i ImageInfo) string {
	return fmt.Sprintf("arn:aws:ecr:%s:%s:image/%s/%s", i.Region, i.RegistryId, i.RepoName, i.Digest)
}

func EcrImageName(i ImageInfo) string {
	return i.RepoName + "@" + i.Digest
}

func initAwsEcrImage(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}

	if len(args) == 0 {
		if ids := getAssetIdentifier(runtime); ids != nil {
			args["name"] = llx.StringData(ids.name)
			args["arn"] = llx.StringData(ids.arn)
		}
	}

	if args["arn"] == nil {
		return nil, nil, errors.New("arn required to fetch ecr image")
	}

	obj, err := CreateResource(runtime, "aws.ecr", map[string]*llx.RawData{})
	if err != nil {
		return nil, nil, err
	}
	ecr := obj.(*mqlAwsEcr)

	rawResources := ecr.GetImages()
	if rawResources.Error != nil {
		return nil, nil, rawResources.Error
	}
	arnVal := args["arn"].Value.(string)
	for _, rawResource := range rawResources.Data {
		image := rawResource.(*mqlAwsEcrImage)
		if image.Arn.Data == arnVal {
			return args, image, nil
		}
	}
	return nil, nil, errors.New("ecr image does not exist")
}

func (a *mqlAwsEcr) publicRepositories() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	svc := conn.EcrPublic("us-east-1") // only supported for us-east-1
	res := []any{}

	paginator := ecrpublic.NewDescribeRepositoriesPaginator(svc, &ecrpublic.DescribeRepositoriesInput{RegistryId: aws.String(conn.AccountId())})
	for paginator.HasMorePages() {
		repoResp, err := paginator.NextPage(context.TODO())
		if err != nil {
			return nil, err
		}

		for _, r := range repoResp.Repositories {
			mqlRepoResource, err := CreateResource(a.MqlRuntime, ResourceAwsEcrRepository,
				map[string]*llx.RawData{
					"arn":                llx.StringDataPtr(r.RepositoryArn),
					"name":               llx.StringDataPtr(r.RepositoryName),
					"uri":                llx.StringDataPtr(r.RepositoryUri),
					"registryId":         llx.StringDataPtr(r.RegistryId),
					"public":             llx.BoolData(true),
					"region":             llx.StringData("us-east-1"),
					"imageScanOnPush":    llx.BoolData(false),
					"imageTagMutability": llx.StringData("IMMUTABLE"),
					"encryptionType":     llx.StringData("AES256"),
					"createdAt":          llx.TimeDataPtr(r.CreatedAt),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlRepoResource)
		}
	}

	return res, nil
}
