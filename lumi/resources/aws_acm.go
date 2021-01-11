package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/acm"
	"go.mondoo.io/mondoo/lumi/library/jobpool"
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
				certs, err := svc.ListCertificatesRequest(params).Send(ctx)
				if err != nil {
					return nil, err
				}
				for _, cert := range certs.CertificateSummaryList {
					certDetails, err := svc.DescribeCertificateRequest(&acm.DescribeCertificateInput{CertificateArn: cert.CertificateArn}).Send(ctx)
					if err != nil {
						return nil, err
					}
					stringStatus, err := certDetails.Certificate.Status.MarshalValue()
					if err != nil {
						return nil, err
					}
					lumiCert, err := a.Runtime.CreateResource("aws.acm.certificate",
						"arn", toString(certDetails.Certificate.CertificateArn),
						"notBefore", certDetails.Certificate.NotBefore,
						"notAfter", certDetails.Certificate.NotAfter,
						"createdAt", certDetails.Certificate.CreatedAt,
						"domainName", toString(certDetails.Certificate.DomainName),
						"status", stringStatus,
						"subject", toString(certDetails.Certificate.Subject),
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

func (e *lumiAwsAcmCertificate) id() (string, error) {
	return e.Arn()
}
