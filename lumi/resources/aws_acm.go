package resources

import (
	"context"
	"errors"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/acm"
	"go.mondoo.io/mondoo/lumi"
	"go.mondoo.io/mondoo/lumi/library/jobpool"
	"go.mondoo.io/mondoo/lumi/resources/certificates"
)

func (a *lumiAwsAcm) id() (string, error) {
	return "aws.acm", nil
}

func (a *lumiAwsAcm) GetCertificates() ([]interface{}, error) {
	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(a.getCertificates(), 5)
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

func (a *lumiAwsAcm) getCertificates() []*jobpool.Job {
	var tasks = make([]*jobpool.Job, 0)
	at, err := awstransport(a.Runtime.Motor.Transport)
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}
	regions, err := at.GetRegions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		regionVal := region
		f := func() (jobpool.JobResult, error) {

			svc := at.Acm(regionVal)
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
					lumiCert, err := a.Runtime.CreateResource("aws.acm.certificate",
						"arn", toString(cert.CertificateArn),
					)
					if err != nil {
						return nil, err
					}
					res = append(res, lumiCert)
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

func (a *lumiAwsAcmCertificate) id() (string, error) {
	return a.Arn()
}

func (a *lumiAwsAcmCertificate) init(args *lumi.Args) (*lumi.Args, AwsAcmCertificate, error) {
	if len(*args) > 2 {
		return args, nil, nil
	}

	if (*args)["arn"] == nil {
		return nil, nil, errors.New("arn required to fetch aws acm certificate")
	}

	arnVal := (*args)["arn"].(string)
	region, err := getRegionFromArn(arnVal)
	if err != nil {
		return args, nil, nil
	}
	at, err := awstransport(a.Runtime.Motor.Transport)
	if err != nil {
		return nil, nil, err
	}
	svc := at.Acm(region)
	ctx := context.Background()
	certDetails, err := svc.DescribeCertificate(ctx, &acm.DescribeCertificateInput{CertificateArn: &arnVal})
	if err != nil {
		return nil, nil, err
	}

	(*args)["arn"] = arnVal
	(*args)["notBefore"] = certDetails.Certificate.NotBefore
	(*args)["notAfter"] = certDetails.Certificate.NotAfter
	(*args)["createdAt"] = certDetails.Certificate.CreatedAt
	(*args)["domainName"] = toString(certDetails.Certificate.DomainName)
	(*args)["status"] = string(certDetails.Certificate.Status)
	(*args)["subject"] = toString(certDetails.Certificate.Subject)
	return args, nil, nil
}

func (a *lumiAwsAcmCertificate) GetCertificate() (interface{}, error) {
	certArn, err := a.Arn()
	if err != nil {
		return false, err
	}
	region, err := getRegionFromArn(certArn)
	if err != nil {
		return false, err
	}
	at, err := awstransport(a.Runtime.Motor.Transport)
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
	parsedCert, err := certificates.ParseCertFromPEM(strings.NewReader(toString(cert.Certificate)))
	if err != nil {
		return nil, err
	}
	lumiCerts, err := certificatesToLumiCertificates(a.Runtime, parsedCert)
	if err != nil {
		return nil, err
	}
	if len(lumiCerts) == 1 {
		return lumiCerts[0], nil
	}
	return nil, nil
}
