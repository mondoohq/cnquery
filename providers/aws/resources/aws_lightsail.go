// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/service/lightsail"
	lightsail_types "github.com/aws/aws-sdk-go-v2/service/lightsail/types"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/mql/v13/providers/aws/connection"
	"go.mondoo.com/mql/v13/types"
)

func (a *mqlAwsLightsail) id() (string, error) {
	return "aws.lightsail", nil
}

func (a *mqlAwsLightsail) instances() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getInstances(conn), 5)
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

func (a *mqlAwsLightsail) getInstances(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("lightsail>getInstances>calling aws with region %s", region)

			svc := conn.Lightsail(region)
			ctx := context.Background()
			res := []any{}

			var pageToken *string
			for {
				resp, err := svc.GetInstances(ctx, &lightsail.GetInstancesInput{
					PageToken: pageToken,
				})
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS API")
						return res, nil
					}
					if IsServiceNotAvailableInRegionError(err) {
						log.Debug().Str("region", region).Msg("lightsail not available in region")
						return res, nil
					}
					return nil, err
				}

				for i := range resp.Instances {
					inst := resp.Instances[i]
					tags := lightsailTagsToMap(inst.Tags)

					stateName := ""
					var stateCode int64
					if inst.State != nil {
						if inst.State.Name != nil {
							stateName = *inst.State.Name
						}
						if inst.State.Code != nil {
							stateCode = int64(*inst.State.Code)
						}
					}

					az := ""
					if inst.Location != nil && inst.Location.AvailabilityZone != nil {
						az = *inst.Location.AvailabilityZone
					}

					ipv6 := make([]any, len(inst.Ipv6Addresses))
					for j, ip := range inst.Ipv6Addresses {
						ipv6[j] = ip
					}

					var monthlyTransfer int64
					if inst.Networking != nil && inst.Networking.MonthlyTransfer != nil && inst.Networking.MonthlyTransfer.GbPerMonthAllocated != nil {
						monthlyTransfer = int64(*inst.Networking.MonthlyTransfer.GbPerMonthAllocated)
					}

					mqlInst, err := CreateResource(a.MqlRuntime, "aws.lightsail.instance",
						map[string]*llx.RawData{
							"__id":                               llx.StringDataPtr(inst.Arn),
							"name":                               llx.StringDataPtr(inst.Name),
							"arn":                                llx.StringDataPtr(inst.Arn),
							"region":                             llx.StringData(region),
							"availabilityZone":                   llx.StringData(az),
							"blueprintId":                        llx.StringDataPtr(inst.BlueprintId),
							"blueprintName":                      llx.StringDataPtr(inst.BlueprintName),
							"bundleId":                           llx.StringDataPtr(inst.BundleId),
							"state":                              llx.StringData(stateName),
							"stateCode":                          llx.IntData(stateCode),
							"ipAddressType":                      llx.StringData(string(inst.IpAddressType)),
							"publicIpAddress":                    llx.StringDataPtr(inst.PublicIpAddress),
							"privateIpAddress":                   llx.StringDataPtr(inst.PrivateIpAddress),
							"ipv6Addresses":                      llx.ArrayData(ipv6, types.String),
							"isStaticIp":                         llx.BoolDataPtr(inst.IsStaticIp),
							"monthlyTransferGbPerMonthAllocated": llx.IntData(monthlyTransfer),
							"username":                           llx.StringDataPtr(inst.Username),
							"sshKeyName":                         llx.StringDataPtr(inst.SshKeyName),
							"supportCode":                        llx.StringDataPtr(inst.SupportCode),
							"createdAt":                          llx.TimeDataPtr(inst.CreatedAt),
							"tags":                               llx.MapData(tags, types.String),
						})
					if err != nil {
						return nil, err
					}
					mqlInstRes := mqlInst.(*mqlAwsLightsailInstance)
					mqlInstRes.cacheHardware = inst.Hardware
					mqlInstRes.cacheNetworking = inst.Networking
					mqlInstRes.cacheAddOns = inst.AddOns
					res = append(res, mqlInstRes)
				}

				if resp.NextPageToken == nil {
					break
				}
				pageToken = resp.NextPageToken
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

type mqlAwsLightsailInstanceInternal struct {
	cacheHardware   *lightsail_types.InstanceHardware
	cacheNetworking *lightsail_types.InstanceNetworking
	cacheAddOns     []lightsail_types.AddOn
}

func (a *mqlAwsLightsailInstance) cpuCount() (int64, error) {
	if a.cacheHardware == nil || a.cacheHardware.CpuCount == nil {
		return 0, nil
	}
	return int64(*a.cacheHardware.CpuCount), nil
}

func (a *mqlAwsLightsailInstance) ramSizeInGb() (float64, error) {
	if a.cacheHardware == nil || a.cacheHardware.RamSizeInGb == nil {
		return 0, nil
	}
	return float64(*a.cacheHardware.RamSizeInGb), nil
}

func (a *mqlAwsLightsailInstance) firewallRules() ([]any, error) {
	if a.cacheNetworking == nil {
		return []any{}, nil
	}
	res := make([]any, 0, len(a.cacheNetworking.Ports))
	for _, p := range a.cacheNetworking.Ports {
		rule := map[string]any{
			"fromPort":   int64(p.FromPort),
			"toPort":     int64(p.ToPort),
			"protocol":   string(p.Protocol),
			"accessType": string(p.AccessType),
			"cidrs":      p.Cidrs,
			"ipv6Cidrs":  p.Ipv6Cidrs,
		}
		if p.AccessFrom != nil {
			rule["accessFrom"] = *p.AccessFrom
		}
		if p.CommonName != nil {
			rule["commonName"] = *p.CommonName
		}
		res = append(res, rule)
	}
	return res, nil
}

func (a *mqlAwsLightsailInstance) ports() ([]any, error) {
	if a.cacheNetworking == nil {
		return []any{}, nil
	}
	res := make([]any, 0, len(a.cacheNetworking.Ports))
	for _, p := range a.cacheNetworking.Ports {
		entry := map[string]any{
			"fromPort":        int64(p.FromPort),
			"toPort":          int64(p.ToPort),
			"protocol":        string(p.Protocol),
			"accessType":      string(p.AccessType),
			"accessDirection": string(p.AccessDirection),
			"cidrs":           p.Cidrs,
			"ipv6Cidrs":       p.Ipv6Cidrs,
			"cidrListAliases": p.CidrListAliases,
		}
		if p.AccessFrom != nil {
			entry["accessFrom"] = *p.AccessFrom
		}
		if p.CommonName != nil {
			entry["commonName"] = *p.CommonName
		}
		res = append(res, entry)
	}
	return res, nil
}

func (a *mqlAwsLightsailInstance) addOns() ([]any, error) {
	res := make([]any, 0, len(a.cacheAddOns))
	for _, ao := range a.cacheAddOns {
		entry := map[string]any{}
		if ao.Name != nil {
			entry["name"] = *ao.Name
		}
		if ao.Status != nil {
			entry["status"] = *ao.Status
		}
		if ao.SnapshotTimeOfDay != nil {
			entry["snapshotTimeOfDay"] = *ao.SnapshotTimeOfDay
		}
		if ao.NextSnapshotTimeOfDay != nil {
			entry["nextSnapshotTimeOfDay"] = *ao.NextSnapshotTimeOfDay
		}
		if ao.Threshold != nil {
			entry["threshold"] = *ao.Threshold
		}
		if ao.Duration != nil {
			entry["duration"] = *ao.Duration
		}
		res = append(res, entry)
	}
	return res, nil
}

func (a *mqlAwsLightsail) databases() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getDatabases(conn), 5)
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

func (a *mqlAwsLightsail) getDatabases(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("lightsail>getDatabases>calling aws with region %s", region)

			svc := conn.Lightsail(region)
			ctx := context.Background()
			res := []any{}

			var pageToken *string
			for {
				resp, err := svc.GetRelationalDatabases(ctx, &lightsail.GetRelationalDatabasesInput{
					PageToken: pageToken,
				})
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS API")
						return res, nil
					}
					if IsServiceNotAvailableInRegionError(err) {
						return res, nil
					}
					return nil, err
				}

				for i := range resp.RelationalDatabases {
					db := resp.RelationalDatabases[i]
					tags := lightsailTagsToMap(db.Tags)

					az := ""
					if db.Location != nil && db.Location.AvailabilityZone != nil {
						az = *db.Location.AvailabilityZone
					}

					mqlDb, err := CreateResource(a.MqlRuntime, "aws.lightsail.database",
						map[string]*llx.RawData{
							"__id":                          llx.StringDataPtr(db.Arn),
							"name":                          llx.StringDataPtr(db.Name),
							"arn":                           llx.StringDataPtr(db.Arn),
							"region":                        llx.StringData(region),
							"availabilityZone":              llx.StringData(az),
							"secondaryAvailabilityZone":     llx.StringDataPtr(db.SecondaryAvailabilityZone),
							"engine":                        llx.StringDataPtr(db.Engine),
							"engineVersion":                 llx.StringDataPtr(db.EngineVersion),
							"relationalDatabaseBlueprintId": llx.StringDataPtr(db.RelationalDatabaseBlueprintId),
							"relationalDatabaseBundleId":    llx.StringDataPtr(db.RelationalDatabaseBundleId),
							"state":                         llx.StringDataPtr(db.State),
							"masterUsername":                llx.StringDataPtr(db.MasterUsername),
							"masterDatabaseName":            llx.StringDataPtr(db.MasterDatabaseName),
							"backupRetentionEnabled":        llx.BoolDataPtr(db.BackupRetentionEnabled),
							"latestRestorableTime":          llx.TimeDataPtr(db.LatestRestorableTime),
							"preferredBackupWindow":         llx.StringDataPtr(db.PreferredBackupWindow),
							"preferredMaintenanceWindow":    llx.StringDataPtr(db.PreferredMaintenanceWindow),
							"publiclyAccessible":            llx.BoolDataPtr(db.PubliclyAccessible),
							"parameterApplyStatus":          llx.StringDataPtr(db.ParameterApplyStatus),
							"caCertificateIdentifier":       llx.StringDataPtr(db.CaCertificateIdentifier),
							"supportCode":                   llx.StringDataPtr(db.SupportCode),
							"createdAt":                     llx.TimeDataPtr(db.CreatedAt),
							"tags":                          llx.MapData(tags, types.String),
						})
					if err != nil {
						return nil, err
					}
					mqlDbRes := mqlDb.(*mqlAwsLightsailDatabase)
					mqlDbRes.cacheEndpoint = db.MasterEndpoint
					mqlDbRes.cachePendingModifiedValues = db.PendingModifiedValues
					mqlDbRes.cacheHardware = db.Hardware
					res = append(res, mqlDbRes)
				}

				if resp.NextPageToken == nil {
					break
				}
				pageToken = resp.NextPageToken
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

type mqlAwsLightsailDatabaseInternal struct {
	cacheEndpoint              *lightsail_types.RelationalDatabaseEndpoint
	cachePendingModifiedValues *lightsail_types.PendingModifiedRelationalDatabaseValues
	cacheHardware              *lightsail_types.RelationalDatabaseHardware
}

func (a *mqlAwsLightsailDatabase) endpointAddress() (string, error) {
	if a.cacheEndpoint == nil {
		return "", nil
	}
	if a.cacheEndpoint.Address == nil {
		return "", nil
	}
	return *a.cacheEndpoint.Address, nil
}

func (a *mqlAwsLightsailDatabase) endpointPort() (int64, error) {
	if a.cacheEndpoint == nil || a.cacheEndpoint.Port == nil {
		return 0, nil
	}
	return int64(*a.cacheEndpoint.Port), nil
}

func (a *mqlAwsLightsailDatabase) hasPendingModifiedValues() (bool, error) {
	if a.cachePendingModifiedValues == nil {
		return false, nil
	}
	p := a.cachePendingModifiedValues
	return p.BackupRetentionEnabled != nil || p.EngineVersion != nil || p.MasterUserPassword != nil, nil
}

func (a *mqlAwsLightsailDatabase) cpuCount() (int64, error) {
	if a.cacheHardware == nil || a.cacheHardware.CpuCount == nil {
		return 0, nil
	}
	return int64(*a.cacheHardware.CpuCount), nil
}

func (a *mqlAwsLightsailDatabase) ramSizeInGb() (float64, error) {
	if a.cacheHardware == nil || a.cacheHardware.RamSizeInGb == nil {
		return 0, nil
	}
	return float64(*a.cacheHardware.RamSizeInGb), nil
}

func (a *mqlAwsLightsailDatabase) diskSizeInGb() (int64, error) {
	if a.cacheHardware == nil || a.cacheHardware.DiskSizeInGb == nil {
		return 0, nil
	}
	return int64(*a.cacheHardware.DiskSizeInGb), nil
}

func (a *mqlAwsLightsailDatabase) pendingModifiedValues() (any, error) {
	if a.cachePendingModifiedValues == nil {
		return nil, nil
	}
	p := a.cachePendingModifiedValues
	entry := map[string]any{}
	if p.BackupRetentionEnabled != nil {
		entry["backupRetentionEnabled"] = *p.BackupRetentionEnabled
	}
	if p.EngineVersion != nil {
		entry["engineVersion"] = *p.EngineVersion
	}
	if p.MasterUserPassword != nil {
		entry["masterUserPassword"] = *p.MasterUserPassword
	}
	if len(entry) == 0 {
		return nil, nil
	}
	return entry, nil
}

func (a *mqlAwsLightsail) loadBalancers() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getLoadBalancers(conn), 5)
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

func (a *mqlAwsLightsail) getLoadBalancers(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("lightsail>getLoadBalancers>calling aws with region %s", region)

			svc := conn.Lightsail(region)
			ctx := context.Background()
			res := []any{}

			var pageToken *string
			for {
				resp, err := svc.GetLoadBalancers(ctx, &lightsail.GetLoadBalancersInput{
					PageToken: pageToken,
				})
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS API")
						return res, nil
					}
					if IsServiceNotAvailableInRegionError(err) {
						return res, nil
					}
					return nil, err
				}

				for i := range resp.LoadBalancers {
					lb := resp.LoadBalancers[i]
					tags := lightsailTagsToMap(lb.Tags)

					state := string(lb.State)

					publicPorts := make([]any, len(lb.PublicPorts))
					for j, p := range lb.PublicPorts {
						publicPorts[j] = int64(p)
					}

					azs := make([]any, 0)
					if lb.Location != nil && lb.Location.AvailabilityZone != nil {
						azs = append(azs, *lb.Location.AvailabilityZone)
					}

					mqlLb, err := CreateResource(a.MqlRuntime, "aws.lightsail.loadBalancer",
						map[string]*llx.RawData{
							"__id":              llx.StringDataPtr(lb.Arn),
							"name":              llx.StringDataPtr(lb.Name),
							"arn":               llx.StringDataPtr(lb.Arn),
							"region":            llx.StringData(region),
							"availabilityZones": llx.ArrayData(azs, types.String),
							"state":             llx.StringData(state),
							"protocol":          llx.StringData(string(lb.Protocol)),
							"ipAddressType":     llx.StringData(string(lb.IpAddressType)),
							"publicPorts":       llx.ArrayData(publicPorts, types.Int),
							"healthCheckPath":   llx.StringDataPtr(lb.HealthCheckPath),
							"instancePort":      llx.IntDataDefault(lb.InstancePort, 0),
							"tlsPolicyName":     llx.StringDataPtr(lb.TlsPolicyName),
							"dnsName":           llx.StringDataPtr(lb.DnsName),
							"supportCode":       llx.StringDataPtr(lb.SupportCode),
							"createdAt":         llx.TimeDataPtr(lb.CreatedAt),
							"tags":              llx.MapData(tags, types.String),
						})
					if err != nil {
						return nil, err
					}
					mqlLbRes := mqlLb.(*mqlAwsLightsailLoadBalancer)
					mqlLbRes.cacheInstanceHealthSummary = lb.InstanceHealthSummary
					mqlLbRes.cacheTlsCertificateSummaries = lb.TlsCertificateSummaries
					mqlLbRes.cacheConfigurationOptions = lb.ConfigurationOptions
					res = append(res, mqlLbRes)
				}

				if resp.NextPageToken == nil {
					break
				}
				pageToken = resp.NextPageToken
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

type mqlAwsLightsailLoadBalancerInternal struct {
	cacheInstanceHealthSummary   []lightsail_types.InstanceHealthSummary
	cacheTlsCertificateSummaries []lightsail_types.LoadBalancerTlsCertificateSummary
	cacheConfigurationOptions    map[string]string
}

func (a *mqlAwsLightsailLoadBalancer) instanceHealthSummary() ([]any, error) {
	res, err := convert.JsonToDictSlice(a.cacheInstanceHealthSummary)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (a *mqlAwsLightsailLoadBalancer) tlsCertificateSummaries() ([]any, error) {
	res, err := convert.JsonToDictSlice(a.cacheTlsCertificateSummaries)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (a *mqlAwsLightsailLoadBalancer) httpsRedirectionEnabled() (bool, error) {
	if a.cacheConfigurationOptions == nil {
		return false, nil
	}
	val, ok := a.cacheConfigurationOptions["HttpsRedirectionEnabled"]
	return ok && val == "true", nil
}

func (a *mqlAwsLightsailLoadBalancer) sessionStickinessEnabled() (bool, error) {
	if a.cacheConfigurationOptions == nil {
		return false, nil
	}
	val, ok := a.cacheConfigurationOptions["SessionStickinessEnabled"]
	return ok && val == "true", nil
}

func (a *mqlAwsLightsailLoadBalancer) sessionStickinessLbCookieDurationSeconds() (int64, error) {
	if a.cacheConfigurationOptions == nil {
		return 0, nil
	}
	val, ok := a.cacheConfigurationOptions["SessionStickiness_LB_CookieDurationSeconds"]
	if !ok || val == "" {
		return 0, nil
	}
	n, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		log.Warn().Str("value", val).Err(err).Msg("lightsail: invalid SessionStickiness_LB_CookieDurationSeconds")
		return 0, nil
	}
	return n, nil
}

func (a *mqlAwsLightsail) buckets() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getBuckets(conn), 5)
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

func (a *mqlAwsLightsail) getBuckets(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("lightsail>getBuckets>calling aws with region %s", region)

			svc := conn.Lightsail(region)
			ctx := context.Background()
			res := []any{}

			var pageToken *string
			for {
				resp, err := svc.GetBuckets(ctx, &lightsail.GetBucketsInput{
					PageToken: pageToken,
				})
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS API")
						return res, nil
					}
					if IsServiceNotAvailableInRegionError(err) {
						return res, nil
					}
					return nil, err
				}

				for i := range resp.Buckets {
					b := resp.Buckets[i]
					tags := lightsailTagsToMap(b.Tags)

					state := ""
					if b.State != nil && b.State.Code != nil {
						state = *b.State.Code
					}

					readonlyAccessAccounts := make([]any, len(b.ReadonlyAccessAccounts))
					for j, acc := range b.ReadonlyAccessAccounts {
						readonlyAccessAccounts[j] = acc
					}

					mqlBucket, err := CreateResource(a.MqlRuntime, "aws.lightsail.bucket",
						map[string]*llx.RawData{
							"__id":                   llx.StringDataPtr(b.Arn),
							"name":                   llx.StringDataPtr(b.Name),
							"arn":                    llx.StringDataPtr(b.Arn),
							"region":                 llx.StringData(region),
							"bundleId":               llx.StringDataPtr(b.BundleId),
							"state":                  llx.StringData(state),
							"objectVersioning":       llx.StringDataPtr(b.ObjectVersioning),
							"ableToUpdateBundle":     llx.BoolDataPtr(b.AbleToUpdateBundle),
							"url":                    llx.StringDataPtr(b.Url),
							"createdAt":              llx.TimeDataPtr(b.CreatedAt),
							"tags":                   llx.MapData(tags, types.String),
							"readonlyAccessAccounts": llx.ArrayData(readonlyAccessAccounts, types.String),
						})
					if err != nil {
						return nil, err
					}
					mqlBucketRes := mqlBucket.(*mqlAwsLightsailBucket)
					mqlBucketRes.cacheAccessRules = b.AccessRules
					mqlBucketRes.cacheResourcesReceivingAccess = b.ResourcesReceivingAccess
					res = append(res, mqlBucketRes)
				}

				if resp.NextPageToken == nil {
					break
				}
				pageToken = resp.NextPageToken
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

type mqlAwsLightsailBucketInternal struct {
	cacheAccessRules              *lightsail_types.AccessRules
	cacheResourcesReceivingAccess []lightsail_types.ResourceReceivingAccess
}

func (a *mqlAwsLightsailBucket) accessRules() (any, error) {
	if a.cacheAccessRules == nil {
		return nil, nil
	}
	dict, err := convert.JsonToDict(a.cacheAccessRules)
	if err != nil {
		return nil, err
	}
	return dict, nil
}

func (a *mqlAwsLightsailBucket) resourcesReceivingAccess() ([]any, error) {
	res := make([]any, 0, len(a.cacheResourcesReceivingAccess))
	for _, r := range a.cacheResourcesReceivingAccess {
		entry := map[string]any{}
		if r.Name != nil {
			entry["name"] = *r.Name
		}
		if r.ResourceType != nil {
			entry["resourceType"] = *r.ResourceType
		}
		res = append(res, entry)
	}
	return res, nil
}

func (a *mqlAwsLightsail) disks() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getDisks(conn), 5)
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

func (a *mqlAwsLightsail) getDisks(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("lightsail>getDisks>calling aws with region %s", region)

			svc := conn.Lightsail(region)
			ctx := context.Background()
			res := []any{}

			var pageToken *string
			for {
				resp, err := svc.GetDisks(ctx, &lightsail.GetDisksInput{
					PageToken: pageToken,
				})
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS API")
						return res, nil
					}
					if IsServiceNotAvailableInRegionError(err) {
						return res, nil
					}
					return nil, err
				}

				for i := range resp.Disks {
					d := resp.Disks[i]
					tags := lightsailTagsToMap(d.Tags)

					az := ""
					if d.Location != nil && d.Location.AvailabilityZone != nil {
						az = *d.Location.AvailabilityZone
					}

					mqlDisk, err := CreateResource(a.MqlRuntime, "aws.lightsail.disk",
						map[string]*llx.RawData{
							"__id":             llx.StringDataPtr(d.Arn),
							"name":             llx.StringDataPtr(d.Name),
							"arn":              llx.StringDataPtr(d.Arn),
							"region":           llx.StringData(region),
							"availabilityZone": llx.StringData(az),
							"resourceType":     llx.StringData(string(d.ResourceType)),
							"sizeInGb":         llx.IntDataDefault(d.SizeInGb, 0),
							"state":            llx.StringData(string(d.State)),
							"iops":             llx.IntDataDefault(d.Iops, 0),
							"path":             llx.StringDataPtr(d.Path),
							"isSystemDisk":     llx.BoolDataPtr(d.IsSystemDisk),
							"isAttached":       llx.BoolDataPtr(d.IsAttached),
							"attachedToName":   llx.StringDataPtr(d.AttachedTo),
							"supportCode":      llx.StringDataPtr(d.SupportCode),
							"createdAt":        llx.TimeDataPtr(d.CreatedAt),
							"tags":             llx.MapData(tags, types.String),
						})
					if err != nil {
						return nil, err
					}
					mqlDiskRes := mqlDisk.(*mqlAwsLightsailDisk)
					mqlDiskRes.cacheAttachedToName = d.AttachedTo
					res = append(res, mqlDiskRes)
				}

				if resp.NextPageToken == nil {
					break
				}
				pageToken = resp.NextPageToken
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

type mqlAwsLightsailDiskInternal struct {
	cacheAttachedToName *string
}

func (a *mqlAwsLightsailDisk) attachedTo() (*mqlAwsLightsailInstance, error) {
	if a.cacheAttachedToName == nil || *a.cacheAttachedToName == "" {
		a.AttachedTo.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	target := *a.cacheAttachedToName

	lightsailRes, err := NewResource(a.MqlRuntime, "aws.lightsail", map[string]*llx.RawData{})
	if err != nil {
		return nil, err
	}
	lightsail := lightsailRes.(*mqlAwsLightsail)
	instancesVal := lightsail.GetInstances()
	if instancesVal.Error != nil {
		return nil, instancesVal.Error
	}
	for _, raw := range instancesVal.Data {
		inst, ok := raw.(*mqlAwsLightsailInstance)
		if !ok {
			continue
		}
		nameVal := inst.GetName()
		if nameVal.Error != nil {
			continue
		}
		if inst.Region.Data == a.Region.Data && nameVal.Data == target {
			return inst, nil
		}
	}
	a.AttachedTo.State = plugin.StateIsSet | plugin.StateIsNull
	return nil, nil
}

func lightsailTagsToMap(tags []lightsail_types.Tag) map[string]any {
	result := make(map[string]any)
	for _, t := range tags {
		if t.Key != nil {
			val := ""
			if t.Value != nil {
				val = *t.Value
			}
			result[*t.Key] = val
		}
	}
	return result
}
