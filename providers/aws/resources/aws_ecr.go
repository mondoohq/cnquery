// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/aws/aws-sdk-go-v2/service/ecrpublic"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/cnquery/providers/aws/connection"

	"go.mondoo.com/cnquery/types"
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

func (a *mqlAwsEcr) images() ([]interface{}, error) {
	obj, err := a.MqlRuntime.CreateResource(a.MqlRuntime, "aws.ecr", map[string]*llx.RawData{})
	if err != nil {
		return nil, err
	}
	ecr := obj.(*mqlAwsEcr)
	res := []interface{}{}

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

func (a *mqlAwsEcr) privateRepositories() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(a.getPrivateRepositories(conn), 5)
	poolOfJobs.Run()

	// check for errors
	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	// get all the results
	for i := range poolOfJobs.Jobs {
		res = append(res, poolOfJobs.Jobs[i].Result.([]interface{})...)
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
			res := []interface{}{}

			repoResp, err := svc.DescribeRepositories(ctx, &ecr.DescribeRepositoriesInput{})
			if err != nil {
				if Is400AccessDeniedError(err) {
					log.Warn().Str("region", region).Msg("error accessing region for AWS API")
					return res, nil
				}
				return nil, err
			}
			for i := range repoResp.Repositories {
				r := repoResp.Repositories[i]
				mqlRepoResource, err := a.MqlRuntime.CreateResource(a.MqlRuntime, "aws.ecr.repository",
					map[string]*llx.RawData{
						"arn":        llx.StringData(toString(r.RepositoryArn)),
						"name":       llx.StringData(toString(r.RepositoryName)),
						"uri":        llx.StringData(toString(r.RepositoryUri)),
						"registryId": llx.StringData(toString(r.RegistryId)),
						"public":     llx.BoolData(false),
						"region":     llx.StringData(region),
					})
				if err != nil {
					return nil, err
				}
				res = append(res, mqlRepoResource)
			}

			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func (a *mqlAwsEcrRepository) images() ([]interface{}, error) {
	name := a.Name.Data
	region := a.Region.Data
	public := a.Public.Data
	uri := a.Uri.Data
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	ctx := context.Background()
	mqlres := []interface{}{}
	if public {
		svc := conn.EcrPublic(region)
		res, err := svc.DescribeImages(ctx, &ecrpublic.DescribeImagesInput{RepositoryName: &name})
		if err != nil {
			if Is400AccessDeniedError(err) {
				log.Warn().Str("region", region).Msg("error accessing region for AWS API")
				return nil, nil
			}
			return nil, err
		}

		for i := range res.ImageDetails {
			image := res.ImageDetails[i]
			tags := []interface{}{}
			for i := range image.ImageTags {
				tags = append(tags, image.ImageTags[i])
			}
			mqlImage, err := a.MqlRuntime.CreateResource(a.MqlRuntime, "aws.ecr.image",
				map[string]*llx.RawData{
					"digest":     llx.StringData(toString(image.ImageDigest)),
					"mediaType":  llx.StringData(toString(image.ImageManifestMediaType)),
					"tags":       llx.ArrayData(tags, types.String),
					"registryId": llx.StringData(toString(image.RegistryId)),
					"repoName":   llx.StringData(name),
					"region":     llx.StringData(region),
					"arn":        llx.StringData(ecrImageArn(ImageInfo{Region: region, RegistryId: toString(image.RegistryId), RepoName: name, Digest: toString(image.ImageDigest)})),
					"uri":        llx.StringData(uri),
				})
			if err != nil {
				return nil, err
			}
			mqlres = append(mqlres, mqlImage)
		}
	} else {
		svc := conn.Ecr(region)
		res, err := svc.DescribeImages(ctx, &ecr.DescribeImagesInput{RepositoryName: &name})
		if err != nil {
			if Is400AccessDeniedError(err) {
				log.Warn().Str("region", region).Msg("error accessing region for AWS API")
				return nil, nil
			}
			return nil, err
		}
		for i := range res.ImageDetails {
			image := res.ImageDetails[i]
			tags := []interface{}{}
			for i := range image.ImageTags {
				tags = append(tags, image.ImageTags[i])
			}
			mqlImage, err := a.MqlRuntime.CreateResource(a.MqlRuntime, "aws.ecr.image",
				map[string]*llx.RawData{
					"digest":     llx.StringData(toString(image.ImageDigest)),
					"mediaType":  llx.StringData(toString(image.ImageManifestMediaType)),
					"tags":       llx.ArrayData(tags, types.String),
					"registryId": llx.StringData(toString(image.RegistryId)),
					"repoName":   llx.StringData(name),
					"region":     llx.StringData(region),
					"arn":        llx.StringData(ecrImageArn(ImageInfo{Region: region, RegistryId: toString(image.RegistryId), RepoName: name, Digest: toString(image.ImageDigest)})),
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

	obj, err := runtime.CreateResource(runtime, "aws.ecr", map[string]*llx.RawData{})
	if err != nil {
		return nil, nil, err
	}
	ecr := obj.(*mqlAwsEcr)

	rawResources, err := ecr.images()
	if err != nil {
		return nil, nil, err
	}

	arnVal := args["arn"].Value.(string)
	for i := range rawResources {
		image := rawResources[i].(*mqlAwsEcrImage)
		if image.Arn.Data == arnVal {
			return args, image, nil
		}
	}
	return nil, nil, errors.New("ecr image does not exist")
}

func (a *mqlAwsEcr) publicRepositories() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	svc := conn.EcrPublic("us-east-1") // only supported for us-east-1
	res := []interface{}{}

	repoResp, err := svc.DescribeRepositories(context.TODO(), &ecrpublic.DescribeRepositoriesInput{RegistryId: aws.String(conn.AccountId())})
	if err != nil {
		return nil, err
	}
	for i := range repoResp.Repositories {
		r := repoResp.Repositories[i]
		mqlRepoResource, err := a.MqlRuntime.CreateResource(a.MqlRuntime, "aws.ecr.repository",
			map[string]*llx.RawData{
				"arn":        llx.StringData(toString(r.RepositoryArn)),
				"name":       llx.StringData(toString(r.RepositoryName)),
				"uri":        llx.StringData(toString(r.RepositoryUri)),
				"registryId": llx.StringData(toString(r.RegistryId)),
				"public":     llx.BoolData(true),
				"region":     llx.StringData("us-east-1"),
			})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlRepoResource)
	}

	return res, nil
}
