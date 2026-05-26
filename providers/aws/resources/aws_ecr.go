// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	ecrtypes "github.com/aws/aws-sdk-go-v2/service/ecr/types"
	"github.com/aws/aws-sdk-go-v2/service/ecrpublic"
	ecrpublic_types "github.com/aws/aws-sdk-go-v2/service/ecrpublic/types"
	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/mql/v13/providers/aws/connection"

	"go.mondoo.com/mql/v13/types"
)

// ecrLifecyclePolicyDoc represents the parsed JSON lifecycle policy document.
// The AWS SDK returns this as a raw JSON string with no typed structs.
type ecrLifecyclePolicyDoc struct {
	Rules []ecrLifecyclePolicyRule `json:"rules"`
}

type ecrLifecyclePolicyRule struct {
	RulePriority int                         `json:"rulePriority"`
	Description  string                      `json:"description"`
	Selection    ecrLifecyclePolicySelection `json:"selection"`
	Action       ecrLifecyclePolicyAction    `json:"action"`
}

type ecrLifecyclePolicySelection struct {
	TagStatus      string   `json:"tagStatus"`
	TagPatternList []string `json:"tagPatternList"`
	TagPrefixList  []string `json:"tagPrefixList"`
	CountType      string   `json:"countType"`
	CountUnit      string   `json:"countUnit"`
	CountNumber    int      `json:"countNumber"`
}

type ecrLifecyclePolicyAction struct {
	Type string `json:"type"`
}

func (a *mqlAwsEcr) id() (string, error) {
	return "aws.ecr", nil
}

func (a *mqlAwsEcr) replicationConfiguration() (any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	svc := conn.Ecr("")
	ctx := context.Background()

	resp, err := svc.DescribeRegistry(ctx, &ecr.DescribeRegistryInput{})
	if err != nil {
		return nil, err
	}
	if resp.ReplicationConfiguration == nil {
		return nil, nil
	}
	return convert.JsonToDict(resp.ReplicationConfiguration)
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

func (a *mqlAwsEcrLifecyclePolicy) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAwsEcrLifecyclePolicyRule) id() (string, error) {
	return a.Id.Data, nil
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

	if conn.Filters.Ecr.Scope == connection.EcrScopePublic {
		return []any{}, nil
	}

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

			req := &ecr.DescribeRepositoriesInput{}
			if len(conn.Filters.Ecr.PrivateRepositoryNames) > 0 {
				req.RepositoryNames = conn.Filters.Ecr.PrivateRepositoryNames
			}

			paginator := ecr.NewDescribeRepositoriesPaginator(svc, req)
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
				mqlImage.(*mqlAwsEcrImage).cachePublic = true
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

func newMqlEcrLifecyclePolicyRule(runtime *plugin.Runtime, repoArn string, rule ecrLifecyclePolicyRule) (*mqlAwsEcrLifecyclePolicyRule, error) {
	ruleId := fmt.Sprintf("%s/lifecyclePolicy/rule/%d", repoArn, rule.RulePriority)

	tagPatternList := rule.Selection.TagPatternList
	if tagPatternList == nil {
		tagPatternList = []string{}
	}
	tagPrefixList := rule.Selection.TagPrefixList
	if tagPrefixList == nil {
		tagPrefixList = []string{}
	}

	resource, err := CreateResource(runtime, "aws.ecr.lifecyclePolicy.rule",
		map[string]*llx.RawData{
			"__id":           llx.StringData(ruleId),
			"id":             llx.StringData(ruleId),
			"rulePriority":   llx.IntData(rule.RulePriority),
			"description":    llx.StringData(rule.Description),
			"tagStatus":      llx.StringData(rule.Selection.TagStatus),
			"tagPatternList": llx.ArrayData(toInterfaceArr(tagPatternList), types.String),
			"tagPrefixList":  llx.ArrayData(toInterfaceArr(tagPrefixList), types.String),
			"countType":      llx.StringData(rule.Selection.CountType),
			"countUnit":      llx.StringData(rule.Selection.CountUnit),
			"countNumber":    llx.IntData(rule.Selection.CountNumber),
			"actionType":     llx.StringData(rule.Action.Type),
		})
	if err != nil {
		return nil, err
	}
	return resource.(*mqlAwsEcrLifecyclePolicyRule), nil
}

func (a *mqlAwsEcrRepository) policy() (any, error) {
	if a.Public.Data {
		a.Policy.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}

	name := a.Name.Data
	region := a.Region.Data
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	svc := conn.Ecr(region)
	ctx := context.Background()

	resp, err := svc.GetRepositoryPolicy(ctx, &ecr.GetRepositoryPolicyInput{
		RepositoryName: &name,
	})
	if err != nil {
		if Is400AccessDeniedError(err) {
			a.Policy.State = plugin.StateIsNull | plugin.StateIsSet
			return nil, nil
		}
		var notFoundErr *ecrtypes.RepositoryPolicyNotFoundException
		if errors.As(err, &notFoundErr) {
			a.Policy.State = plugin.StateIsNull | plugin.StateIsSet
			return nil, nil
		}
		return nil, err
	}

	if resp.PolicyText == nil {
		a.Policy.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}

	var policyDoc any
	if jsonErr := json.Unmarshal([]byte(*resp.PolicyText), &policyDoc); jsonErr != nil {
		return nil, jsonErr
	}
	return policyDoc, nil
}

func (a *mqlAwsEcrRepository) lifecyclePolicy() (*mqlAwsEcrLifecyclePolicy, error) {
	if a.Public.Data {
		a.LifecyclePolicy.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}

	name := a.Name.Data
	region := a.Region.Data
	repoArn := a.Arn.Data
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	svc := conn.Ecr(region)
	ctx := context.Background()

	resp, err := svc.GetLifecyclePolicy(ctx, &ecr.GetLifecyclePolicyInput{
		RepositoryName: &name,
	})
	if err != nil {
		if Is400AccessDeniedError(err) {
			a.LifecyclePolicy.State = plugin.StateIsNull | plugin.StateIsSet
			return nil, nil
		}
		// LifecyclePolicyNotFoundException means no policy is set
		var notFoundErr *ecrtypes.LifecyclePolicyNotFoundException
		if errors.As(err, &notFoundErr) {
			a.LifecyclePolicy.State = plugin.StateIsNull | plugin.StateIsSet
			return nil, nil
		}
		return nil, err
	}

	if resp.LifecyclePolicyText == nil {
		a.LifecyclePolicy.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}

	var doc ecrLifecyclePolicyDoc
	if jsonErr := json.Unmarshal([]byte(*resp.LifecyclePolicyText), &doc); jsonErr != nil {
		return nil, jsonErr
	}

	rules := []any{}
	for _, rule := range doc.Rules {
		mqlRule, err := newMqlEcrLifecyclePolicyRule(a.MqlRuntime, repoArn, rule)
		if err != nil {
			return nil, err
		}
		rules = append(rules, mqlRule)
	}

	policyId := repoArn + "/lifecyclePolicy"
	resource, err := CreateResource(a.MqlRuntime, "aws.ecr.lifecyclePolicy",
		map[string]*llx.RawData{
			"__id":            llx.StringData(policyId),
			"id":              llx.StringData(policyId),
			"lastEvaluatedAt": llx.TimeDataPtr(resp.LastEvaluatedAt),
			"rules":           llx.ArrayData(rules, types.Resource("aws.ecr.lifecyclePolicy.rule")),
		})
	if err != nil {
		return nil, err
	}
	return resource.(*mqlAwsEcrLifecyclePolicy), nil
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

	if conn.Filters.Ecr.Scope == connection.EcrScopePrivate {
		return []any{}, nil
	}

	svc := conn.EcrPublic("us-east-1") // only supported for us-east-1
	res := []any{}

	req := &ecrpublic.DescribeRepositoriesInput{
		RegistryId: aws.String(conn.AccountId()),
	}
	if len(conn.Filters.Ecr.PublicRepositoryNames) > 0 {
		// AWS does not do partial results and returns an error if a single repository
		// supplied in the filters is not found
		req.RepositoryNames = conn.Filters.Ecr.PublicRepositoryNames
	}

	paginator := ecrpublic.NewDescribeRepositoriesPaginator(svc, req)
	for paginator.HasMorePages() {
		repoResp, err := paginator.NextPage(context.TODO())
		if err != nil {
			return nil, err
		}

		for _, r := range repoResp.Repositories {
			mqlRepoResource, err := CreateResource(a.MqlRuntime, ResourceAwsEcrRepository,
				map[string]*llx.RawData{
					"arn":        llx.StringDataPtr(r.RepositoryArn),
					"name":       llx.StringDataPtr(r.RepositoryName),
					"uri":        llx.StringDataPtr(r.RepositoryUri),
					"registryId": llx.StringDataPtr(r.RegistryId),
					"public":     llx.BoolData(true),
					"region":     llx.StringData("us-east-1"),
					// Public ECR does not support scan-on-push, uses immutable tags,
					// and always uses AES256 encryption. These are platform-enforced
					// defaults (not returned by the public ECR DescribeRepositories API).
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

type mqlAwsEcrRepositoryInternal struct {
	catalogFetched bool
	catalogData    *ecrpublic_types.RepositoryCatalogData
	catalogLock    sync.Mutex
}

func (a *mqlAwsEcrRepository) fetchCatalogData() (*ecrpublic_types.RepositoryCatalogData, error) {
	if a.catalogFetched {
		return a.catalogData, nil
	}
	a.catalogLock.Lock()
	defer a.catalogLock.Unlock()
	if a.catalogFetched {
		return a.catalogData, nil
	}

	if !a.Public.Data {
		a.catalogFetched = true
		a.catalogData = nil
		return nil, nil
	}

	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	svc := conn.EcrPublic("us-east-1")
	ctx := context.Background()

	name := a.Name.Data
	resp, err := svc.GetRepositoryCatalogData(ctx, &ecrpublic.GetRepositoryCatalogDataInput{
		RegistryId:     aws.String(conn.AccountId()),
		RepositoryName: &name,
	})
	if err != nil {
		log.Warn().Str("repository", name).Err(err).Msg("could not fetch ECR public catalog data")
		a.catalogFetched = true
		a.catalogData = nil
		return nil, nil
	}

	a.catalogFetched = true
	a.catalogData = resp.CatalogData
	return a.catalogData, nil
}

func (a *mqlAwsEcrRepository) aboutText() (string, error) {
	data, err := a.fetchCatalogData()
	if err != nil || data == nil {
		return "", err
	}
	return convert.ToValue(data.AboutText), nil
}

func (a *mqlAwsEcrRepository) usageText() (string, error) {
	data, err := a.fetchCatalogData()
	if err != nil || data == nil {
		return "", err
	}
	return convert.ToValue(data.UsageText), nil
}

func (a *mqlAwsEcrRepository) catalogDescription() (string, error) {
	data, err := a.fetchCatalogData()
	if err != nil || data == nil {
		return "", err
	}
	return convert.ToValue(data.Description), nil
}

func (a *mqlAwsEcrRepository) operatingSystems() ([]any, error) {
	data, err := a.fetchCatalogData()
	if err != nil || data == nil {
		return []any{}, err
	}
	return convert.SliceAnyToInterface(data.OperatingSystems), nil
}

func (a *mqlAwsEcrRepository) architectures() ([]any, error) {
	data, err := a.fetchCatalogData()
	if err != nil || data == nil {
		return []any{}, err
	}
	return convert.SliceAnyToInterface(data.Architectures), nil
}

func (a *mqlAwsEcrRepository) tags() (map[string]any, error) {
	if a.Public.Data {
		return nil, nil
	}
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	svc := conn.Ecr(a.Region.Data)
	ctx := context.Background()
	arnVal := a.Arn.Data

	resp, err := svc.ListTagsForResource(ctx, &ecr.ListTagsForResourceInput{
		ResourceArn: &arnVal,
	})
	if err != nil {
		if Is400AccessDeniedError(err) {
			return nil, nil
		}
		return nil, err
	}
	tags := make(map[string]any)
	for _, t := range resp.Tags {
		if t.Key != nil && t.Value != nil {
			tags[*t.Key] = *t.Value
		}
	}
	return tags, nil
}

// ==================== ECR Image Scan Findings ====================

type mqlAwsEcrImageInternal struct {
	cachePublic             bool
	scanFetched             bool
	scanFindingsCache       []ecrtypes.ImageScanFinding
	scanStatusCache         string
	scanSeverityCountsCache map[string]int32
	scanLock                sync.Mutex
}

func (a *mqlAwsEcrImage) fetchScanFindings() error {
	if a.scanFetched {
		return nil
	}
	a.scanLock.Lock()
	defer a.scanLock.Unlock()
	if a.scanFetched {
		return nil
	}

	repoName := a.RepoName.Data
	digest := a.Digest.Data
	region := a.Region.Data

	// Public images don't support scan findings
	if a.cachePublic {
		a.scanFetched = true
		a.scanStatusCache = "NOT_SCANNED"
		return nil
	}

	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	svc := conn.Ecr(region)
	ctx := context.Background()

	var findings []ecrtypes.ImageScanFinding
	paginator := ecr.NewDescribeImageScanFindingsPaginator(svc, &ecr.DescribeImageScanFindingsInput{
		RepositoryName: &repoName,
		ImageId:        &ecrtypes.ImageIdentifier{ImageDigest: &digest},
	})
	for paginator.HasMorePages() {
		resp, err := paginator.NextPage(ctx)
		if err != nil {
			var scanNotFound *ecrtypes.ScanNotFoundException
			if errors.As(err, &scanNotFound) {
				a.scanFetched = true
				a.scanStatusCache = "NOT_SCANNED"
				return nil
			}
			if Is400AccessDeniedError(err) {
				a.scanFetched = true
				a.scanStatusCache = ""
				return nil
			}
			return err
		}
		if resp.ImageScanStatus != nil {
			a.scanStatusCache = string(resp.ImageScanStatus.Status)
		}
		if resp.ImageScanFindings != nil {
			findings = append(findings, resp.ImageScanFindings.Findings...)
			if resp.ImageScanFindings.FindingSeverityCounts != nil {
				a.scanSeverityCountsCache = make(map[string]int32)
				for k, v := range resp.ImageScanFindings.FindingSeverityCounts {
					a.scanSeverityCountsCache[string(k)] = v
				}
			}
		}
	}
	a.scanFindingsCache = findings
	a.scanFetched = true
	return nil
}

func (a *mqlAwsEcrImage) scanStatus() (string, error) {
	if err := a.fetchScanFindings(); err != nil {
		return "", err
	}
	return a.scanStatusCache, nil
}

func (a *mqlAwsEcrImage) scanFindings() ([]any, error) {
	if err := a.fetchScanFindings(); err != nil {
		return nil, err
	}

	imageArn := a.Arn.Data
	res := make([]any, 0, len(a.scanFindingsCache))
	for i, f := range a.scanFindingsCache {
		attrs := map[string]any{}
		for _, attr := range f.Attributes {
			if attr.Key != nil {
				attrs[*attr.Key] = convert.ToValue(attr.Value)
			}
		}

		findingId := fmt.Sprintf("%s/scanFinding/%d", imageArn, i)
		mqlFinding, err := CreateResource(a.MqlRuntime, "aws.ecr.image.scanFinding",
			map[string]*llx.RawData{
				"__id":        llx.StringData(findingId),
				"name":        llx.StringDataPtr(f.Name),
				"description": llx.StringDataPtr(f.Description),
				"uri":         llx.StringDataPtr(f.Uri),
				"severity":    llx.StringData(string(f.Severity)),
				"attributes":  llx.DictData(attrs),
			})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlFinding)
	}
	return res, nil
}

func (a *mqlAwsEcrImage) scanFindingSeverityCounts() (any, error) {
	if err := a.fetchScanFindings(); err != nil {
		return nil, err
	}
	if a.scanSeverityCountsCache == nil {
		return nil, nil
	}
	counts := make(map[string]any)
	for k, v := range a.scanSeverityCountsCache {
		counts[k] = int64(v)
	}
	return counts, nil
}

func (a *mqlAwsEcrImageScanFinding) id() (string, error) {
	return a.__id, nil
}

// ==================== ECR Registry Scanning Configuration ====================

func (a *mqlAwsEcr) scanningConfiguration() (*mqlAwsEcrScanningConfiguration, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	svc := conn.Ecr("")
	ctx := context.Background()

	resp, err := svc.GetRegistryScanningConfiguration(ctx, &ecr.GetRegistryScanningConfigurationInput{})
	if err != nil {
		if Is400AccessDeniedError(err) {
			a.ScanningConfiguration.State = plugin.StateIsNull | plugin.StateIsSet
			return nil, nil
		}
		return nil, err
	}
	if resp.ScanningConfiguration == nil {
		a.ScanningConfiguration.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}

	rules := make([]any, 0, len(resp.ScanningConfiguration.Rules))
	for i, rule := range resp.ScanningConfiguration.Rules {
		filters := make([]any, 0, len(rule.RepositoryFilters))
		for _, f := range rule.RepositoryFilters {
			filters = append(filters, map[string]any{
				"filter":     convert.ToValue(f.Filter),
				"filterType": string(f.FilterType),
			})
		}

		mqlRule, err := CreateResource(a.MqlRuntime, "aws.ecr.scanningConfiguration.rule",
			map[string]*llx.RawData{
				"__id":              llx.StringData(fmt.Sprintf("aws.ecr.scanningConfiguration.rule/%d", i)),
				"scanFrequency":     llx.StringData(string(rule.ScanFrequency)),
				"repositoryFilters": llx.ArrayData(filters, types.Dict),
			})
		if err != nil {
			return nil, err
		}
		rules = append(rules, mqlRule)
	}

	mqlConfig, err := CreateResource(a.MqlRuntime, "aws.ecr.scanningConfiguration",
		map[string]*llx.RawData{
			"__id":     llx.StringData("aws.ecr.scanningConfiguration"),
			"scanType": llx.StringData(string(resp.ScanningConfiguration.ScanType)),
			"rules":    llx.ArrayData(rules, types.Resource("aws.ecr.scanningConfiguration.rule")),
		})
	if err != nil {
		return nil, err
	}
	return mqlConfig.(*mqlAwsEcrScanningConfiguration), nil
}

func (a *mqlAwsEcrScanningConfiguration) id() (string, error) {
	return "aws.ecr.scanningConfiguration", nil
}

func (a *mqlAwsEcrScanningConfigurationRule) id() (string, error) {
	return a.__id, nil
}

func initAwsEcrRepository(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}

	if len(args) == 0 {
		if ids := getAssetIdentifier(runtime); ids != nil {
			args["name"] = llx.StringData(ids.name)
			args["arn"] = llx.StringData(ids.arn)
		}
	}

	if args["arn"] == nil && args["name"] == nil {
		return nil, nil, errors.New("arn or name required to fetch ecr repository")
	}

	obj, err := CreateResource(runtime, ResourceAwsEcr, map[string]*llx.RawData{})
	if err != nil {
		return nil, nil, err
	}
	e := obj.(*mqlAwsEcr)

	repos := []any{}
	var lastErr error
	if priv, err := e.privateRepositories(); err == nil {
		repos = append(repos, priv...)
	} else {
		lastErr = err
	}
	if pub, err := e.publicRepositories(); err == nil {
		repos = append(repos, pub...)
	} else {
		lastErr = err
	}
	if len(repos) == 0 && lastErr != nil {
		return nil, nil, fmt.Errorf("failed to list ecr repositories: %w", lastErr)
	}

	var arnVal, nameVal string
	if args["arn"] != nil {
		arnVal, _ = args["arn"].Value.(string)
	}
	if args["name"] != nil {
		nameVal, _ = args["name"].Value.(string)
	}
	for _, raw := range repos {
		r := raw.(*mqlAwsEcrRepository)
		if (arnVal != "" && r.Arn.Data == arnVal) || (nameVal != "" && r.Name.Data == nameVal) {
			return args, r, nil
		}
	}
	return nil, nil, errors.New("ecr repository does not exist")
}
