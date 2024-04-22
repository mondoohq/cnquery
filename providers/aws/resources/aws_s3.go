// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/aws-sdk-go-v2/service/s3control"
	s3controltypes "github.com/aws/aws-sdk-go-v2/service/s3control/types"
	"github.com/aws/aws-sdk-go/aws"
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

	buckets, err := svc.ListBuckets(ctx, &s3.ListBucketsInput{})
	if err != nil {
		return nil, err
	}

	res := []interface{}{}
	for i := range buckets.Buckets {
		bucket := buckets.Buckets[i]

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
				"arn":         llx.StringData(fmt.Sprintf(s3ArnPattern, convert.ToString(bucket.Name))),
				"exists":      llx.BoolData(true),
				"location":    llx.StringData(region),
				"createdTime": llx.TimeDataPtr(bucket.CreationDate),
			})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlS3Bucket)
	}

	return res, nil
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
	for i := range rawResources.Data {
		bucket := rawResources.Data[i].(*mqlAwsS3Bucket)
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

func emptyAwsS3BucketPolicy(runtime *plugin.Runtime) (*mqlAwsS3BucketPolicy, error) {
	res, err := CreateResource(runtime, "aws.s3.bucket.policy", map[string]*llx.RawData{
		"name":       llx.StringData(""),
		"document":   llx.StringData("{}"),
		"version":    llx.StringData(""),
		"id":         llx.StringData(""),
		"statements": llx.ArrayData([]interface{}{}, types.Dict),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAwsS3BucketPolicy), nil
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
			return emptyAwsS3BucketPolicy(a.MqlRuntime)
		}
		return nil, err
	}

	if policy != nil && policy.Policy != nil {
		// create the policy resource
		mqlS3BucketPolicy, err := CreateResource(a.MqlRuntime, "aws.s3.bucket.policy",
			map[string]*llx.RawData{
				"name":     llx.StringData(bucketname),
				"document": llx.StringDataPtr(policy.Policy),
			})
		if err != nil {
			return nil, err
		}
		return mqlS3BucketPolicy.(*mqlAwsS3BucketPolicy), nil
	}

	// no bucket policy found, return nil for the policy
	return emptyAwsS3BucketPolicy(a.MqlRuntime)
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
		res[convert.ToString(tag.Key)] = convert.ToString(tag.Value)
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
			"id":           convert.ToString(grant.Grantee.ID),
			"name":         convert.ToString(grant.Grantee.DisplayName),
			"emailAddress": convert.ToString(grant.Grantee.EmailAddress),
			"type":         string(grant.Grantee.Type),
			"uri":          convert.ToString(grant.Grantee.URI),
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
	res["id"] = convert.ToString(acl.Owner.ID)
	res["name"] = convert.ToString(acl.Owner.DisplayName)

	return res, nil
}

// see https://docs.aws.amazon.com/AmazonS3/latest/dev/acl-overview.html
const (
	s3AuthenticatedUsersGroup = "http://acs.amazonaws.com/groups/global/AuthenticatedUsers"
	s3AllUsersGroup           = "http://acs.amazonaws.com/groups/global/AllUsers"
)

func (a *mqlAwsS3Bucket) public() (bool, error) {
	acl, err := a.gatherAcl()
	if err != nil {
		return false, err
	}

	for i := range acl.Grants {
		grant := acl.Grants[i]
		if grant.Grantee.Type == s3types.TypeGroup && (convert.ToString(grant.Grantee.URI) == s3AuthenticatedUsersGroup || convert.ToString(grant.Grantee.URI) == s3AllUsersGroup) {
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
			res["TargetPrefix"] = convert.ToString(logging.LoggingEnabled.TargetPrefix)
		}

		if logging.LoggingEnabled.TargetBucket != nil {
			res["TargetBucket"] = convert.ToString(logging.LoggingEnabled.TargetBucket)
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
			res["ErrorDocument"] = convert.ToString(website.ErrorDocument.Key)
		}

		if website.IndexDocument != nil {
			res["IndexDocument"] = convert.ToString(website.IndexDocument.Suffix)
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
	policy, err := a.parsePolicyDocument()
	if err != nil || policy == nil {
		return "none", err
	}

	a.Id = plugin.TValue[string]{Data: policy.Id, State: plugin.StateIsSet}
	return policy.Id, nil
}

func (a *mqlAwsS3BucketPolicy) parsePolicyDocument() (*awspolicy.S3BucketPolicy, error) {
	data := a.Document.Data

	var policy awspolicy.S3BucketPolicy
	err := json.Unmarshal([]byte(data), &policy)
	if err != nil {
		return nil, err
	}

	return &policy, nil
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
