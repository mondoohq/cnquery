package aws

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
	"github.com/aws/smithy-go/transport/http"
	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/resources"
	"go.mondoo.io/mondoo/resources/packs/aws/awspolicy"
	"go.mondoo.io/mondoo/resources/packs/core"
)

const (
	s3ArnPattern = "arn:aws:s3:::%s"
)

func (p *mqlAwsS3) id() (string, error) {
	return "aws.s3", nil
}

func (p *mqlAwsS3) GetBuckets() ([]interface{}, error) {
	provider, err := awsProvider(p.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	svc := provider.S3("")
	ctx := context.Background()

	buckets, err := svc.ListBuckets(ctx, &s3.ListBucketsInput{})
	if err != nil {
		return nil, err
	}

	res := []interface{}{}
	for i := range buckets.Buckets {
		bucket := buckets.Buckets[i]

		mqlS3Bucket, err := p.MotorRuntime.CreateResource("aws.s3.bucket",
			"name", core.ToString(bucket.Name),
			"arn", fmt.Sprintf(s3ArnPattern, core.ToString(bucket.Name)),
			"exists", true,
		)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlS3Bucket)
	}

	return res, nil
}

func (p *mqlAwsS3Bucket) init(args *resources.Args) (*resources.Args, AwsS3Bucket, error) {
	// NOTE: bucket only initializes with arn and name
	if len(*args) >= 2 {
		return args, nil, nil
	}

	if (*args)["arn"] == nil && (*args)["name"] == nil {
		return nil, nil, errors.New("arn or name required to fetch aws s3 bucket")
	}

	// construct arn of bucket name if misssing
	var arn string
	if (*args)["arn"] != nil {
		arn = (*args)["arn"].(string)
	} else {
		nameVal := (*args)["name"].(string)
		arn = fmt.Sprintf(s3ArnPattern, nameVal)
	}
	log.Debug().Str("arn", arn).Msg("init s3 bucket with arn")

	// load all s3 buckets
	obj, err := p.MotorRuntime.CreateResource("aws.s3")
	if err != nil {
		return nil, nil, err
	}
	awsS3 := obj.(AwsS3)

	rawResources, err := awsS3.Buckets()
	if err != nil {
		return nil, nil, err
	}

	// iterate over security groups and find the one with the arn
	for i := range rawResources {
		bucket := rawResources[i].(AwsS3Bucket)
		mqlBucketArn, err := bucket.Arn()
		if err != nil {
			return nil, nil, err
		}
		if mqlBucketArn == arn {
			return args, bucket, nil
		}
	}
	// it is possible for a resource to reference a non-existent/deleted bucket, so here we
	// create the object, noting that it no longer exists but is still recorded as part of some resources
	splitArn := strings.Split(arn, ":::")
	name := splitArn[1]
	log.Debug().Msgf("no bucket found for %s", arn)
	mqlAwsS3Bucket, err := p.MotorRuntime.CreateResource("aws.s3.bucket",
		"arn", arn,
		"name", name,
		"exists", false,
	)
	return nil, mqlAwsS3Bucket.(AwsS3Bucket), err
}

func (p *mqlAwsS3Bucket) id() (string, error) {
	// assumes bucket names are globally unique, which they are right now
	return p.Arn()
}

func (p *mqlAwsS3Bucket) GetPolicy() (interface{}, error) {
	bucketname, err := p.Name()
	if err != nil {
		return nil, err
	}

	location, err := p.Location()
	if err != nil {
		return nil, err
	}

	provider, err := awsProvider(p.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	svc := provider.S3(location)
	ctx := context.Background()

	policy, err := svc.GetBucketPolicy(ctx, &s3.GetBucketPolicyInput{
		Bucket: &bucketname,
	})
	if err != nil {
		if isNotFoundForS3(err) {
			return nil, nil
		}
		return nil, err
	}

	if policy != nil && policy.Policy != nil {
		// create the plicy resource
		mqlS3BucketPolicy, err := p.MotorRuntime.CreateResource("aws.s3.bucket.policy",
			"name", bucketname,
			"document", core.ToString(policy.Policy),
		)
		if err != nil {
			return nil, err
		}
		return mqlS3BucketPolicy, nil
	}

	// no bucket policy found, return nil for the policy
	return nil, nil
}

func (p *mqlAwsS3Bucket) GetTags() (map[string]interface{}, error) {
	bucketname, err := p.Name()
	if err != nil {
		return nil, err
	}
	location, err := p.Location()
	if err != nil {
		return nil, err
	}

	provider, err := awsProvider(p.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	svc := provider.S3(location)
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
		res[core.ToString(tag.Key)] = core.ToString(tag.Value)
	}

	return res, nil
}

func (p *mqlAwsS3Bucket) GetLocation() (string, error) {
	bucketname, err := p.Name()
	if err != nil {
		return "", err
	}

	at, err := awsProvider(p.MotorRuntime.Motor.Provider)
	if err != nil {
		return "", err
	}

	svc := at.S3("")
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

func (p *mqlAwsS3Bucket) gatherAcl() (*s3.GetBucketAclOutput, error) {
	bucketname, err := p.Name()
	if err != nil {
		return nil, err
	}
	location, err := p.Location()
	if err != nil {
		return nil, err
	}

	provider, err := awsProvider(p.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	svc := provider.S3(location)
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

func (p *mqlAwsS3Bucket) GetAcl() ([]interface{}, error) {
	bucketname, err := p.Name()
	if err != nil {
		return nil, err
	}

	acl, err := p.gatherAcl()
	if err != nil {
		return nil, err
	}

	res := []interface{}{}
	for i := range acl.Grants {
		grant := acl.Grants[i]

		grantee := map[string]interface{}{}
		grantee["id"] = core.ToString(grant.Grantee.ID)
		grantee["name"] = core.ToString(grant.Grantee.DisplayName)
		grantee["emailAddress"] = core.ToString(grant.Grantee.EmailAddress)
		grantee["type"] = string(grant.Grantee.Type)
		grantee["uri"] = core.ToString(grant.Grantee.URI)

		// NOTE: not all grantees have URI and IDs, canonical users have id, groups have URIs and the
		// display name may not be unique
		if grant.Grantee == nil || (grant.Grantee.URI == nil && grant.Grantee.ID == nil) {
			return nil, fmt.Errorf("unsupported grant: %v", grant)
		}

		id := bucketname + "/" + string(grant.Permission)
		if grant.Grantee.URI != nil {
			id = id + "/" + *grant.Grantee.URI
		} else {
			id = id + "/" + *grant.Grantee.ID
		}

		mqlBucketGrant, err := p.MotorRuntime.CreateResource("aws.s3.bucket.grant",
			"id", id,
			"name", bucketname,
			"permission", string(grant.Permission),
			"grantee", grantee,
		)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlBucketGrant)
	}
	return res, nil
}

func (p *mqlAwsS3Bucket) GetPublicAccessBlock() (interface{}, error) {
	bucketname, err := p.Name()
	if err != nil {
		return nil, err
	}
	location, err := p.Location()
	if err != nil {
		return nil, err
	}
	provider, err := awsProvider(p.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	svc := provider.S3(location)
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

	return core.JsonToDict(publicAccessBlock.PublicAccessBlockConfiguration)
}

func (p *mqlAwsS3Bucket) GetOwner() (map[string]interface{}, error) {
	acl, err := p.gatherAcl()
	if err != nil {
		return nil, err
	}

	if acl.Owner == nil {
		return nil, errors.New("could not gather aws s3 bucket's owner information")
	}

	res := map[string]interface{}{}
	res["id"] = core.ToString(acl.Owner.ID)
	res["name"] = core.ToString(acl.Owner.DisplayName)

	return res, nil
}

// see https://docs.aws.amazon.com/AmazonS3/latest/dev/acl-overview.html
const (
	s3AuthenticatedUsersGroup = "http://acs.amazonaws.com/groups/global/AuthenticatedUsers"
	s3AllUsersGroup           = "http://acs.amazonaws.com/groups/global/AllUsers"
)

func (p *mqlAwsS3Bucket) GetPublic() (bool, error) {
	acl, err := p.gatherAcl()
	if err != nil {
		return false, err
	}

	for i := range acl.Grants {
		grant := acl.Grants[i]
		if grant.Grantee.Type == types.TypeGroup && (core.ToString(grant.Grantee.URI) == s3AuthenticatedUsersGroup || core.ToString(grant.Grantee.URI) == s3AllUsersGroup) {
			return true, nil
		}
	}
	return false, nil
}

func (p *mqlAwsS3Bucket) GetCors() ([]interface{}, error) {
	bucketname, err := p.Name()
	if err != nil {
		return nil, err
	}
	location, err := p.Location()
	if err != nil {
		return nil, err
	}

	provider, err := awsProvider(p.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	svc := provider.S3(location)
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
		mqlBucketCors, err := p.MotorRuntime.CreateResource("aws.s3.bucket.corsrule",
			"name", bucketname,
			"allowedHeaders", corsrule.AllowedHeaders,
			"allowedMethods", corsrule.AllowedMethods,
			"allowedOrigins", corsrule.AllowedOrigins,
			"exposeHeaders", corsrule.ExposeHeaders,
			"maxAgeSeconds", int64(corsrule.MaxAgeSeconds),
		)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlBucketCors)
	}

	return res, nil
}

func (p *mqlAwsS3Bucket) GetLogging() (map[string]interface{}, error) {
	bucketname, err := p.Name()
	if err != nil {
		return nil, err
	}
	bucketlocation, err := p.Location()
	if err != nil {
		return nil, err
	}

	provider, err := awsProvider(p.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	svc := provider.S3(bucketlocation)
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
			res["TargetPrefix"] = core.ToString(logging.LoggingEnabled.TargetPrefix)
		}

		if logging.LoggingEnabled.TargetBucket != nil {
			res["TargetBucket"] = core.ToString(logging.LoggingEnabled.TargetBucket)
		}

		// it is becoming a more complex object similar to aws.s3.bucket.grant
		// if logging.LoggingEnabled.TargetGrants != nil {
		// 	res["TargetGrants"] = *logging.LoggingEnabled.TargetGrants
		// }
	}

	return res, nil
}

func (p *mqlAwsS3Bucket) GetVersioning() (map[string]interface{}, error) {
	bucketname, err := p.Name()
	if err != nil {
		return nil, err
	}
	location, err := p.Location()
	if err != nil {
		return nil, err
	}

	provider, err := awsProvider(p.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	svc := provider.S3(location)
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

func (p *mqlAwsS3Bucket) GetReplication() (interface{}, error) {
	bucketname, err := p.Name()
	if err != nil {
		return nil, err
	}
	region, err := p.Location()
	if err != nil {
		return nil, err
	}

	provider, err := awsProvider(p.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	svc := provider.S3(region)
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
	return core.JsonToDict(bucketReplication.ReplicationConfiguration)
}

func (p *mqlAwsS3Bucket) GetEncryption() (interface{}, error) {
	bucketname, err := p.Name()
	if err != nil {
		return nil, err
	}
	region, err := p.Location()
	if err != nil {
		return nil, err
	}

	provider, err := awsProvider(p.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	svc := provider.S3(region)
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

	return core.JsonToDict(encryption.ServerSideEncryptionConfiguration)
}

func (p *mqlAwsS3Bucket) GetDefaultLock() (string, error) {
	bucketname, err := p.Name()
	if err != nil {
		return "", err
	}
	region, err := p.Location()
	if err != nil {
		return "", err
	}

	provider, err := awsProvider(p.MotorRuntime.Motor.Provider)
	if err != nil {
		return "", err
	}

	svc := provider.S3(region)
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

func (p *mqlAwsS3Bucket) GetStaticWebsiteHosting() (map[string]interface{}, error) {
	bucketname, err := p.Name()
	if err != nil {
		return nil, err
	}
	location, err := p.Location()
	if err != nil {
		return nil, err
	}

	provider, err := awsProvider(p.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	svc := provider.S3(location)
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
			res["ErrorDocument"] = core.ToString(website.ErrorDocument.Key)
		}

		if website.IndexDocument != nil {
			res["IndexDocument"] = core.ToString(website.IndexDocument.Suffix)
		}
	}

	return res, nil
}

func (p *mqlAwsS3BucketGrant) id() (string, error) {
	return p.Id()
}

func (p *mqlAwsS3BucketCorsrule) id() (string, error) {
	name, err := p.Name()
	if err != nil {
		return "", err
	}
	return "s3.bucket.corsrule " + name, nil
}

func (p *mqlAwsS3BucketPolicy) id() (string, error) {
	return p.Name()
}

func (p *mqlAwsS3BucketPolicy) parsePolicyDocument() (*awspolicy.S3BucketPolicy, error) {
	data, err := p.Document()
	if err != nil {
		return nil, err
	}

	var policy awspolicy.S3BucketPolicy
	err = json.Unmarshal([]byte(data), &policy)
	if err != nil {
		return nil, err
	}

	return &policy, nil
}

func (p *mqlAwsS3BucketPolicy) GetVersion() (string, error) {
	policy, err := p.parsePolicyDocument()
	if err != nil {
		return "", err
	}
	return policy.Version, nil
}

func (p *mqlAwsS3BucketPolicy) GetId() (string, error) {
	policy, err := p.parsePolicyDocument()
	if err != nil {
		return "", err
	}
	return policy.Id, nil
}

func (p *mqlAwsS3BucketPolicy) GetStatements() ([]interface{}, error) {
	policy, err := p.parsePolicyDocument()
	if err != nil {
		return nil, err
	}
	return core.JsonToDictSlice(policy.Statements)
}

func isNotFoundForS3(err error) bool {
	if err == nil {
		return false
	}

	var respErr *http.ResponseError
	var notFoundErr *types.NotFound

	if errors.As(err, &notFoundErr) {
		return true
	} else if errors.As(err, &respErr) {
		if respErr.HTTPStatusCode() == 404 {
			return true
		}
	}

	return false
}
