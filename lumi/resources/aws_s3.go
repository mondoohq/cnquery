package resources

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/lumi"
	"go.mondoo.io/mondoo/lumi/resources/awspolicy"

	"github.com/aws/aws-sdk-go-v2/service/s3"
)

const (
	s3ArnPattern = "arn:aws:s3:::%s"
)

func (p *lumiAwsS3) id() (string, error) {
	return "aws.s3", nil
}

func (p *lumiAwsS3) GetBuckets() ([]interface{}, error) {
	at, err := awstransport(p.Runtime.Motor.Transport)
	if err != nil {
		return nil, err
	}

	svc := at.S3("")
	ctx := context.Background()

	buckets, err := svc.ListBucketsRequest(&s3.ListBucketsInput{}).Send(ctx)
	if err != nil {
		return nil, err
	}

	res := []interface{}{}
	for i := range buckets.Buckets {
		bucket := buckets.Buckets[i]

		lumiS3Bucket, err := p.Runtime.CreateResource("aws.s3.bucket",
			"name", toString(bucket.Name),
			"arn", fmt.Sprintf(s3ArnPattern, toString(bucket.Name)),
		)
		if err != nil {
			return nil, err
		}
		res = append(res, lumiS3Bucket)
	}

	return res, nil
}

func (p *lumiAwsS3Bucket) init(args *lumi.Args) (*lumi.Args, AwsS3Bucket, error) {
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
	obj, err := p.Runtime.CreateResource("aws.s3")
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
		lumiBucketArn, err := bucket.Arn()
		if err != nil {
			return nil, nil, err
		}
		if lumiBucketArn == arn {
			return args, bucket, nil
		}
	}

	return nil, nil, errors.New("aws s3 bucket does not exist")
}

func (p *lumiAwsS3Bucket) id() (string, error) {
	// assumes bucket names are globally unique, which they are right now
	return p.Arn()
}

func (p *lumiAwsS3Bucket) GetPolicy() (interface{}, error) {
	bucketname, err := p.Name()
	if err != nil {
		return nil, err
	}

	location, err := p.Location()
	if err != nil {
		return nil, err
	}

	at, err := awstransport(p.Runtime.Motor.Transport)
	if err != nil {
		return nil, err
	}

	svc := at.S3(location)
	ctx := context.Background()

	policy, err := svc.GetBucketPolicyRequest(&s3.GetBucketPolicyInput{
		Bucket: &bucketname,
	}).Send(ctx)
	isAwsErr, code := IsAwsCode(err)
	// aws code NoSuchBucketPolicy in case no policy exists
	if err != nil && isAwsErr && code == "NoSuchBucketPolicy" {
		return nil, errors.New("the specified bucket does not have a bucket policy")
	} else if err != nil {
		log.Error().Err(err).Msg("could not retrieve bucket policy")
		return nil, err
	}

	if policy != nil && policy.Policy != nil {
		// create the plicy resource
		lumiS3BucketPolicy, err := p.Runtime.CreateResource("aws.s3.bucket.policy",
			"name", bucketname,
			"document", toString(policy.Policy),
		)
		if err != nil {
			return nil, err
		}
		return lumiS3BucketPolicy, nil
	}

	// no bucket policy found
	return nil, errors.New("the specified bucket does not have a bucket policy")
}

func (p *lumiAwsS3Bucket) GetTags() (map[string]interface{}, error) {
	bucketname, err := p.Name()
	if err != nil {
		return nil, err
	}
	location, err := p.Location()
	if err != nil {
		return nil, err
	}

	at, err := awstransport(p.Runtime.Motor.Transport)
	if err != nil {
		return nil, err
	}

	svc := at.S3(location)
	ctx := context.Background()

	tags, err := svc.GetBucketTaggingRequest(&s3.GetBucketTaggingInput{
		Bucket: &bucketname,
	}).Send(ctx)
	isAwsErr, code := IsAwsCode(err)
	// aws code NoSuchTagSetError in case no tag is set
	if err != nil && isAwsErr && code == "NoSuchTagSetError" {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	res := map[string]interface{}{}
	for i := range tags.TagSet {
		tag := tags.TagSet[i]
		res[toString(tag.Key)] = toString(tag.Value)
	}

	return res, nil
}

func (p *lumiAwsS3Bucket) GetLocation() (string, error) {
	bucketname, err := p.Name()
	if err != nil {
		return "", err
	}

	at, err := awstransport(p.Runtime.Motor.Transport)
	if err != nil {
		return "", err
	}

	svc := at.S3("")
	ctx := context.Background()

	location, err := svc.GetBucketLocationRequest(&s3.GetBucketLocationInput{
		Bucket: &bucketname,
	}).Send(ctx)
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

func (p *lumiAwsS3Bucket) gatherAcl() (*s3.GetBucketAclOutput, error) {
	bucketname, err := p.Name()
	if err != nil {
		return nil, err
	}
	location, err := p.Location()
	if err != nil {
		return nil, err
	}

	at, err := awstransport(p.Runtime.Motor.Transport)
	if err != nil {
		return nil, err
	}

	svc := at.S3(location)
	ctx := context.Background()

	acl, err := svc.GetBucketAclRequest(&s3.GetBucketAclInput{
		Bucket: &bucketname,
	}).Send(ctx)
	if err != nil {
		return nil, err
	}

	// TODO: store in cache
	return acl.GetBucketAclOutput, nil
}

func (p *lumiAwsS3Bucket) GetAcl() ([]interface{}, error) {
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
		grantee["id"] = toString(grant.Grantee.ID)
		grantee["name"] = toString(grant.Grantee.DisplayName)
		grantee["emailAddress"] = toString(grant.Grantee.EmailAddress)
		grantee["type"] = string(grant.Grantee.Type)
		grantee["uri"] = toString(grant.Grantee.URI)

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

		lumiBucketGrant, err := p.Runtime.CreateResource("aws.s3.bucket.grant",
			"id", id,
			"name", bucketname,
			"permission", string(grant.Permission),
			"grantee", grantee,
		)
		if err != nil {
			return nil, err
		}
		res = append(res, lumiBucketGrant)
	}
	return res, nil
}

func (p *lumiAwsS3Bucket) GetPublicAccessBlock() (interface{}, error) {
	bucketname, err := p.Name()
	if err != nil {
		return nil, err
	}
	location, err := p.Location()
	if err != nil {
		return nil, err
	}
	at, err := awstransport(p.Runtime.Motor.Transport)
	if err != nil {
		return nil, err
	}

	svc := at.S3(location)
	ctx := context.Background()

	publicAccessBlock, err := svc.GetPublicAccessBlockRequest(&s3.GetPublicAccessBlockInput{
		Bucket: &bucketname,
	}).Send(ctx)
	isAwsErr, code := IsAwsCode(err)
	if err != nil && isAwsErr && code == "NoSuchPublicAccessBlockConfiguration" {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	return jsonToDict(publicAccessBlock.PublicAccessBlockConfiguration)
}

func (p *lumiAwsS3Bucket) GetOwner() (map[string]interface{}, error) {
	acl, err := p.gatherAcl()
	if err != nil {
		return nil, err
	}

	if acl.Owner == nil {
		return nil, errors.New("could not gather aws s3 bucket's owner information")
	}

	res := map[string]interface{}{}
	res["id"] = acl.Owner.ID
	res["name"] = acl.Owner.DisplayName

	return res, nil
}

// see https://docs.aws.amazon.com/AmazonS3/latest/dev/acl-overview.html
const s3AuthenticatedUsersGroup = "http://acs.amazonaws.com/groups/global/AuthenticatedUsers"
const s3AllUsersGroup = "http://acs.amazonaws.com/groups/global/AllUsers"

func (p *lumiAwsS3Bucket) GetPublic() (bool, error) {
	acl, err := p.gatherAcl()
	if err != nil {
		return false, err
	}

	for i := range acl.Grants {
		grant := acl.Grants[i]
		if grant.Grantee.Type == s3.TypeGroup && (toString(grant.Grantee.URI) == s3AuthenticatedUsersGroup || toString(grant.Grantee.URI) == s3AllUsersGroup) {
			return true, nil
		}
	}
	return false, nil
}

func (p *lumiAwsS3Bucket) GetCors() ([]interface{}, error) {
	bucketname, err := p.Name()
	if err != nil {
		return nil, err
	}
	location, err := p.Location()
	if err != nil {
		return nil, err
	}

	at, err := awstransport(p.Runtime.Motor.Transport)
	if err != nil {
		return nil, err
	}

	svc := at.S3(location)
	ctx := context.Background()

	cors, err := svc.GetBucketCorsRequest(&s3.GetBucketCorsInput{
		Bucket: &bucketname,
	}).Send(ctx)

	isAwsErr, code := IsAwsCode(err)
	// aws code NoSuchTagSetError in case no tag is set
	if err != nil && isAwsErr && code == "NoSuchCORSConfiguration" {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	res := []interface{}{}
	for i := range cors.CORSRules {
		corsrule := cors.CORSRules[i]
		lumiBucketCors, err := p.Runtime.CreateResource("aws.s3.bucket.corsrule",
			"name", bucketname,
			"allowedHeaders", corsrule.AllowedHeaders,
			"allowedMethods", corsrule.AllowedMethods,
			"allowedOrigins", corsrule.AllowedOrigins,
			"exposeHeaders", corsrule.ExposeHeaders,
			"maxAgeSeconds", toInt64(corsrule.MaxAgeSeconds),
		)
		if err != nil {
			return nil, err
		}
		res = append(res, lumiBucketCors)
	}

	return res, nil
}

func (p *lumiAwsS3Bucket) GetLogging() (map[string]interface{}, error) {
	bucketname, err := p.Name()
	if err != nil {
		return nil, err
	}
	bucketlocation, err := p.Location()
	if err != nil {
		return nil, err
	}

	at, err := awstransport(p.Runtime.Motor.Transport)
	if err != nil {
		return nil, err
	}

	svc := at.S3(bucketlocation)
	ctx := context.Background()

	logging, err := svc.GetBucketLoggingRequest(&s3.GetBucketLoggingInput{
		Bucket: &bucketname,
	}).Send(ctx)
	if err != nil {
		return nil, err
	}

	res := map[string]interface{}{}

	if logging != nil && logging.LoggingEnabled != nil {
		if logging.LoggingEnabled.TargetPrefix != nil {
			res["TargetPrefix"] = *logging.LoggingEnabled.TargetPrefix
		}

		if logging.LoggingEnabled.TargetBucket != nil {
			res["TargetBucket"] = *logging.LoggingEnabled.TargetBucket
		}

		// it is becoming a more complex object similar to aws.s3.bucket.grant
		// if logging.LoggingEnabled.TargetGrants != nil {
		// 	res["TargetGrants"] = *logging.LoggingEnabled.TargetGrants
		// }
	}

	return res, nil
}

func (p *lumiAwsS3Bucket) GetVersioning() (map[string]interface{}, error) {
	bucketname, err := p.Name()
	if err != nil {
		return nil, err
	}
	location, err := p.Location()
	if err != nil {
		return nil, err
	}

	at, err := awstransport(p.Runtime.Motor.Transport)
	if err != nil {
		return nil, err
	}

	svc := at.S3(location)
	ctx := context.Background()

	versioning, err := svc.GetBucketVersioningRequest(&s3.GetBucketVersioningInput{
		Bucket: &bucketname,
	}).Send(ctx)
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

func (p *lumiAwsS3Bucket) GetReplication() (interface{}, error) {
	bucketname, err := p.Name()
	if err != nil {
		return nil, err
	}
	region, err := p.Location()
	if err != nil {
		return nil, err
	}

	at, err := awstransport(p.Runtime.Motor.Transport)
	if err != nil {
		return nil, err
	}

	svc := at.S3(region)
	ctx := context.Background()

	bucketReplication, err := svc.GetBucketReplicationRequest(&s3.GetBucketReplicationInput{
		Bucket: &bucketname,
	}).Send(ctx)
	isAwsErr, code := IsAwsCode(err)
	if err != nil && isAwsErr && code == "ReplicationConfigurationNotFoundError" {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	return jsonToDict(bucketReplication.ReplicationConfiguration)
}

func (p *lumiAwsS3Bucket) GetEncryption() (interface{}, error) {
	bucketname, err := p.Name()
	if err != nil {
		return nil, err
	}
	region, err := p.Location()
	if err != nil {
		return nil, err
	}

	at, err := awstransport(p.Runtime.Motor.Transport)
	if err != nil {
		return nil, err
	}

	svc := at.S3(region)
	ctx := context.Background()

	encryption, err := svc.GetBucketEncryptionRequest(&s3.GetBucketEncryptionInput{
		Bucket: &bucketname,
	}).Send(ctx)
	isAwsErr, code := IsAwsCode(err)
	if err != nil && isAwsErr && code == "ServerSideEncryptionConfigurationNotFoundError" {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	return jsonToDict(encryption.ServerSideEncryptionConfiguration)
}

func (p *lumiAwsS3Bucket) GetDefaultLock() (string, error) {
	bucketname, err := p.Name()
	if err != nil {
		return "", err
	}
	region, err := p.Location()
	if err != nil {
		return "", err
	}

	at, err := awstransport(p.Runtime.Motor.Transport)
	if err != nil {
		return "", err
	}

	svc := at.S3(region)
	ctx := context.Background()

	objectLockConfiguration, err := svc.GetObjectLockConfigurationRequest(&s3.GetObjectLockConfigurationInput{
		Bucket: &bucketname,
	}).Send(ctx)
	isAwsErr, code := IsAwsCode(err)
	if err != nil && isAwsErr && code == "ObjectLockConfigurationNotFoundError" {
		return "", nil
	} else if err != nil {
		return "", err
	}

	lockEnabled, err := objectLockConfiguration.ObjectLockConfiguration.ObjectLockEnabled.MarshalValue()
	if err != nil {
		return "", err
	}
	return lockEnabled, nil
}

func (p *lumiAwsS3Bucket) GetStaticWebsiteHosting() (map[string]interface{}, error) {
	bucketname, err := p.Name()
	if err != nil {
		return nil, err
	}
	location, err := p.Location()
	if err != nil {
		return nil, err
	}

	at, err := awstransport(p.Runtime.Motor.Transport)
	if err != nil {
		return nil, err
	}

	svc := at.S3(location)
	ctx := context.Background()

	website, err := svc.GetBucketWebsiteRequest(&s3.GetBucketWebsiteInput{
		Bucket: &bucketname,
	}).Send(ctx)
	if err != nil {
		return nil, err
	}

	res := map[string]interface{}{}

	if website != nil {
		if website.ErrorDocument != nil {
			res["ErrorDocument"] = toString(website.ErrorDocument.Key)
		}

		if website.IndexDocument != nil {
			res["IndexDocument"] = toString(website.IndexDocument.Suffix)
		}
	}

	return res, nil
}

func (p *lumiAwsS3BucketGrant) id() (string, error) {
	return p.Id()
}

func (p *lumiAwsS3BucketCorsrule) id() (string, error) {
	name, err := p.Name()
	if err != nil {
		return "", err
	}
	return "s3.bucket.corsrule " + name, nil
}

func (p *lumiAwsS3BucketPolicy) id() (string, error) {
	return p.Name()
}

func (p *lumiAwsS3BucketPolicy) parsePolicyDocument() (*awspolicy.S3BucketPolicy, error) {
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

func (p *lumiAwsS3BucketPolicy) GetVersion() (string, error) {
	policy, err := p.parsePolicyDocument()
	if err != nil {
		return "", err
	}
	return policy.Version, nil
}

func (p *lumiAwsS3BucketPolicy) GetId() (string, error) {
	policy, err := p.parsePolicyDocument()
	if err != nil {
		return "", err
	}
	return policy.Id, nil
}

func (p *lumiAwsS3BucketPolicy) GetStatements() ([]interface{}, error) {
	policy, err := p.parsePolicyDocument()
	if err != nil {
		return nil, err
	}
	return jsonToDictSlice(policy.Statements)
}
