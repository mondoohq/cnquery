// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/aws-sdk-go-v2/service/s3control"
	s3controltypes "github.com/aws/aws-sdk-go-v2/service/s3control/types"
	"github.com/aws/smithy-go"
	"github.com/aws/smithy-go/transport/http"
	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v11/providers/aws/connection"
	"go.mondoo.com/cnquery/v11/providers/aws/resources/awspolicy"
	"go.mondoo.com/cnquery/v11/types"
)

func (a *mqlAwsS3control) id() (string, error) {
	return "aws.s3control", nil
}

func (a *mqlAwsS3control) accountPublicAccessBlock() (interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	svc := conn.S3Control("")
	ctx := context.Background()

	publicAccessBlock, err := svc.GetPublicAccessBlock(ctx, &s3control.GetPublicAccessBlockInput{
		AccountId: aws.String(conn.AccountId()),
	})
	if err != nil {
		var notFoundErr *s3controltypes.NoSuchPublicAccessBlockConfiguration
		if errors.As(err, &notFoundErr) {
			return nil, nil
		}
		return nil, err
	}

	return convert.JsonToDict(publicAccessBlock.PublicAccessBlockConfiguration)
}

func (a *mqlAwsS3) id() (string, error) {
	return "aws.s3", nil
}

func (a *mqlAwsS3) buckets() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	svc := conn.S3("")
	ctx := context.Background()

	totalBuckets := make([]s3types.Bucket, 0)
	params := &s3.ListBucketsInput{}
	paginator := s3.NewListBucketsPaginator(svc, params, func(o *s3.ListBucketsPaginatorOptions) {
		o.Limit = 100
	})
	for paginator.HasMorePages() {
		output, err := paginator.NextPage(context.TODO())
		if err != nil {
			return nil, err
		}
		totalBuckets = append(totalBuckets, output.Buckets...)
	}

	res := []interface{}{}
	for i := range totalBuckets {
		bucket := totalBuckets[i]

		location, err := svc.GetBucketLocation(ctx, &s3.GetBucketLocationInput{
			Bucket: bucket.Name,
		})
		if err != nil {
			log.Error().Err(err).Str("bucket", *bucket.Name).Msg("Could not get bucket location")
			continue
		}
		if location == nil {
			log.Error().Err(err).Str("bucket", *bucket.Name).Msg("Could not get bucket location (returned null)")
			continue
		}

		region := string(location.LocationConstraint)
		// us-east-1 returns "" therefore we set it explicitly
		// https://docs.aws.amazon.com/AmazonS3/latest/API/API_GetBucketLocation.html#API_GetBucketLocation_ResponseSyntax
		if region == "" {
			region = "us-east-1"
		}
		mqlS3Bucket, err := CreateResource(a.MqlRuntime, "aws.s3.bucket",
			map[string]*llx.RawData{
				"name":        llx.StringDataPtr(bucket.Name),
				"arn":         llx.StringData(fmt.Sprintf(s3ArnPattern, convert.ToValue(bucket.Name))),
				"exists":      llx.BoolData(true),
				"location":    llx.StringData(region),
				"createdTime": llx.TimeDataPtr(bucket.CreationDate),
				"createdAt":   llx.TimeDataPtr(bucket.CreationDate),
			})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlS3Bucket)
	}

	return res, nil
}

func initAwsS3BucketPolicy(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	// reuse the init func for the bucket
	_, s3bucketResource, err := initAwsS3Bucket(runtime, args)
	if err != nil {
		return args, nil, err
	}
	// then use it to get its policy
	policyResource := s3bucketResource.(*mqlAwsS3Bucket).GetPolicy()
	if policyResource != nil && policyResource.State == plugin.StateIsSet {
		return args, policyResource.Data, nil
	}

	// no policy found
	resource := &mqlAwsS3BucketPolicy{}
	resource.Id.State = plugin.StateIsNull | plugin.StateIsSet
	resource.Name.State = plugin.StateIsNull | plugin.StateIsSet
	resource.Document.State = plugin.StateIsNull | plugin.StateIsSet
	resource.Version.State = plugin.StateIsNull | plugin.StateIsSet
	resource.Statements.State = plugin.StateIsNull | plugin.StateIsSet
	resource.BucketName = plugin.TValue[string]{
		Data: s3bucketResource.(*mqlAwsS3Bucket).GetName().Data, State: plugin.StateIsSet,
	}
	return args, resource, nil
}

func initAwsS3Bucket(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	// NOTE: bucket only initializes with arn and name
	if len(args) >= 2 {
		return args, nil, nil
	}
	if len(args) == 0 {
		if ids := getAssetIdentifier(runtime); ids != nil {
			args["name"] = llx.StringData(ids.name)
			args["arn"] = llx.StringData(ids.arn)
		}
	}
	if args["arn"] == nil && args["name"] == nil {
		return nil, nil, errors.New("arn or name required to fetch aws s3 bucket")
	}

	// construct arn of bucket name if misssing
	var arn string
	if args["arn"] != nil {
		arn = args["arn"].Value.(string)
		if !strings.HasPrefix(arn, "arn:aws:s3:") {
			return nil, nil, errors.Newf("not a valid bucket ARN '%s'", arn)
		}
	} else {
		nameVal := args["name"].Value.(string)
		arn = fmt.Sprintf(s3ArnPattern, nameVal)
	}
	log.Debug().Str("arn", arn).Msg("init s3 bucket with arn")

	// load all s3 buckets
	obj, err := runtime.CreateResource(runtime, "aws.s3", map[string]*llx.RawData{})
	if err != nil {
		return nil, nil, err
	}
	awsS3 := obj.(*mqlAwsS3)

	rawResources := awsS3.GetBuckets()
	if rawResources.Error != nil {
		return nil, nil, rawResources.Error
	}

	// iterate over security groups and find the one with the arn
	for _, rawResource := range rawResources.Data {
		bucket := rawResource.(*mqlAwsS3Bucket)
		if bucket.Arn.Data == arn {
			return args, bucket, nil
		}
	}
	// it is possible for a resource to reference a non-existent/deleted bucket, so here we
	// create the object, noting that it no longer exists but is still recorded as part of some resources
	splitArn := strings.Split(arn, ":::")
	if len(splitArn) != 2 {
		return args, nil, nil
	}
	name := splitArn[1]
	log.Debug().Msgf("no bucket found for %s", arn)
	mqlAwsS3Bucket, err := CreateResource(runtime, "aws.s3.bucket",
		map[string]*llx.RawData{
			"arn":    llx.StringData(arn),
			"name":   llx.StringData(name),
			"exists": llx.BoolData(false),
		})
	return nil, mqlAwsS3Bucket, err
}

func (a *mqlAwsS3Bucket) id() (string, error) {
	// assumes bucket names are globally unique, which they are right now
	return a.Arn.Data, nil
}

func (a *mqlAwsS3Bucket) policy() (*mqlAwsS3BucketPolicy, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	bucketname := a.Name.Data
	location := a.Location.Data
	svc := conn.S3(location)
	ctx := context.Background()

	policy, err := svc.GetBucketPolicy(ctx, &s3.GetBucketPolicyInput{
		Bucket: &bucketname,
	})
	if err != nil {
		if isNotFoundForS3(err) {
			a.Policy.State = plugin.StateIsNull | plugin.StateIsSet
			return nil, nil
		}
		return nil, err
	}

	if policy != nil && policy.Policy != nil {
		parsedPolicy, err := parseS3BucketPolicy(*policy.Policy)
		if err != nil {
			return nil, err
		}
		// create the policy resource
		mqlS3BucketPolicy, err := CreateResource(a.MqlRuntime, "aws.s3.bucket.policy",
			map[string]*llx.RawData{
				"id":         llx.StringData(parsedPolicy.Id),
				"name":       llx.StringData(bucketname),
				"bucketName": llx.StringData(bucketname),
				"version":    llx.StringData(parsedPolicy.Version),
				"document":   llx.StringDataPtr(policy.Policy),
			})
		if err != nil {
			return nil, err
		}

		return mqlS3BucketPolicy.(*mqlAwsS3BucketPolicy), nil
	}

	// no bucket policy found, return nil for the policy
	a.Policy.State = plugin.StateIsNull | plugin.StateIsSet
	return nil, nil
}

func (a *mqlAwsS3Bucket) tags() (map[string]interface{}, error) {
	bucketname := a.Name.Data
	location := a.Location.Data

	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	svc := conn.S3(location)
	ctx := context.Background()

	tags, err := svc.GetBucketTagging(ctx, &s3.GetBucketTaggingInput{
		Bucket: &bucketname,
	})
	if err != nil {
		var apiErr smithy.APIError
		if errors.As(err, &apiErr) {
			if apiErr.ErrorCode() == "NoSuchTagSet" {
				return nil, nil
			}
		}

		return nil, err
	}

	res := map[string]interface{}{}
	for i := range tags.TagSet {
		tag := tags.TagSet[i]
		res[convert.ToValue(tag.Key)] = convert.ToValue(tag.Value)
	}

	return res, nil
}

func (a *mqlAwsS3Bucket) location() (string, error) {
	bucketname := a.Name.Data

	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	svc := conn.S3("")
	ctx := context.Background()

	location, err := svc.GetBucketLocation(ctx, &s3.GetBucketLocationInput{
		Bucket: &bucketname,
	})
	if err != nil {
		return "", err
	}

	region := string(location.LocationConstraint)
	// us-east-1 returns "" therefore we set it explicitly
	// https://docs.aws.amazon.com/AmazonS3/latest/API/API_GetBucketLocation.html#API_GetBucketLocation_ResponseSyntax
	if region == "" {
		region = "us-east-1"
	}
	return region, nil
}

func (a *mqlAwsS3Bucket) gatherAcl() (*s3.GetBucketAclOutput, error) {
	bucketname := a.Name.Data
	location := a.Location.Data

	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	svc := conn.S3(location)
	ctx := context.Background()

	acl, err := svc.GetBucketAcl(ctx, &s3.GetBucketAclInput{
		Bucket: &bucketname,
	})
	if err != nil {
		return nil, err
	}

	// TODO: store in cache
	return acl, nil
}

func (a *mqlAwsS3Bucket) acl() ([]interface{}, error) {
	bucketname := a.Name.Data

	acl, err := a.gatherAcl()
	if err != nil {
		return nil, err
	}

	res := []interface{}{}
	for i := range acl.Grants {
		grant := acl.Grants[i]

		// NOTE: not all grantees have URI and IDs, canonical users have id, groups have URIs and the
		// display name may not be unique
		if grant.Grantee == nil || (grant.Grantee.URI == nil && grant.Grantee.ID == nil) {
			return nil, fmt.Errorf("unsupported grant: %v", grant)
		}

		grantee := map[string]interface{}{
			"id":           convert.ToValue(grant.Grantee.ID),
			"name":         convert.ToValue(grant.Grantee.DisplayName),
			"emailAddress": convert.ToValue(grant.Grantee.EmailAddress),
			"type":         string(grant.Grantee.Type),
			"uri":          convert.ToValue(grant.Grantee.URI),
		}

		id := bucketname + "/" + string(grant.Permission)
		if grant.Grantee.URI != nil {
			id = id + "/" + *grant.Grantee.URI
		} else {
			id = id + "/" + *grant.Grantee.ID
		}

		mqlBucketGrant, err := CreateResource(a.MqlRuntime, "aws.s3.bucket.grant",
			map[string]*llx.RawData{
				"id":         llx.StringData(id),
				"name":       llx.StringData(bucketname),
				"permission": llx.StringData(string(grant.Permission)),
				"grantee":    llx.MapData(grantee, types.String),
			})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlBucketGrant)
	}
	return res, nil
}

func (a *mqlAwsS3Bucket) publicAccessBlock() (interface{}, error) {
	bucketname := a.Name.Data
	location := a.Location.Data
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	svc := conn.S3(location)
	ctx := context.Background()

	publicAccessBlock, err := svc.GetPublicAccessBlock(ctx, &s3.GetPublicAccessBlockInput{
		Bucket: &bucketname,
	})
	if err != nil {
		if isNotFoundForS3(err) {
			return nil, nil
		}
		return nil, err
	}

	return convert.JsonToDict(publicAccessBlock.PublicAccessBlockConfiguration)
}

func (a *mqlAwsS3Bucket) owner() (map[string]interface{}, error) {
	acl, err := a.gatherAcl()
	if err != nil {
		return nil, err
	}

	if acl.Owner == nil {
		return nil, errors.New("could not gather aws s3 bucket's owner information")
	}

	res := map[string]interface{}{}
	res["id"] = convert.ToValue(acl.Owner.ID)
	res["name"] = convert.ToValue(acl.Owner.DisplayName)

	return res, nil
}

// see https://docs.aws.amazon.com/AmazonS3/latest/dev/acl-overview.html
const (
	s3AuthenticatedUsersGroup = "http://acs.amazonaws.com/groups/global/AuthenticatedUsers"
	s3AllUsersGroup           = "http://acs.amazonaws.com/groups/global/AllUsers"
)

func (a *mqlAwsS3Bucket) public() (bool, error) {
	var (
		bucketname = a.Name.Data
		location   = a.Location.Data
		conn       = a.MqlRuntime.Connection.(*connection.AwsConnection)
		svc        = conn.S3(location)
		ctx        = context.Background()
	)

	// Check Public Access Block settings first
	publicAccess, err := svc.GetPublicAccessBlock(ctx, &s3.GetPublicAccessBlockInput{
		Bucket: &bucketname,
	})
	if err != nil && !isNotFoundForS3(err) {
		return false, err
	}

	notPublic := false
	if publicAccess != nil && publicAccess.PublicAccessBlockConfiguration != nil {
		accessBlock := publicAccess.PublicAccessBlockConfiguration
		if accessBlock.BlockPublicAcls != nil && *accessBlock.BlockPublicAcls {
			notPublic = true
		}
		if accessBlock.BlockPublicPolicy != nil && *accessBlock.BlockPublicPolicy {
			notPublic = true
		}
		if accessBlock.IgnorePublicAcls != nil && *accessBlock.IgnorePublicAcls {
			notPublic = true
		}
		if accessBlock.RestrictPublicBuckets != nil && *accessBlock.RestrictPublicBuckets {
			notPublic = true
		}
	}
	if notPublic {
		return false, nil // Public access is restricted
	}

	// Then, use GetBucketPolicyStatus to determine public access
	statusOutput, err := svc.GetBucketPolicyStatus(ctx, &s3.GetBucketPolicyStatusInput{
		Bucket: &bucketname,
	})
	if err != nil && !isNotFoundForS3(err) {
		return false, err
	}
	if statusOutput != nil &&
		statusOutput.PolicyStatus != nil &&
		statusOutput.PolicyStatus.IsPublic != nil {
		return *statusOutput.PolicyStatus.IsPublic, nil
	}

	// If that didn't work, fetch the bucket policy manually and parse it
	bucketPolicyResource := a.GetPolicy()
	if bucketPolicyResource.State == plugin.StateIsSet {
		bucketPolicy, err := bucketPolicyResource.Data.parsePolicyDocument()
		if err != nil {
			return false, err
		}

		for _, statement := range bucketPolicy.Statements {
			if statement.Effect != "Allow" {
				continue
			}
			if awsPrincipal, ok := statement.Principal["AWS"]; ok {
				if slices.Contains(awsPrincipal, "*") {
					return true, nil
				}
			}
		}
	}

	// Finally check for bucket ACLs
	acl, err := a.gatherAcl()
	if err != nil {
		return false, err
	}

	for i := range acl.Grants {
		grant := acl.Grants[i]
		if grant.Grantee.Type == s3types.TypeGroup && (convert.ToValue(grant.Grantee.URI) == s3AuthenticatedUsersGroup || convert.ToValue(grant.Grantee.URI) == s3AllUsersGroup) {
			return true, nil
		}
	}
	return false, nil
}

func (a *mqlAwsS3Bucket) cors() ([]interface{}, error) {
	bucketname := a.Name.Data
	location := a.Location.Data

	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	svc := conn.S3(location)
	ctx := context.Background()

	cors, err := svc.GetBucketCors(ctx, &s3.GetBucketCorsInput{
		Bucket: &bucketname,
	})
	if err != nil {
		if isNotFoundForS3(err) {
			return nil, nil
		}
		return nil, err
	}

	res := []interface{}{}
	for i := range cors.CORSRules {
		corsrule := cors.CORSRules[i]
		mqlBucketCors, err := CreateResource(a.MqlRuntime, "aws.s3.bucket.corsrule",
			map[string]*llx.RawData{
				"name":           llx.StringData(bucketname),
				"allowedHeaders": llx.ArrayData(toInterfaceArr(corsrule.AllowedHeaders), types.String),
				"allowedMethods": llx.ArrayData(toInterfaceArr(corsrule.AllowedMethods), types.String),
				"allowedOrigins": llx.ArrayData(toInterfaceArr(corsrule.AllowedOrigins), types.String),
				"exposeHeaders":  llx.ArrayData(toInterfaceArr(corsrule.ExposeHeaders), types.String),
				"maxAgeSeconds":  llx.IntDataDefault(corsrule.MaxAgeSeconds, 0),
			})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlBucketCors)
	}

	return res, nil
}

func (a *mqlAwsS3Bucket) logging() (map[string]interface{}, error) {
	bucketname := a.Name.Data
	bucketlocation := a.Location.Data

	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	svc := conn.S3(bucketlocation)
	ctx := context.Background()

	logging, err := svc.GetBucketLogging(ctx, &s3.GetBucketLoggingInput{
		Bucket: &bucketname,
	})
	if err != nil {
		return nil, err
	}

	res := map[string]interface{}{}

	if logging != nil && logging.LoggingEnabled != nil {
		if logging.LoggingEnabled.TargetPrefix != nil {
			res["TargetPrefix"] = convert.ToValue(logging.LoggingEnabled.TargetPrefix)
		}

		if logging.LoggingEnabled.TargetBucket != nil {
			res["TargetBucket"] = convert.ToValue(logging.LoggingEnabled.TargetBucket)
		}

		// it is becoming a more complex object similar to aws.s3.bucket.grant
		// if logging.LoggingEnabled.TargetGrants != nil {
		// 	res["TargetGrants"] = *logging.LoggingEnabled.TargetGrants
		// }
	}

	return res, nil
}

func (a *mqlAwsS3Bucket) versioning() (map[string]interface{}, error) {
	bucketname := a.Name.Data
	location := a.Location.Data

	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	svc := conn.S3(location)
	ctx := context.Background()

	versioning, err := svc.GetBucketVersioning(ctx, &s3.GetBucketVersioningInput{
		Bucket: &bucketname,
	})
	if err != nil {
		return nil, err
	}

	res := map[string]interface{}{}

	if versioning != nil {
		res["MFADelete"] = string(versioning.MFADelete)
		res["Status"] = string(versioning.Status)
	}

	return res, nil
}

func (a *mqlAwsS3Bucket) replication() (interface{}, error) {
	bucketname := a.Name.Data
	region := a.Location.Data

	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	svc := conn.S3(region)
	ctx := context.Background()

	bucketReplication, err := svc.GetBucketReplication(ctx, &s3.GetBucketReplicationInput{
		Bucket: &bucketname,
	})
	if err != nil {
		if isNotFoundForS3(err) {
			return nil, nil
		}
		return nil, err
	}
	return convert.JsonToDict(bucketReplication.ReplicationConfiguration)
}

func (a *mqlAwsS3Bucket) encryption() (interface{}, error) {
	bucketname := a.Name.Data
	region := a.Location.Data

	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	svc := conn.S3(region)
	ctx := context.Background()

	encryption, err := svc.GetBucketEncryption(ctx, &s3.GetBucketEncryptionInput{
		Bucket: &bucketname,
	})
	var res interface{}
	if err != nil {
		var ae smithy.APIError
		if errors.As(err, &ae) {
			if ae.ErrorCode() == "ServerSideEncryptionConfigurationNotFoundError" {
				return res, nil
			}
		}
		return nil, err
	}

	return convert.JsonToDict(encryption.ServerSideEncryptionConfiguration)
}

func (a *mqlAwsS3Bucket) defaultLock() (string, error) {
	bucketname := a.Name.Data
	region := a.Location.Data

	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	svc := conn.S3(region)
	ctx := context.Background()

	objectLockConfiguration, err := svc.GetObjectLockConfiguration(ctx, &s3.GetObjectLockConfigurationInput{
		Bucket: &bucketname,
	})
	if err != nil {
		if isNotFoundForS3(err) {
			return "", nil
		}
		return "", err
	}

	return string(objectLockConfiguration.ObjectLockConfiguration.ObjectLockEnabled), nil
}

func (a *mqlAwsS3Bucket) staticWebsiteHosting() (map[string]interface{}, error) {
	bucketname := a.Name.Data
	region := a.Location.Data

	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	svc := conn.S3(region)
	ctx := context.Background()

	website, err := svc.GetBucketWebsite(ctx, &s3.GetBucketWebsiteInput{
		Bucket: &bucketname,
	})
	if err != nil {
		if isNotFoundForS3(err) {
			return nil, nil
		}
		return nil, err
	}

	res := map[string]interface{}{}

	if website != nil {
		if website.ErrorDocument != nil {
			res["ErrorDocument"] = convert.ToValue(website.ErrorDocument.Key)
		}

		if website.IndexDocument != nil {
			res["IndexDocument"] = convert.ToValue(website.IndexDocument.Suffix)
		}
	}

	return res, nil
}

func (a *mqlAwsS3BucketGrant) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAwsS3BucketCorsrule) id() (string, error) {
	return "s3.bucket.corsrule " + a.Name.Data, nil
}

func (a *mqlAwsS3BucketPolicy) id() (string, error) {
	// NOTE that `policy.Id` might or might not exist and,
	// it is NOT unique for s3 bucket policies. what we need
	// here is the bucket name, which is unique globally.
	return fmt.Sprintf("aws.s3.bucket/%s/policy", a.BucketName.Data), nil
}

func (a *mqlAwsS3BucketPolicy) parsePolicyDocument() (*awspolicy.S3BucketPolicy, error) {
	return parseS3BucketPolicy(a.Document.Data)
}

func parseS3BucketPolicy(document string) (*awspolicy.S3BucketPolicy, error) {
	var policy awspolicy.S3BucketPolicy
	err := json.Unmarshal([]byte(document), &policy)
	return &policy, err
}

func (a *mqlAwsS3BucketPolicy) version() (string, error) {
	policy, err := a.parsePolicyDocument()
	if err != nil {
		return "", err
	}
	return policy.Version, nil
}

func (a *mqlAwsS3BucketPolicy) statements() ([]interface{}, error) {
	policy, err := a.parsePolicyDocument()
	if err != nil {
		return nil, err
	}
	return convert.JsonToDictSlice(policy.Statements)
}

func isNotFoundForS3(err error) bool {
	if err == nil {
		return false
	}

	var respErr *http.ResponseError
	var notFoundErr *s3types.NotFound

	if errors.As(err, &notFoundErr) {
		return true
	} else if errors.As(err, &respErr) {
		if respErr.HTTPStatusCode() == 404 {
			return true
		}
	}

	return false
}
