// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go-v2/service/acm"
	acmtypes "github.com/aws/aws-sdk-go-v2/service/acm/types"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/cnquery/providers/aws/connection"

	"go.mondoo.com/cnquery/types"
)

func (a *mqlAwsAcm) id() (string, error) {
	return "aws.acm", nil
}

func (a *mqlAwsAcm) certificates() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(a.getCertificates(conn), 5)
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

func (a *mqlAwsAcm) getCertificates(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		regionVal := region
		f := func() (jobpool.JobResult, error) {
			svc := conn.Acm(regionVal)
			ctx := context.Background()
			res := []interface{}{}

			nextToken := aws.String("no_token_to_start_with")
			params := &acm.ListCertificatesInput{}
			for nextToken != nil {
				certs, err := svc.ListCertificates(ctx, params)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", regionVal).Msg("error accessing region for AWS API")
						return res, nil
					}
					return nil, err
				}
				for _, cert := range certs.CertificateSummaryList {
					mqlCert, err := a.MqlRuntime.CreateResource(a.MqlRuntime, "aws.acm.certificate",
						map[string]*llx.RawData{
							"arn": llx.StringData(convert.ToString(cert.CertificateArn)),
						})
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
	return a.Arn.Data, nil
}

func initAwsAcmCertificate(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}

	if args["arn"] == nil {
		return nil, nil, errors.New("arn required to fetch aws acm certificate")
	}

	arnVal := args["arn"].Value.(string)
	region, err := GetRegionFromArn(arnVal)
	if err != nil {
		return args, nil, nil
	}
	conn := runtime.Connection.(*connection.AwsConnection)

	svc := conn.Acm(region)
	ctx := context.Background()
	certDetails, err := svc.DescribeCertificate(ctx, &acm.DescribeCertificateInput{CertificateArn: &arnVal})
	if err != nil {
		return nil, nil, err
	}
	certTags, err := svc.ListTagsForCertificate(ctx, &acm.ListTagsForCertificateInput{CertificateArn: &arnVal})
	if err != nil {
		return nil, nil, err
	}

	args["arn"] = llx.StringData(arnVal)
	args["notBefore"] = llx.TimeData(toTime(certDetails.Certificate.NotBefore))
	args["notAfter"] = llx.TimeData(toTime(certDetails.Certificate.NotAfter))
	args["createdAt"] = llx.TimeData(toTime(certDetails.Certificate.CreatedAt))
	args["domainName"] = llx.StringData(convert.ToString(certDetails.Certificate.DomainName))
	args["status"] = llx.StringData(string(certDetails.Certificate.Status))
	args["subject"] = llx.StringData(convert.ToString(certDetails.Certificate.Subject))
	args["tags"] = llx.MapData(CertTagsToMapTags(certTags.Tags), types.String)
	return args, nil, nil
}

func CertTagsToMapTags(tags []acmtypes.Tag) map[string]interface{} {
	mapTags := make(map[string]interface{})
	for i := range tags {
		if tags[i].Key != nil && tags[i].Value != nil {
			mapTags[*tags[i].Key] = *tags[i].Value
		}
	}
	return mapTags
}

// func (a *mqlAwsAcmCertificate) certificate() (*mqlCertificate, error) {
// 	certArn := a.Arn.Data
// 	region, err := GetRegionFromArn(certArn)
// 	if err != nil {
// 		return nil, err
// 	}
// 	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
// 	svc := conn.Acm(region)
// 	ctx := context.Background()
// 	cert, err := svc.GetCertificate(ctx, &acm.GetCertificateInput{CertificateArn: &certArn})
// 	if err != nil {
// 		return nil, err
// 	}
// 	if cert.Certificate == nil {
// 		return nil, nil
// 	}
// 	parsedCert, err := ParseCertsFromPEM(strings.NewReader(convert.ToString(cert.Certificate)))
// 	if err != nil {
// 		return nil, err
// 	}
// 	mqlCerts, err := CertificatesToMqlCertificates(a.MqlRuntime, parsedCert)
// 	if err != nil {
// 		return nil, err
// 	}
// 	if len(mqlCerts) == 1 {
// 		return mqlCerts[0].(*mqlCertificate), nil
// 	}
// 	return nil, nil
// }
