// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"sync"

	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/mql/v13/providers/aws/connection"

	"go.mondoo.com/mql/v13/types"
)

func (a *mqlAwsSes) id() (string, error) {
	return "aws.ses", nil
}

// ---- Account ----

func (a *mqlAwsSes) account() (*mqlAwsSesAccount, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	svc := conn.Sesv2("")
	ctx := context.Background()

	resp, err := svc.GetAccount(ctx, &sesv2.GetAccountInput{})
	if err != nil {
		if Is400AccessDeniedError(err) {
			log.Warn().Msg("access denied querying SES account; returning defaults")
			res, createErr := CreateResource(a.MqlRuntime, ResourceAwsSesAccount,
				map[string]*llx.RawData{
					"sendingEnabled":          llx.BoolData(false),
					"productionAccessEnabled": llx.BoolData(false),
					"dedicatedIpWarmupEnabled": llx.BoolData(false),
					"reputationMetrics":        llx.DictData(nil),
					"suppressionOptions":       llx.DictData(nil),
					"vdmAttributes":            llx.DictData(nil),
				})
			if createErr != nil {
				return nil, createErr
			}
			return res.(*mqlAwsSesAccount), nil
		}
		return nil, err
	}

	var reputationMetrics any
	if resp.SendQuota != nil {
		reputationMetrics, _ = convert.JsonToDict(resp.SendQuota)
	}
	var suppressionOpts any
	if resp.SuppressionAttributes != nil {
		suppressionOpts, _ = convert.JsonToDict(resp.SuppressionAttributes)
	}
	var vdmAttrs any
	if resp.VdmAttributes != nil {
		vdmAttrs, _ = convert.JsonToDict(resp.VdmAttributes)
	}

	res, err := CreateResource(a.MqlRuntime, ResourceAwsSesAccount,
		map[string]*llx.RawData{
			"sendingEnabled":           llx.BoolData(resp.SendingEnabled),
			"productionAccessEnabled":  llx.BoolData(resp.ProductionAccessEnabled),
			"dedicatedIpWarmupEnabled": llx.BoolData(resp.DedicatedIpAutoWarmupEnabled),
			"reputationMetrics":        llx.DictData(reputationMetrics),
			"suppressionOptions":       llx.DictData(suppressionOpts),
			"vdmAttributes":            llx.DictData(vdmAttrs),
		})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAwsSesAccount), nil
}

func (a *mqlAwsSesAccount) id() (string, error) {
	return "aws.ses.account", nil
}

// ---- Identities ----

func (a *mqlAwsSes) identities() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getIdentities(conn), 5)
	poolOfJobs.Run()
	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	for i := range poolOfJobs.Jobs {
		if poolOfJobs.Jobs[i].Result != nil {
			res = append(res, poolOfJobs.Jobs[i].Result.([]any)...)
		}
	}
	return res, nil
}

func (a *mqlAwsSes) getIdentities(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}
	for _, region := range regions {
		region := region
		f := func() (jobpool.JobResult, error) {
			svc := conn.Sesv2(region)
			ctx := context.Background()
			res := []any{}
			paginator := sesv2.NewListEmailIdentitiesPaginator(svc, &sesv2.ListEmailIdentitiesInput{})
			for paginator.HasMorePages() {
				page, err := paginator.NextPage(ctx)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS SES API")
						return res, nil
					}
					return nil, err
				}
				for _, identity := range page.EmailIdentities {
					mqlIdentity, err := CreateResource(a.MqlRuntime, ResourceAwsSesIdentity,
						map[string]*llx.RawData{
							"name":           llx.StringDataPtr(identity.IdentityName),
							"region":         llx.StringData(region),
							"type":           llx.StringData(string(identity.IdentityType)),
							"sendingEnabled": llx.BoolData(identity.SendingEnabled),
							"tags":           llx.MapData(nil, types.String),
						})
					if err != nil {
						return nil, err
					}
					res = append(res, mqlIdentity)
				}
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

type mqlAwsSesIdentityInternal struct {
	fetched        bool
	fetchedDetails *sesv2.GetEmailIdentityOutput
	lock           sync.Mutex
}

func (a *mqlAwsSesIdentity) id() (string, error) {
	return a.Region.Data + "/" + a.Name.Data, nil
}

func (a *mqlAwsSesIdentity) fetchIdentityDetails() (*sesv2.GetEmailIdentityOutput, error) {
	if a.fetched {
		return a.fetchedDetails, nil
	}
	a.lock.Lock()
	defer a.lock.Unlock()
	if a.fetched {
		return a.fetchedDetails, nil
	}
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	svc := conn.Sesv2(a.Region.Data)
	name := a.Name.Data
	resp, err := svc.GetEmailIdentity(context.Background(), &sesv2.GetEmailIdentityInput{EmailIdentity: &name})
	if err != nil {
		return nil, err
	}
	// Update tags from the detail response
	if resp.Tags != nil {
		tags := make(map[string]any, len(resp.Tags))
		for _, t := range resp.Tags {
			if t.Key != nil && t.Value != nil {
				tags[*t.Key] = *t.Value
			}
		}
		a.Tags = plugin.TValue[map[string]any]{Data: tags, State: plugin.StateIsSet}
	}
	a.fetchedDetails = resp
	a.fetched = true
	return resp, nil
}

func (a *mqlAwsSesIdentity) verificationStatus() (string, error) {
	resp, err := a.fetchIdentityDetails()
	if err != nil {
		return "", err
	}
	return string(resp.VerificationStatus), nil
}

func (a *mqlAwsSesIdentity) dkimSigningEnabled() (bool, error) {
	resp, err := a.fetchIdentityDetails()
	if err != nil {
		return false, err
	}
	if resp.DkimAttributes == nil {
		return false, nil
	}
	return resp.DkimAttributes.SigningEnabled, nil
}

func (a *mqlAwsSesIdentity) dkimStatus() (string, error) {
	resp, err := a.fetchIdentityDetails()
	if err != nil {
		return "", err
	}
	if resp.DkimAttributes == nil {
		return "", nil
	}
	return string(resp.DkimAttributes.Status), nil
}

func (a *mqlAwsSesIdentity) dkimTokens() ([]any, error) {
	resp, err := a.fetchIdentityDetails()
	if err != nil {
		return nil, err
	}
	if resp.DkimAttributes == nil {
		return []any{}, nil
	}
	return llx.TArr2Raw(resp.DkimAttributes.Tokens), nil
}

func (a *mqlAwsSesIdentity) mailFromDomain() (string, error) {
	resp, err := a.fetchIdentityDetails()
	if err != nil {
		return "", err
	}
	if resp.MailFromAttributes == nil {
		return "", nil
	}
	return convert.ToValue(resp.MailFromAttributes.MailFromDomain), nil
}

func (a *mqlAwsSesIdentity) mailFromStatus() (string, error) {
	resp, err := a.fetchIdentityDetails()
	if err != nil {
		return "", err
	}
	if resp.MailFromAttributes == nil {
		return "", nil
	}
	return string(resp.MailFromAttributes.MailFromDomainStatus), nil
}

func (a *mqlAwsSesIdentity) feedbackForwardingEnabled() (bool, error) {
	resp, err := a.fetchIdentityDetails()
	if err != nil {
		return false, err
	}
	return resp.FeedbackForwardingStatus, nil
}

func (a *mqlAwsSesIdentity) configurationSetName() (string, error) {
	resp, err := a.fetchIdentityDetails()
	if err != nil {
		return "", err
	}
	return convert.ToValue(resp.ConfigurationSetName), nil
}

func (a *mqlAwsSesIdentity) policies() (map[string]any, error) {
	resp, err := a.fetchIdentityDetails()
	if err != nil {
		return nil, err
	}
	if resp.Policies == nil {
		return map[string]any{}, nil
	}
	result := make(map[string]any, len(resp.Policies))
	for k, v := range resp.Policies {
		result[k] = v
	}
	return result, nil
}

// ---- Configuration Sets ----

func (a *mqlAwsSes) configurationSets() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getConfigurationSets(conn), 5)
	poolOfJobs.Run()
	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	for i := range poolOfJobs.Jobs {
		if poolOfJobs.Jobs[i].Result != nil {
			res = append(res, poolOfJobs.Jobs[i].Result.([]any)...)
		}
	}
	return res, nil
}

func (a *mqlAwsSes) getConfigurationSets(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}
	for _, region := range regions {
		region := region
		f := func() (jobpool.JobResult, error) {
			svc := conn.Sesv2(region)
			ctx := context.Background()
			res := []any{}
			paginator := sesv2.NewListConfigurationSetsPaginator(svc, &sesv2.ListConfigurationSetsInput{})
			for paginator.HasMorePages() {
				page, err := paginator.NextPage(ctx)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS SES API")
						return res, nil
					}
					return nil, err
				}
				for _, cs := range page.ConfigurationSets {
					mqlCS, err := CreateResource(a.MqlRuntime, ResourceAwsSesConfigurationSet,
						map[string]*llx.RawData{
							"name":   llx.StringData(cs),
							"region": llx.StringData(region),
							"tags":   llx.MapData(nil, types.String),
						})
					if err != nil {
						return nil, err
					}
					res = append(res, mqlCS)
				}
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

type mqlAwsSesConfigurationSetInternal struct {
	fetched        bool
	fetchedDetails *sesv2.GetConfigurationSetOutput
	lock           sync.Mutex
}

func (a *mqlAwsSesConfigurationSet) id() (string, error) {
	return a.Region.Data + "/" + a.Name.Data, nil
}

func (a *mqlAwsSesConfigurationSet) fetchConfigSetDetails() (*sesv2.GetConfigurationSetOutput, error) {
	if a.fetched {
		return a.fetchedDetails, nil
	}
	a.lock.Lock()
	defer a.lock.Unlock()
	if a.fetched {
		return a.fetchedDetails, nil
	}
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	svc := conn.Sesv2(a.Region.Data)
	name := a.Name.Data
	resp, err := svc.GetConfigurationSet(context.Background(), &sesv2.GetConfigurationSetInput{ConfigurationSetName: &name})
	if err != nil {
		return nil, err
	}
	if resp.Tags != nil {
		tags := make(map[string]any, len(resp.Tags))
		for _, t := range resp.Tags {
			if t.Key != nil && t.Value != nil {
				tags[*t.Key] = *t.Value
			}
		}
		a.Tags = plugin.TValue[map[string]any]{Data: tags, State: plugin.StateIsSet}
	}
	a.fetchedDetails = resp
	a.fetched = true
	return resp, nil
}

func (a *mqlAwsSesConfigurationSet) eventDestinations() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	svc := conn.Sesv2(a.Region.Data)
	name := a.Name.Data
	resp, err := svc.GetConfigurationSetEventDestinations(context.Background(), &sesv2.GetConfigurationSetEventDestinationsInput{
		ConfigurationSetName: &name,
	})
	if err != nil {
		return nil, err
	}
	return convert.JsonToDictSlice(resp.EventDestinations)
}

func (a *mqlAwsSesConfigurationSet) deliveryOptions() (map[string]any, error) {
	resp, err := a.fetchConfigSetDetails()
	if err != nil {
		return nil, err
	}
	if resp.DeliveryOptions == nil {
		return nil, nil
	}
	return convert.JsonToDict(resp.DeliveryOptions)
}

func (a *mqlAwsSesConfigurationSet) trackingOptions() (map[string]any, error) {
	resp, err := a.fetchConfigSetDetails()
	if err != nil {
		return nil, err
	}
	if resp.TrackingOptions == nil {
		return nil, nil
	}
	return convert.JsonToDict(resp.TrackingOptions)
}

func (a *mqlAwsSesConfigurationSet) reputationOptions() (map[string]any, error) {
	resp, err := a.fetchConfigSetDetails()
	if err != nil {
		return nil, err
	}
	if resp.ReputationOptions == nil {
		return nil, nil
	}
	return convert.JsonToDict(resp.ReputationOptions)
}

func (a *mqlAwsSesConfigurationSet) suppressionOptions() (map[string]any, error) {
	resp, err := a.fetchConfigSetDetails()
	if err != nil {
		return nil, err
	}
	if resp.SuppressionOptions == nil {
		return nil, nil
	}
	return convert.JsonToDict(resp.SuppressionOptions)
}

func (a *mqlAwsSesConfigurationSet) sendingOptions() (map[string]any, error) {
	resp, err := a.fetchConfigSetDetails()
	if err != nil {
		return nil, err
	}
	if resp.SendingOptions == nil {
		return nil, nil
	}
	return convert.JsonToDict(resp.SendingOptions)
}

// ---- Dedicated IP Pools ----

func (a *mqlAwsSes) dedicatedIpPools() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getDedicatedIpPools(conn), 5)
	poolOfJobs.Run()
	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	for i := range poolOfJobs.Jobs {
		if poolOfJobs.Jobs[i].Result != nil {
			res = append(res, poolOfJobs.Jobs[i].Result.([]any)...)
		}
	}
	return res, nil
}

func (a *mqlAwsSes) getDedicatedIpPools(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}
	for _, region := range regions {
		region := region
		f := func() (jobpool.JobResult, error) {
			svc := conn.Sesv2(region)
			ctx := context.Background()
			res := []any{}
			paginator := sesv2.NewListDedicatedIpPoolsPaginator(svc, &sesv2.ListDedicatedIpPoolsInput{})
			for paginator.HasMorePages() {
				page, err := paginator.NextPage(ctx)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS SES API")
						return res, nil
					}
					return nil, err
				}
				for _, poolName := range page.DedicatedIpPools {
					mqlPool, err := CreateResource(a.MqlRuntime, ResourceAwsSesDedicatedIpPool,
						map[string]*llx.RawData{
							"name":   llx.StringData(poolName),
							"region": llx.StringData(region),
						})
					if err != nil {
						return nil, err
					}
					res = append(res, mqlPool)
				}
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func (a *mqlAwsSesDedicatedIpPool) id() (string, error) {
	return a.Region.Data + "/" + a.Name.Data, nil
}

func (a *mqlAwsSesDedicatedIpPool) scalingMode() (string, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	svc := conn.Sesv2(a.Region.Data)
	name := a.Name.Data
	resp, err := svc.GetDedicatedIpPool(context.Background(), &sesv2.GetDedicatedIpPoolInput{PoolName: &name})
	if err != nil {
		return "", err
	}
	if resp.DedicatedIpPool == nil {
		return "", nil
	}
	return string(resp.DedicatedIpPool.ScalingMode), nil
}

func (a *mqlAwsSesDedicatedIpPool) ips() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	svc := conn.Sesv2(a.Region.Data)
	name := a.Name.Data
	ctx := context.Background()
	paginator := sesv2.NewGetDedicatedIpsPaginator(svc, &sesv2.GetDedicatedIpsInput{PoolName: &name})
	var ips []any
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, ip := range page.DedicatedIps {
			d, err := convert.JsonToDict(ip)
			if err != nil {
				return nil, err
			}
			ips = append(ips, d)
		}
	}
	if ips == nil {
		ips = []any{}
	}
	return ips, nil
}

// ---- Templates ----

func (a *mqlAwsSes) templates() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getTemplates(conn), 5)
	poolOfJobs.Run()
	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	for i := range poolOfJobs.Jobs {
		if poolOfJobs.Jobs[i].Result != nil {
			res = append(res, poolOfJobs.Jobs[i].Result.([]any)...)
		}
	}
	return res, nil
}

func (a *mqlAwsSes) getTemplates(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}
	for _, region := range regions {
		region := region
		f := func() (jobpool.JobResult, error) {
			svc := conn.Sesv2(region)
			ctx := context.Background()
			res := []any{}
			paginator := sesv2.NewListEmailTemplatesPaginator(svc, &sesv2.ListEmailTemplatesInput{})
			for paginator.HasMorePages() {
				page, err := paginator.NextPage(ctx)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS SES API")
						return res, nil
					}
					return nil, err
				}
				for _, tmpl := range page.TemplatesMetadata {
					mqlTmpl, err := CreateResource(a.MqlRuntime, ResourceAwsSesTemplate,
						map[string]*llx.RawData{
							"name":      llx.StringDataPtr(tmpl.TemplateName),
							"region":    llx.StringData(region),
							"createdAt": llx.TimeDataPtr(tmpl.CreatedTimestamp),
						})
					if err != nil {
						return nil, err
					}
					res = append(res, mqlTmpl)
				}
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

type mqlAwsSesTemplateInternal struct {
	fetched        bool
	fetchedDetails *sesv2.GetEmailTemplateOutput
	lock           sync.Mutex
}

func (a *mqlAwsSesTemplate) id() (string, error) {
	return a.Region.Data + "/" + a.Name.Data, nil
}

func (a *mqlAwsSesTemplate) fetchTemplateDetails() (*sesv2.GetEmailTemplateOutput, error) {
	if a.fetched {
		return a.fetchedDetails, nil
	}
	a.lock.Lock()
	defer a.lock.Unlock()
	if a.fetched {
		return a.fetchedDetails, nil
	}
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	svc := conn.Sesv2(a.Region.Data)
	name := a.Name.Data
	resp, err := svc.GetEmailTemplate(context.Background(), &sesv2.GetEmailTemplateInput{TemplateName: &name})
	if err != nil {
		return nil, err
	}
	a.fetchedDetails = resp
	a.fetched = true
	return resp, nil
}

func (a *mqlAwsSesTemplate) subject() (string, error) {
	resp, err := a.fetchTemplateDetails()
	if err != nil {
		return "", err
	}
	if resp.TemplateContent == nil {
		return "", nil
	}
	return convert.ToValue(resp.TemplateContent.Subject), nil
}

func (a *mqlAwsSesTemplate) htmlBody() (string, error) {
	resp, err := a.fetchTemplateDetails()
	if err != nil {
		return "", err
	}
	if resp.TemplateContent == nil {
		return "", nil
	}
	return convert.ToValue(resp.TemplateContent.Html), nil
}

func (a *mqlAwsSesTemplate) textBody() (string, error) {
	resp, err := a.fetchTemplateDetails()
	if err != nil {
		return "", err
	}
	if resp.TemplateContent == nil {
		return "", nil
	}
	return convert.ToValue(resp.TemplateContent.Text), nil
}
