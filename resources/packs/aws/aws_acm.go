package aws

import (
	"context"
	"errors"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/acm"
	"github.com/aws/aws-sdk-go-v2/service/acm/types"
	aws_provider "go.mondoo.io/mondoo/motor/providers/aws"
	"go.mondoo.io/mondoo/resources"
	"go.mondoo.io/mondoo/resources/library/jobpool"
	"go.mondoo.io/mondoo/resources/packs/core"
	"go.mondoo.io/mondoo/resources/packs/core/certificates"
)

func (a *mqlAwsAcm) id() (string, error) {
	return "aws.acm", nil
}

func (a *mqlAwsAcm) GetCertificates() ([]interface{}, error) {
	at, err := awsProvider(a.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(a.getCertificates(at), 5)
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

func (a *mqlAwsAcm) getCertificates(provider *aws_provider.Provider) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := provider.GetRegions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		regionVal := region
		f := func() (jobpool.JobResult, error) {
			svc := provider.Acm(regionVal)
			ctx := context.Background()
			res := []interface{}{}

			nextToken := aws.String("no_token_to_start_with")
			params := &acm.ListCertificatesInput{}
			for nextToken != nil {
				certs, err := svc.ListCertificates(ctx, params)
				if err != nil {
					return nil, err
				}
				for _, cert := range certs.CertificateSummaryList {
					mqlCert, err := a.MotorRuntime.CreateResource("aws.acm.certificate",
						"arn", core.ToString(cert.CertificateArn),
					)
					if err != nil {
						return nil, err
					}
					res = append(res, mqlCert)
				}
				nextToken = certs.NextToken
				if certs.NextToken != nil {
					params.NextToken = nextToken
				}
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func (a *mqlAwsAcmCertificate) id() (string, error) {
	return a.Arn()
}

func (a *mqlAwsAcmCertificate) init(args *resources.Args) (*resources.Args, AwsAcmCertificate, error) {
	if len(*args) > 2 {
		return args, nil, nil
	}

	if (*args)["arn"] == nil {
		return nil, nil, errors.New("arn required to fetch aws acm certificate")
	}

	arnVal := (*args)["arn"].(string)
	region, err := GetRegionFromArn(arnVal)
	if err != nil {
		return args, nil, nil
	}
	at, err := awsProvider(a.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, nil, err
	}
	svc := at.Acm(region)
	ctx := context.Background()
	certDetails, err := svc.DescribeCertificate(ctx, &acm.DescribeCertificateInput{CertificateArn: &arnVal})
	if err != nil {
		return nil, nil, err
	}
	certTags, err := svc.ListTagsForCertificate(ctx, &acm.ListTagsForCertificateInput{CertificateArn: &arnVal})
	if err != nil {
		return nil, nil, err
	}

	(*args)["arn"] = arnVal
	(*args)["notBefore"] = certDetails.Certificate.NotBefore
	(*args)["notAfter"] = certDetails.Certificate.NotAfter
	(*args)["createdAt"] = certDetails.Certificate.CreatedAt
	(*args)["domainName"] = core.ToString(certDetails.Certificate.DomainName)
	(*args)["status"] = string(certDetails.Certificate.Status)
	(*args)["subject"] = core.ToString(certDetails.Certificate.Subject)
	(*args)["tags"] = CertTagsToMapTags(certTags.Tags)
	return args, nil, nil
}

func CertTagsToMapTags(tags []types.Tag) map[string]interface{} {
	mapTags := make(map[string]interface{})
	for i := range tags {
		if tags[i].Key != nil && tags[i].Value != nil {
			mapTags[*tags[i].Key] = *tags[i].Value
		}
	}
	return mapTags
}

func (a *mqlAwsAcmCertificate) GetCertificate() (interface{}, error) {
	certArn, err := a.Arn()
	if err != nil {
		return false, err
	}
	region, err := GetRegionFromArn(certArn)
	if err != nil {
		return false, err
	}
	at, err := awsProvider(a.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	svc := at.Acm(region)
	ctx := context.Background()
	cert, err := svc.GetCertificate(ctx, &acm.GetCertificateInput{CertificateArn: &certArn})
	if err != nil {
		return nil, err
	}
	if cert.Certificate == nil {
		return nil, nil
	}
	parsedCert, err := certificates.ParseCertFromPEM(strings.NewReader(core.ToString(cert.Certificate)))
	if err != nil {
		return nil, err
	}
	mqlCerts, err := core.CertificatesToMqlCertificates(a.MotorRuntime, parsedCert)
	if err != nil {
		return nil, err
	}
	if len(mqlCerts) == 1 {
		return mqlCerts[0], nil
	}
	return nil, nil
}
