package resources

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws/endpoints"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/rs/zerolog/log"

	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func s3client(region string) *s3.Client {
	// TODO: cfg needs to come from the transport
	cfg, err := external.LoadDefaultAWSConfig(external.WithSharedConfigProfile("mondoo-inc"))
	// cfg, err := external.LoadDefaultAWSConfig()
	if err != nil {
		panic(err)
	}

	if region == "" {
		cfg.Region = endpoints.UsEast1RegionID
	} else {
		// NOTE: for s3 buckets, we need to switch the region to gather the policy documents
		cfg.Region = region
	}

	// iterate over each region?
	svc := s3.New(cfg)
	return svc
}

func (p *lumiAwsS3) id() (string, error) {
	return "aws.s3", nil
}

func (p *lumiAwsS3) GetBuckets() ([]interface{}, error) {
	ctx := context.Background()
	svc := s3client("")
	buckets, err := svc.ListBucketsRequest(&s3.ListBucketsInput{}).Send(ctx)
	if err != nil {
		return nil, err
	}

	res := []interface{}{}
	for i := range buckets.Buckets {
		bucket := buckets.Buckets[i]

		lumiS3Bucket, err := p.Runtime.CreateResource("aws.s3.bucket",
			"name", toString(bucket.Name),
		)
		if err != nil {
			return nil, err
		}
		res = append(res, lumiS3Bucket)
	}

	return res, nil
}

func (p *lumiAwsS3Bucket) id() (string, error) {
	// assumes bucket names are globally unique, which they are right now
	return p.Name()
}

func (p *lumiAwsS3Bucket) GetPolicy() (string, error) {
	bucketname, err := p.Name()
	if err != nil {
		return "", err
	}

	location, err := p.Location()
	if err != nil {
		return "", err
	}

	ctx := context.Background()
	svc := s3client(location)

	policy, err := svc.GetBucketPolicyRequest(&s3.GetBucketPolicyInput{
		Bucket: &bucketname,
	}).Send(ctx)
	isAwsErr, code := IsAwsCode(err)
	// aws code NoSuchBucketPolicy in case no policy exists
	if err != nil && isAwsErr && code == "NoSuchBucketPolicy" {
		return "", nil
	} else if err != nil {
		log.Error().Err(err).Msg("could not retrieve bucket policy")
		return "", err
	}

	if policy != nil && policy.Policy != nil {
		return *policy.Policy, nil
	}
	return "", nil
}

func (p *lumiAwsS3Bucket) GetTags() (map[string]interface{}, error) {
	bucketname, err := p.Name()
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	svc := s3client("")

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

	ctx := context.Background()
	svc := s3client("")

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

	ctx := context.Background()
	svc := s3client("")

	acl, err := svc.GetBucketAclRequest(&s3.GetBucketAclInput{
		Bucket: &bucketname,
	}).Send(ctx)

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

	ctx := context.Background()
	svc := s3client("")

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
