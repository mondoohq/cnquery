// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/route53"
	route53types "github.com/aws/aws-sdk-go-v2/service/route53/types"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/aws/connection"
	"go.mondoo.com/mql/v13/types"
)

// aws.route53
func (a *mqlAwsRoute53) id() (string, error) {
	return "aws.route53", nil
}

func (a *mqlAwsRoute53) hostedZones() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	ctx := context.Background()
	svc := conn.Route53("")

	// Collect all hosted zones first
	var allZones []route53types.HostedZone
	paginator := route53.NewListHostedZonesPaginator(svc, &route53.ListHostedZonesInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			if Is400AccessDeniedError(err) {
				log.Warn().Msg("error accessing Route 53 API")
				return []interface{}{}, nil
			}
			return nil, err
		}
		allZones = append(allZones, page.HostedZones...)
	}

	// Batch-fetch tags (up to 10 per API call)
	tagsByID := batchFetchTags(ctx, svc, route53types.TagResourceTypeHostedzone, allZones, func(hz route53types.HostedZone) string {
		return convert.ToValue(hz.Id)
	})

	res := []interface{}{}
	for _, hz := range allZones {
		tags := tagsByID[convert.ToValue(hz.Id)]

		// Filter by tags
		if conn.Filters.General.IsFilteredOutByTags(mapStringInterfaceToStringString(tags)) {
			log.Debug().Interface("hostedZone", hz.Id).Msg("skipping Route 53 hosted zone due to filters")
			continue
		}

		zoneType := "PUBLIC"
		isPrivate := false
		comment := ""
		config := make(map[string]interface{})

		if hz.Config != nil {
			if hz.Config.PrivateZone {
				zoneType = "PRIVATE"
				isPrivate = true
			}
			comment = convert.ToValue(hz.Config.Comment)
			config["comment"] = comment
			config["privateZone"] = isPrivate
		}

		resourceRecordSetCount := int64(0)
		if hz.ResourceRecordSetCount != nil {
			resourceRecordSetCount = *hz.ResourceRecordSetCount
		}

		mqlHz, err := CreateResource(a.MqlRuntime, "aws.route53.hostedZone",
			map[string]*llx.RawData{
				"__id":                   llx.StringData(convert.ToValue(hz.Id)),
				"id":                     llx.StringData(convert.ToValue(hz.Id)),
				"name":                   llx.StringData(convert.ToValue(hz.Name)),
				"arn":                    llx.StringData(hostedZoneIdToArn(hz.Id)),
				"resourceRecordSetCount": llx.IntData(resourceRecordSetCount),
				"type":                   llx.StringData(zoneType),
				"isPrivate":              llx.BoolData(isPrivate),
				"comment":                llx.StringData(comment),
				"tags":                   llx.MapData(tags, types.String),
				"config":                 llx.DictData(config),
			})
		if err != nil {
			return nil, err
		}

		res = append(res, mqlHz)
	}

	return res, nil
}

func (a *mqlAwsRoute53) healthChecks() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	ctx := context.Background()
	svc := conn.Route53("")

	// Collect all health checks first
	var allChecks []route53types.HealthCheck
	paginator := route53.NewListHealthChecksPaginator(svc, &route53.ListHealthChecksInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			if Is400AccessDeniedError(err) {
				log.Warn().Msg("error accessing Route 53 health checks")
				return []interface{}{}, nil
			}
			return nil, err
		}
		allChecks = append(allChecks, page.HealthChecks...)
	}

	// Batch-fetch tags (up to 10 per API call)
	tagsByID := batchFetchTags(ctx, svc, route53types.TagResourceTypeHealthcheck, allChecks, func(hc route53types.HealthCheck) string {
		return convert.ToValue(hc.Id)
	})

	res := []interface{}{}
	for _, hc := range allChecks {
		tags := tagsByID[convert.ToValue(hc.Id)]

		if conn.Filters.General.IsFilteredOutByTags(mapStringInterfaceToStringString(tags)) {
			continue
		}

		mqlHc, err := newMqlAwsRoute53HealthCheck(a.MqlRuntime, hc, tags)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlHc)
	}

	return res, nil
}

func (a *mqlAwsRoute53) queryLoggingConfigs() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	ctx := context.Background()
	svc := conn.Route53("")
	res := []interface{}{}

	paginator := route53.NewListQueryLoggingConfigsPaginator(svc, &route53.ListQueryLoggingConfigsInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			if Is400AccessDeniedError(err) {
				log.Warn().Msg("error accessing Route 53 query logging configs")
				return res, nil
			}
			return nil, err
		}

		for _, qlc := range page.QueryLoggingConfigs {
			mqlQlc, err := CreateResource(a.MqlRuntime, "aws.route53.queryLoggingConfig",
				map[string]*llx.RawData{
					"__id":                      llx.StringData(convert.ToValue(qlc.Id)),
					"id":                        llx.StringData(convert.ToValue(qlc.Id)),
					"hostedZoneId":              llx.StringData(convert.ToValue(qlc.HostedZoneId)),
					"cloudWatchLogsLogGroupArn": llx.StringData(convert.ToValue(qlc.CloudWatchLogsLogGroupArn)),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlQlc)
		}
	}

	return res, nil
}

// initAwsRoute53HostedZone resolves a hosted zone by ID from the cached list.
func initAwsRoute53HostedZone(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}

	if args["id"] == nil {
		return nil, nil, errors.New("id required to fetch aws route53 hosted zone")
	}

	idVal, ok := args["id"].Value.(string)
	if !ok {
		return nil, nil, errors.New("invalid id for aws route53 hosted zone")
	}

	obj, err := CreateResource(runtime, "aws.route53", map[string]*llx.RawData{})
	if err != nil {
		return nil, nil, err
	}
	r53 := obj.(*mqlAwsRoute53)

	rawResources := r53.GetHostedZones()
	if rawResources.Error != nil {
		return nil, nil, rawResources.Error
	}

	for _, rawResource := range rawResources.Data {
		hz := rawResource.(*mqlAwsRoute53HostedZone)
		if hz.Id.Data == idVal {
			return args, hz, nil
		}
	}

	return nil, nil, errors.New("aws route53 hosted zone not found: " + idVal)
}

// aws.route53.hostedZone

type mqlAwsRoute53HostedZoneInternal struct {
	getHostedZoneResp *route53.GetHostedZoneOutput
	getHostedZoneDone bool
	getDNSSECResp     *route53.GetDNSSECOutput
	getDNSSECDone     bool
}

func (a *mqlAwsRoute53HostedZone) id() (string, error) {
	return a.Id.Data, nil
}

// getHostedZone fetches and caches the GetHostedZone response so that vpcs()
// and nameServers() don't each make a separate API call for the same data.
func (a *mqlAwsRoute53HostedZone) getHostedZone() (*route53.GetHostedZoneOutput, error) {
	if a.getHostedZoneDone {
		return a.getHostedZoneResp, nil
	}

	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	ctx := context.Background()
	svc := conn.Route53("")
	hostedZoneId := a.Id.Data

	resp, err := svc.GetHostedZone(ctx, &route53.GetHostedZoneInput{Id: &hostedZoneId})
	if err != nil {
		if Is400AccessDeniedError(err) {
			a.getHostedZoneDone = true
			return nil, nil
		}
		return nil, err
	}

	a.getHostedZoneResp = resp
	a.getHostedZoneDone = true
	return resp, nil
}

// getDNSSEC fetches and caches the GetDNSSEC response so that dnssecStatus()
// and keySigningKeys() don't each make a separate API call for the same data.
func (a *mqlAwsRoute53HostedZone) getDNSSEC() (*route53.GetDNSSECOutput, error) {
	if a.getDNSSECDone {
		return a.getDNSSECResp, nil
	}

	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	ctx := context.Background()
	svc := conn.Route53("")
	hostedZoneId := a.Id.Data

	resp, err := svc.GetDNSSEC(ctx, &route53.GetDNSSECInput{HostedZoneId: &hostedZoneId})
	if err != nil {
		if Is400AccessDeniedError(err) {
			a.getDNSSECDone = true
			return nil, nil
		}
		return nil, err
	}

	a.getDNSSECResp = resp
	a.getDNSSECDone = true
	return resp, nil
}

func (a *mqlAwsRoute53HostedZone) vpcs() ([]interface{}, error) {
	resp, err := a.getHostedZone()
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, nil
	}

	vpcs := []interface{}{}
	for _, vpc := range resp.VPCs {
		vpcs = append(vpcs, map[string]interface{}{
			"vpcId":     convert.ToValue(vpc.VPCId),
			"vpcRegion": string(vpc.VPCRegion),
		})
	}
	return vpcs, nil
}

func (a *mqlAwsRoute53HostedZone) nameServers() ([]interface{}, error) {
	resp, err := a.getHostedZone()
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, nil
	}

	nameServers := []interface{}{}
	if resp.DelegationSet != nil {
		for _, ns := range resp.DelegationSet.NameServers {
			nameServers = append(nameServers, ns)
		}
	}
	return nameServers, nil
}

func (a *mqlAwsRoute53HostedZone) records() ([]interface{}, error) {
	hostedZoneId := a.Id.Data
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	ctx := context.Background()
	svc := conn.Route53("")
	res := []interface{}{}

	paginator := route53.NewListResourceRecordSetsPaginator(svc,
		&route53.ListResourceRecordSetsInput{
			HostedZoneId: &hostedZoneId,
		})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, rrs := range page.ResourceRecordSets {
			mqlRecord, err := newMqlAwsRoute53Record(a.MqlRuntime, hostedZoneId, rrs)
			if err != nil {
				return nil, err
			}
			res = append(res, mqlRecord)
		}
	}

	return res, nil
}

func (a *mqlAwsRoute53HostedZone) queryLoggingConfig() (*mqlAwsRoute53QueryLoggingConfig, error) {
	hostedZoneId := a.Id.Data
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	ctx := context.Background()
	svc := conn.Route53("")

	listResp, err := svc.ListQueryLoggingConfigs(ctx, &route53.ListQueryLoggingConfigsInput{
		HostedZoneId: &hostedZoneId,
	})
	if err != nil {
		if Is400AccessDeniedError(err) {
			return nil, nil
		}
		return nil, err
	}

	if len(listResp.QueryLoggingConfigs) > 0 {
		qlc := listResp.QueryLoggingConfigs[0]
		mqlQlc, err := CreateResource(a.MqlRuntime, "aws.route53.queryLoggingConfig",
			map[string]*llx.RawData{
				"__id":                      llx.StringData(convert.ToValue(qlc.Id)),
				"id":                        llx.StringData(convert.ToValue(qlc.Id)),
				"hostedZoneId":              llx.StringData(convert.ToValue(qlc.HostedZoneId)),
				"cloudWatchLogsLogGroupArn": llx.StringData(convert.ToValue(qlc.CloudWatchLogsLogGroupArn)),
			})
		if err != nil {
			return nil, err
		}
		return mqlQlc.(*mqlAwsRoute53QueryLoggingConfig), nil
	}

	return nil, nil
}

func (a *mqlAwsRoute53HostedZone) dnssecStatus() (interface{}, error) {
	// DNSSEC is not supported on private hosted zones
	if a.IsPrivate.Data {
		return map[string]interface{}{"serveSignature": "NOT_SIGNING", "statusMessage": "DNSSEC is not supported for private hosted zones"}, nil
	}

	resp, err := a.getDNSSEC()
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, nil
	}

	result := map[string]interface{}{}
	if resp.Status != nil {
		result["serveSignature"] = convert.ToValue(resp.Status.ServeSignature)
		result["statusMessage"] = convert.ToValue(resp.Status.StatusMessage)
	}
	return result, nil
}

func (a *mqlAwsRoute53HostedZone) keySigningKeys() ([]interface{}, error) {
	// DNSSEC is not supported on private hosted zones
	if a.IsPrivate.Data {
		return []interface{}{}, nil
	}

	resp, err := a.getDNSSEC()
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, nil
	}

	hostedZoneId := a.Id.Data

	res := []interface{}{}
	for _, ksk := range resp.KeySigningKeys {
		mqlKsk, err := CreateResource(a.MqlRuntime, "aws.route53.keySigningKey",
			map[string]*llx.RawData{
				"__id":                     llx.StringData(fmt.Sprintf("%s/%s", hostedZoneId, convert.ToValue(ksk.Name))),
				"name":                     llx.StringData(convert.ToValue(ksk.Name)),
				"kmsArn":                   llx.StringData(convert.ToValue(ksk.KmsArn)),
				"hostedZoneId":             llx.StringData(hostedZoneId),
				"flag":                     llx.IntData(int64(ksk.Flag)),
				"signingAlgorithmMnemonic": llx.StringData(convert.ToValue(ksk.SigningAlgorithmMnemonic)),
				"signingAlgorithmType":     llx.IntData(int64(ksk.SigningAlgorithmType)),
				"digestAlgorithmMnemonic":  llx.StringData(convert.ToValue(ksk.DigestAlgorithmMnemonic)),
				"digestAlgorithmType":      llx.IntData(int64(ksk.DigestAlgorithmType)),
				"keyTag":                   llx.IntData(int64(ksk.KeyTag)),
				"digestValue":              llx.StringData(convert.ToValue(ksk.DigestValue)),
				"publicKey":                llx.StringData(convert.ToValue(ksk.PublicKey)),
				"dsRecord":                 llx.StringData(convert.ToValue(ksk.DSRecord)),
				"dnskeyRecord":             llx.StringData(convert.ToValue(ksk.DNSKEYRecord)),
				"status":                   llx.StringData(convert.ToValue(ksk.Status)),
				"statusMessage":            llx.StringData(convert.ToValue(ksk.StatusMessage)),
				"createdDate":              llx.TimeDataPtr(ksk.CreatedDate),
				"lastModifiedDate":         llx.TimeDataPtr(ksk.LastModifiedDate),
			})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlKsk)
	}
	return res, nil
}

// aws.route53.record

type mqlAwsRoute53RecordInternal struct {
	resourceRecordsCache      []interface{}
	aliasTargetCache          map[string]interface{}
	geoLocationCache          map[string]interface{}
	geoProximityLocationCache map[string]interface{}
	cidrRoutingConfigCache    map[string]interface{}
}

func (a *mqlAwsRoute53Record) id() (string, error) {
	return fmt.Sprintf("%s//%s//%s//%s", a.HostedZoneId.Data, a.Name.Data, a.Type.Data, a.SetIdentifier.Data), nil
}

func (a *mqlAwsRoute53Record) resourceRecords() ([]interface{}, error) {
	return a.resourceRecordsCache, nil
}

func (a *mqlAwsRoute53Record) aliasTarget() (interface{}, error) {
	if len(a.aliasTargetCache) == 0 {
		return nil, nil
	}
	return a.aliasTargetCache, nil
}

func (a *mqlAwsRoute53Record) geoLocation() (interface{}, error) {
	if len(a.geoLocationCache) == 0 {
		return nil, nil
	}
	return a.geoLocationCache, nil
}

func (a *mqlAwsRoute53Record) geoProximityLocation() (interface{}, error) {
	if len(a.geoProximityLocationCache) == 0 {
		return nil, nil
	}
	return a.geoProximityLocationCache, nil
}

func (a *mqlAwsRoute53Record) cidrRoutingConfig() (interface{}, error) {
	if len(a.cidrRoutingConfigCache) == 0 {
		return nil, nil
	}
	return a.cidrRoutingConfigCache, nil
}

func (a *mqlAwsRoute53Record) healthCheck() (*mqlAwsRoute53HealthCheck, error) {
	healthCheckId := a.HealthCheckId.Data
	if healthCheckId == "" {
		return nil, nil
	}

	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	ctx := context.Background()
	svc := conn.Route53("")

	resp, err := svc.GetHealthCheck(ctx, &route53.GetHealthCheckInput{
		HealthCheckId: &healthCheckId,
	})
	if err != nil {
		if Is400AccessDeniedError(err) {
			return nil, nil
		}
		return nil, err
	}
	if resp == nil || resp.HealthCheck == nil {
		return nil, nil
	}

	hc := *resp.HealthCheck

	// Fetch tags for this health check
	tagsResp, err := svc.ListTagsForResource(ctx, &route53.ListTagsForResourceInput{
		ResourceType: route53types.TagResourceTypeHealthcheck,
		ResourceId:   &healthCheckId,
	})
	tags := make(map[string]interface{})
	if err == nil && tagsResp.ResourceTagSet != nil {
		for _, tag := range tagsResp.ResourceTagSet.Tags {
			tags[convert.ToValue(tag.Key)] = convert.ToValue(tag.Value)
		}
	}

	mqlHc, err := newMqlAwsRoute53HealthCheck(a.MqlRuntime, hc, tags)
	if err != nil {
		return nil, err
	}
	return mqlHc, nil
}

// aws.route53.healthCheck

type mqlAwsRoute53HealthCheckInternal struct {
	regionsCache               []interface{}
	childHealthChecksCache     []interface{}
	cloudWatchAlarmConfigCache map[string]interface{}
}

func (a *mqlAwsRoute53HealthCheck) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAwsRoute53HealthCheck) regions() ([]interface{}, error) {
	return a.regionsCache, nil
}

func (a *mqlAwsRoute53HealthCheck) childHealthChecks() ([]interface{}, error) {
	return a.childHealthChecksCache, nil
}

func (a *mqlAwsRoute53HealthCheck) cloudWatchAlarmConfiguration() (interface{}, error) {
	if len(a.cloudWatchAlarmConfigCache) == 0 {
		return nil, nil
	}
	return a.cloudWatchAlarmConfigCache, nil
}

func (a *mqlAwsRoute53HealthCheck) status() (string, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	ctx := context.Background()
	svc := conn.Route53("")

	resp, err := svc.GetHealthCheckStatus(ctx, &route53.GetHealthCheckStatusInput{
		HealthCheckId: &a.Id.Data,
	})
	if err != nil {
		return "", err
	}

	if len(resp.HealthCheckObservations) == 0 {
		return "", nil
	}

	// Return the status from the first health checker region. The API returns one
	// observation per region; we surface a single representative status string
	// rather than a per-region map because MQL consumers typically need a simple
	// healthy/unhealthy signal. The full per-region breakdown is available via
	// the AWS console or direct API call if needed.
	obs := resp.HealthCheckObservations[0]
	if obs.StatusReport != nil && obs.StatusReport.Status != nil {
		return *obs.StatusReport.Status, nil
	}
	return "", nil
}

// aws.route53.queryLoggingConfig

func (a *mqlAwsRoute53QueryLoggingConfig) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAwsRoute53QueryLoggingConfig) hostedZone() (*mqlAwsRoute53HostedZone, error) {
	hostedZoneId := a.HostedZoneId.Data
	mqlHz, err := NewResource(a.MqlRuntime, "aws.route53.hostedZone",
		map[string]*llx.RawData{
			"id": llx.StringData(hostedZoneId),
		})
	if err != nil {
		return nil, err
	}
	return mqlHz.(*mqlAwsRoute53HostedZone), nil
}

// aws.route53.keySigningKey

func (a *mqlAwsRoute53KeySigningKey) id() (string, error) {
	return fmt.Sprintf("%s/%s", a.HostedZoneId.Data, a.Name.Data), nil
}

func (a *mqlAwsRoute53KeySigningKey) hostedZone() (*mqlAwsRoute53HostedZone, error) {
	hostedZoneId := a.HostedZoneId.Data
	mqlHz, err := NewResource(a.MqlRuntime, "aws.route53.hostedZone",
		map[string]*llx.RawData{
			"id": llx.StringData(hostedZoneId),
		})
	if err != nil {
		return nil, err
	}
	return mqlHz.(*mqlAwsRoute53HostedZone), nil
}

func (a *mqlAwsRoute53KeySigningKey) kmsKey() (*mqlAwsKmsKey, error) {
	kmsArn := a.KmsArn.Data
	if kmsArn == "" {
		return nil, nil
	}

	mqlKey, err := NewResource(a.MqlRuntime, "aws.kms.key",
		map[string]*llx.RawData{
			"arn": llx.StringData(kmsArn),
		})
	if err != nil {
		return nil, err
	}
	return mqlKey.(*mqlAwsKmsKey), nil
}

// Helper functions

func newMqlAwsRoute53Record(runtime *plugin.Runtime, hostedZoneId string, rrs route53types.ResourceRecordSet) (*mqlAwsRoute53Record, error) {
	resourceRecords := []interface{}{}
	for _, rr := range rrs.ResourceRecords {
		resourceRecords = append(resourceRecords, convert.ToValue(rr.Value))
	}

	isAlias := false
	aliasTargetDnsName := ""
	aliasTargetHostedZoneId := ""
	aliasEvaluateTargetHealth := false

	if rrs.AliasTarget != nil {
		isAlias = true
		aliasTargetDnsName = convert.ToValue(rrs.AliasTarget.DNSName)
		aliasTargetHostedZoneId = convert.ToValue(rrs.AliasTarget.HostedZoneId)
		aliasEvaluateTargetHealth = rrs.AliasTarget.EvaluateTargetHealth
	}

	ttl := int64(0)
	if rrs.TTL != nil {
		ttl = *rrs.TTL
	}

	weight := int64(0)
	if rrs.Weight != nil {
		weight = *rrs.Weight
	}

	setIdentifier := convert.ToValue(rrs.SetIdentifier)

	resource, err := CreateResource(runtime, "aws.route53.record",
		map[string]*llx.RawData{
			"__id":                      llx.StringData(fmt.Sprintf("%s//%s//%s//%s", hostedZoneId, convert.ToValue(rrs.Name), string(rrs.Type), setIdentifier)),
			"hostedZoneId":              llx.StringData(hostedZoneId),
			"name":                      llx.StringData(convert.ToValue(rrs.Name)),
			"type":                      llx.StringData(string(rrs.Type)),
			"ttl":                       llx.IntData(ttl),
			"isAlias":                   llx.BoolData(isAlias),
			"aliasTargetDnsName":        llx.StringData(aliasTargetDnsName),
			"aliasTargetHostedZoneId":   llx.StringData(aliasTargetHostedZoneId),
			"aliasEvaluateTargetHealth": llx.BoolData(aliasEvaluateTargetHealth),
			"setIdentifier":             llx.StringData(setIdentifier),
			"weight":                    llx.IntData(weight),
			"region":                    llx.StringData(string(rrs.Region)),
			"failover":                  llx.StringData(string(rrs.Failover)),
			"multiValueAnswer":          llx.BoolData(rrs.MultiValueAnswer != nil && *rrs.MultiValueAnswer),
			"healthCheckId":             llx.StringData(convert.ToValue(rrs.HealthCheckId)),
			"trafficPolicyInstanceId":   llx.StringData(convert.ToValue(rrs.TrafficPolicyInstanceId)),
		})
	if err != nil {
		return nil, err
	}

	mqlRecord := resource.(*mqlAwsRoute53Record)
	mqlRecord.resourceRecordsCache = resourceRecords

	if rrs.AliasTarget != nil {
		mqlRecord.aliasTargetCache = map[string]interface{}{
			"dnsName":              convert.ToValue(rrs.AliasTarget.DNSName),
			"hostedZoneId":         convert.ToValue(rrs.AliasTarget.HostedZoneId),
			"evaluateTargetHealth": rrs.AliasTarget.EvaluateTargetHealth,
		}
	}

	if rrs.GeoLocation != nil {
		mqlRecord.geoLocationCache = map[string]interface{}{
			"continentCode":   convert.ToValue(rrs.GeoLocation.ContinentCode),
			"countryCode":     convert.ToValue(rrs.GeoLocation.CountryCode),
			"subdivisionCode": convert.ToValue(rrs.GeoLocation.SubdivisionCode),
		}
	}

	if rrs.GeoProximityLocation != nil {
		geo := map[string]interface{}{
			"awsRegion":      convert.ToValue(rrs.GeoProximityLocation.AWSRegion),
			"localZoneGroup": convert.ToValue(rrs.GeoProximityLocation.LocalZoneGroup),
		}
		if rrs.GeoProximityLocation.Bias != nil {
			geo["bias"] = int64(*rrs.GeoProximityLocation.Bias)
		}
		if rrs.GeoProximityLocation.Coordinates != nil {
			geo["latitude"] = convert.ToValue(rrs.GeoProximityLocation.Coordinates.Latitude)
			geo["longitude"] = convert.ToValue(rrs.GeoProximityLocation.Coordinates.Longitude)
		}
		mqlRecord.geoProximityLocationCache = geo
	}

	if rrs.CidrRoutingConfig != nil {
		mqlRecord.cidrRoutingConfigCache = map[string]interface{}{
			"collectionId": convert.ToValue(rrs.CidrRoutingConfig.CollectionId),
			"locationName": convert.ToValue(rrs.CidrRoutingConfig.LocationName),
		}
	}

	return mqlRecord, nil
}

func newMqlAwsRoute53HealthCheck(runtime *plugin.Runtime, hc route53types.HealthCheck, tags map[string]interface{}) (*mqlAwsRoute53HealthCheck, error) {
	config := hc.HealthCheckConfig
	if config == nil {
		return nil, errors.New("health check config is nil for id: " + convert.ToValue(hc.Id))
	}

	regions := []interface{}{}
	for _, region := range config.Regions {
		regions = append(regions, string(region))
	}

	childHealthChecks := []interface{}{}
	for _, childId := range config.ChildHealthChecks {
		childHealthChecks = append(childHealthChecks, childId)
	}

	port := int64(0)
	if config.Port != nil {
		port = int64(*config.Port)
	}

	requestInterval := int64(0)
	if config.RequestInterval != nil {
		requestInterval = int64(*config.RequestInterval)
	}

	failureThreshold := int64(0)
	if config.FailureThreshold != nil {
		failureThreshold = int64(*config.FailureThreshold)
	}

	healthThreshold := int64(0)
	if config.HealthThreshold != nil {
		healthThreshold = int64(*config.HealthThreshold)
	}

	resource, err := CreateResource(runtime, "aws.route53.healthCheck",
		map[string]*llx.RawData{
			"__id":                     llx.StringData(convert.ToValue(hc.Id)),
			"id":                       llx.StringData(convert.ToValue(hc.Id)),
			"arn":                      llx.StringData(healthCheckIdToArn(hc.Id)),
			"tags":                     llx.MapData(tags, types.String),
			"type":                     llx.StringData(string(config.Type)),
			"protocol":                 llx.StringData(healthCheckProtocol(config.Type)),
			"ipAddress":                llx.StringData(convert.ToValue(config.IPAddress)),
			"fullyQualifiedDomainName": llx.StringData(convert.ToValue(config.FullyQualifiedDomainName)),
			"port":                     llx.IntData(port),
			"resourcePath":             llx.StringData(convert.ToValue(config.ResourcePath)),
			"searchString":             llx.StringData(convert.ToValue(config.SearchString)),
			"requestInterval":          llx.IntData(requestInterval),
			"failureThreshold":         llx.IntData(failureThreshold),
			"measureLatency":           llx.BoolData(config.MeasureLatency != nil && *config.MeasureLatency),
			"enableSNI":                llx.BoolData(config.EnableSNI != nil && *config.EnableSNI),
			"healthThreshold":          llx.IntData(healthThreshold),
			"inverted":                 llx.BoolData(config.Inverted != nil && *config.Inverted),
			"disabled":                 llx.BoolData(config.Disabled != nil && *config.Disabled),
			"callerReference":          llx.StringData(convert.ToValue(hc.CallerReference)),
		})
	if err != nil {
		return nil, err
	}

	mqlHc := resource.(*mqlAwsRoute53HealthCheck)
	mqlHc.regionsCache = regions
	mqlHc.childHealthChecksCache = childHealthChecks

	if hc.CloudWatchAlarmConfiguration != nil {
		cwac := hc.CloudWatchAlarmConfiguration
		alarmConfig := map[string]interface{}{
			"comparisonOperator": string(cwac.ComparisonOperator),
			"metricName":         convert.ToValue(cwac.MetricName),
			"namespace":          convert.ToValue(cwac.Namespace),
			"statistic":          string(cwac.Statistic),
		}
		if cwac.EvaluationPeriods != nil {
			alarmConfig["evaluationPeriods"] = int64(*cwac.EvaluationPeriods)
		}
		if cwac.Period != nil {
			alarmConfig["period"] = int64(*cwac.Period)
		}
		if cwac.Threshold != nil {
			alarmConfig["threshold"] = *cwac.Threshold
		}
		dimensions := []interface{}{}
		for _, dim := range cwac.Dimensions {
			dimensions = append(dimensions, map[string]interface{}{
				"name":  convert.ToValue(dim.Name),
				"value": convert.ToValue(dim.Value),
			})
		}
		alarmConfig["dimensions"] = dimensions
		mqlHc.cloudWatchAlarmConfigCache = alarmConfig
	}

	return mqlHc, nil
}

// batchFetchTags fetches tags for Route 53 resources in batches of up to 10
// using the ListTagsForResources (plural) API, returning a map of resource ID
// to tags. This reduces API calls by ~10x compared to per-resource fetching.
func batchFetchTags[T any](ctx context.Context, svc *route53.Client, resourceType route53types.TagResourceType, items []T, getID func(T) string) map[string]map[string]interface{} {
	tagsByID := make(map[string]map[string]interface{}, len(items))
	for _, item := range items {
		tagsByID[getID(item)] = make(map[string]interface{})
	}

	// Collect all IDs and batch in groups of 10
	ids := make([]string, 0, len(items))
	for _, item := range items {
		ids = append(ids, getID(item))
	}

	for i := 0; i < len(ids); i += 10 {
		end := i + 10
		if end > len(ids) {
			end = len(ids)
		}
		batch := ids[i:end]

		resp, err := svc.ListTagsForResources(ctx, &route53.ListTagsForResourcesInput{
			ResourceType: resourceType,
			ResourceIds:  batch,
		})
		if err != nil {
			log.Warn().Err(err).Msg("error batch-fetching Route 53 tags")
			continue
		}

		for _, rts := range resp.ResourceTagSets {
			id := convert.ToValue(rts.ResourceId)
			tags := tagsByID[id]
			for _, tag := range rts.Tags {
				tags[convert.ToValue(tag.Key)] = convert.ToValue(tag.Value)
			}
		}
	}

	return tagsByID
}

func hostedZoneIdToArn(hostedZoneId *string) string {
	if hostedZoneId == nil {
		return ""
	}
	zoneId := strings.TrimPrefix(convert.ToValue(hostedZoneId), "/hostedzone/")
	return fmt.Sprintf("arn:aws:route53:::hostedzone/%s", zoneId)
}

// healthCheckProtocol extracts the protocol (HTTP, HTTPS, TCP) from the health check type.
// Types like HTTP_STR_MATCH and HTTPS_STR_MATCH map to HTTP and HTTPS respectively.
// Returns empty string for types without a protocol (CALCULATED, CLOUDWATCH_METRIC, RECOVERY_CONTROL).
func healthCheckProtocol(t route53types.HealthCheckType) string {
	switch t {
	case route53types.HealthCheckTypeHttp, route53types.HealthCheckTypeHttpStrMatch:
		return "HTTP"
	case route53types.HealthCheckTypeHttps, route53types.HealthCheckTypeHttpsStrMatch:
		return "HTTPS"
	case route53types.HealthCheckTypeTcp:
		return "TCP"
	default:
		return ""
	}
}

func healthCheckIdToArn(healthCheckId *string) string {
	if healthCheckId == nil {
		return ""
	}
	return fmt.Sprintf("arn:aws:route53:::healthcheck/%s", convert.ToValue(healthCheckId))
}
