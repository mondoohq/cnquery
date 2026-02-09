// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/aws/aws-sdk-go-v2/service/opensearch"
	opensearch_types "github.com/aws/aws-sdk-go-v2/service/opensearch/types"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v12/llx"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/cnquery/v12/providers/aws/connection"
)

func (a *mqlAwsOpensearch) id() (string, error) {
	return "aws.opensearch", nil
}

func (a *mqlAwsOpensearch) domains() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getDomains(conn), 5)
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

func (a *mqlAwsOpensearch) getDomains(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("opensearch>getDomains>calling aws with region %s", region)

			svc := conn.OpenSearch(region)
			ctx := context.Background()
			res := []any{}

			// List all domain names first
			listResp, err := svc.ListDomainNames(ctx, &opensearch.ListDomainNamesInput{})
			if err != nil {
				if Is400AccessDeniedError(err) || IsServiceNotAvailableInRegionError(err) {
					log.Warn().Str("region", region).Msg("error accessing region for AWS API")
					return res, nil
				}
				return nil, err
			}

			if len(listResp.DomainNames) == 0 {
				return res, nil
			}

			// Get domain names for describe call
			domainNames := make([]string, 0, len(listResp.DomainNames))
			for _, d := range listResp.DomainNames {
				if d.DomainName != nil {
					domainNames = append(domainNames, *d.DomainName)
				}
			}

			// Describe domains in batches of 5 (API limit)
			for i := 0; i < len(domainNames); i += 5 {
				end := i + 5
				if end > len(domainNames) {
					end = len(domainNames)
				}
				batch := domainNames[i:end]

				descResp, err := svc.DescribeDomains(ctx, &opensearch.DescribeDomainsInput{
					DomainNames: batch,
				})
				if err != nil {
					return nil, err
				}

				for _, domain := range descResp.DomainStatusList {
					mqlDomain, err := newMqlAwsOpensearchDomain(a.MqlRuntime, region, conn.AccountId(), domain)
					if err != nil {
						return nil, err
					}
					res = append(res, mqlDomain)
				}
			}

			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func (a *mqlAwsOpensearchDomain) id() (string, error) {
	return a.Arn.Data, nil
}

func initAwsOpensearchDomain(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}

	// Get asset identifier if no args provided
	if len(args) == 0 {
		if ids := getAssetIdentifier(runtime); ids != nil {
			args["name"] = llx.StringData(ids.name)
			args["arn"] = llx.StringData(ids.arn)
		}
	}

	if args["arn"] == nil && args["name"] == nil {
		return nil, nil, errors.New("arn or name required to fetch opensearch domain")
	}

	// If we have an ARN but missing region or name, extract from ARN
	// ARN format: arn:aws:es:REGION:ACCOUNT:domain/DOMAIN_NAME
	if args["arn"] != nil && (args["region"] == nil || args["name"] == nil) {
		arnVal := args["arn"].Value.(string)
		parsedArn, err := arn.Parse(arnVal)
		if err != nil {
			return nil, nil, errors.New("invalid arn for opensearch domain")
		}
		if args["region"] == nil {
			args["region"] = llx.StringData(parsedArn.Region)
		}
		if args["name"] == nil {
			args["name"] = llx.StringData(strings.TrimPrefix(parsedArn.Resource, "domain/"))
		}
	}

	if args["name"] == nil || args["region"] == nil {
		return nil, nil, errors.New("arn, or name and region required to fetch opensearch domain")
	}

	name := args["name"].Value.(string)
	region := args["region"].Value.(string)

	conn := runtime.Connection.(*connection.AwsConnection)
	svc := conn.OpenSearch(region)
	ctx := context.Background()

	// Describe the specific domain
	descResp, err := svc.DescribeDomains(ctx, &opensearch.DescribeDomainsInput{
		DomainNames: []string{name},
	})
	if err != nil {
		return nil, nil, err
	}

	if len(descResp.DomainStatusList) == 0 {
		return nil, nil, errors.New("opensearch domain not found")
	}

	domain := descResp.DomainStatusList[0]
	mqlDomain, err := newMqlAwsOpensearchDomain(runtime, region, conn.AccountId(), domain)
	if err != nil {
		return nil, nil, err
	}

	return nil, mqlDomain, nil
}

type mqlAwsOpensearchDomainInternal struct {
	securityGroupIdHandler
	region    string
	subnetIds []string
}

func newMqlAwsOpensearchDomain(runtime *plugin.Runtime, region string, accountID string, domain opensearch_types.DomainStatus) (*mqlAwsOpensearchDomain, error) {
	// Convert security group IDs to ARNs
	sgArns := []string{}
	subnetIds := []string{}
	var vpcId string
	if domain.VPCOptions != nil {
		for _, sgId := range domain.VPCOptions.SecurityGroupIds {
			sgArns = append(sgArns, NewSecurityGroupArn(region, accountID, sgId))
		}
		subnetIds = domain.VPCOptions.SubnetIds
		vpcId = convert.ToValue(domain.VPCOptions.VPCId)
	}

	// Get endpoint
	var endpoint string
	if domain.Endpoint != nil {
		endpoint = *domain.Endpoint
	}

	// Encryption at rest options
	var encryptionAtRestEnabled bool
	var encryptionAtRestKmsKeyId string
	if domain.EncryptionAtRestOptions != nil {
		encryptionAtRestEnabled = convert.ToValue(domain.EncryptionAtRestOptions.Enabled)
		encryptionAtRestKmsKeyId = convert.ToValue(domain.EncryptionAtRestOptions.KmsKeyId)
	}

	// Node-to-node encryption
	var nodeToNodeEncryptionEnabled bool
	if domain.NodeToNodeEncryptionOptions != nil {
		nodeToNodeEncryptionEnabled = convert.ToValue(domain.NodeToNodeEncryptionOptions.Enabled)
	}

	// Cluster config
	var dedicatedMasterEnabled bool
	var dedicatedMasterType string
	var dedicatedMasterCount int
	var instanceType string
	var instanceCount int
	var zoneAwarenessEnabled bool
	var availabilityZoneCount int
	var warmEnabled bool
	var warmType string
	var warmCount int
	var coldStorageEnabled bool
	if domain.ClusterConfig != nil {
		dedicatedMasterEnabled = convert.ToValue(domain.ClusterConfig.DedicatedMasterEnabled)
		dedicatedMasterType = string(domain.ClusterConfig.DedicatedMasterType)
		if domain.ClusterConfig.DedicatedMasterCount != nil {
			dedicatedMasterCount = int(*domain.ClusterConfig.DedicatedMasterCount)
		}
		instanceType = string(domain.ClusterConfig.InstanceType)
		if domain.ClusterConfig.InstanceCount != nil {
			instanceCount = int(*domain.ClusterConfig.InstanceCount)
		}
		zoneAwarenessEnabled = convert.ToValue(domain.ClusterConfig.ZoneAwarenessEnabled)
		if domain.ClusterConfig.ZoneAwarenessConfig != nil && domain.ClusterConfig.ZoneAwarenessConfig.AvailabilityZoneCount != nil {
			availabilityZoneCount = int(*domain.ClusterConfig.ZoneAwarenessConfig.AvailabilityZoneCount)
		}
		warmEnabled = convert.ToValue(domain.ClusterConfig.WarmEnabled)
		warmType = string(domain.ClusterConfig.WarmType)
		if domain.ClusterConfig.WarmCount != nil {
			warmCount = int(*domain.ClusterConfig.WarmCount)
		}
		if domain.ClusterConfig.ColdStorageOptions != nil {
			coldStorageEnabled = convert.ToValue(domain.ClusterConfig.ColdStorageOptions.Enabled)
		}
	}

	// EBS options
	var ebsEnabled bool
	var ebsVolumeType string
	var ebsVolumeSize int
	var ebsIops int
	var ebsThroughput int
	if domain.EBSOptions != nil {
		ebsEnabled = convert.ToValue(domain.EBSOptions.EBSEnabled)
		ebsVolumeType = string(domain.EBSOptions.VolumeType)
		if domain.EBSOptions.VolumeSize != nil {
			ebsVolumeSize = int(*domain.EBSOptions.VolumeSize)
		}
		if domain.EBSOptions.Iops != nil {
			ebsIops = int(*domain.EBSOptions.Iops)
		}
		if domain.EBSOptions.Throughput != nil {
			ebsThroughput = int(*domain.EBSOptions.Throughput)
		}
	}

	// Domain endpoint options
	var enforceHTTPS bool
	var tlsSecurityPolicy string
	if domain.DomainEndpointOptions != nil {
		enforceHTTPS = convert.ToValue(domain.DomainEndpointOptions.EnforceHTTPS)
		tlsSecurityPolicy = string(domain.DomainEndpointOptions.TLSSecurityPolicy)
	}

	// Advanced security options
	var advancedSecurityEnabled bool
	var samlEnabled bool
	var anonymousAuthEnabled bool
	var internalUserDatabaseEnabled bool
	if domain.AdvancedSecurityOptions != nil {
		advancedSecurityEnabled = convert.ToValue(domain.AdvancedSecurityOptions.Enabled)
		anonymousAuthEnabled = convert.ToValue(domain.AdvancedSecurityOptions.AnonymousAuthEnabled)
		internalUserDatabaseEnabled = convert.ToValue(domain.AdvancedSecurityOptions.InternalUserDatabaseEnabled)
		if domain.AdvancedSecurityOptions.SAMLOptions != nil {
			samlEnabled = convert.ToValue(domain.AdvancedSecurityOptions.SAMLOptions.Enabled)
		}
	}

	// Auto-tune options
	var autoTuneState string
	if domain.AutoTuneOptions != nil {
		autoTuneState = string(domain.AutoTuneOptions.State)
	}

	// Audit log options
	auditLogEnabled := parseAuditLogEnabled(domain.LogPublishingOptions)

	// Service software options
	var serviceSoftwareNewVersion string
	if domain.ServiceSoftwareOptions != nil {
		serviceSoftwareNewVersion = convert.ToValue(domain.ServiceSoftwareOptions.NewVersion)
	}

	// Created timestamp
	var createdAt *llx.RawData
	if domain.Created != nil && *domain.Created {
		// OpenSearch doesn't return creation time directly, using nil
		createdAt = llx.NilData
	} else {
		createdAt = llx.NilData
	}

	resource, err := CreateResource(runtime, ResourceAwsOpensearchDomain,
		map[string]*llx.RawData{
			"arn":                         llx.StringDataPtr(domain.ARN),
			"name":                        llx.StringDataPtr(domain.DomainName),
			"domainId":                    llx.StringDataPtr(domain.DomainId),
			"region":                      llx.StringData(region),
			"engineVersion":               llx.StringDataPtr(domain.EngineVersion),
			"endpoint":                    llx.StringData(endpoint),
			"encryptionAtRestEnabled":     llx.BoolData(encryptionAtRestEnabled),
			"encryptionAtRestKmsKeyId":    llx.StringData(encryptionAtRestKmsKeyId),
			"nodeToNodeEncryptionEnabled": llx.BoolData(nodeToNodeEncryptionEnabled),
			"dedicatedMasterEnabled":      llx.BoolData(dedicatedMasterEnabled),
			"dedicatedMasterType":         llx.StringData(dedicatedMasterType),
			"dedicatedMasterCount":        llx.IntData(dedicatedMasterCount),
			"instanceType":                llx.StringData(instanceType),
			"instanceCount":               llx.IntData(instanceCount),
			"zoneAwarenessEnabled":        llx.BoolData(zoneAwarenessEnabled),
			"availabilityZoneCount":       llx.IntData(availabilityZoneCount),
			"warmEnabled":                 llx.BoolData(warmEnabled),
			"warmType":                    llx.StringData(warmType),
			"warmCount":                   llx.IntData(warmCount),
			"coldStorageEnabled":          llx.BoolData(coldStorageEnabled),
			"ebsEnabled":                  llx.BoolData(ebsEnabled),
			"ebsVolumeType":               llx.StringData(ebsVolumeType),
			"ebsVolumeSize":               llx.IntData(ebsVolumeSize),
			"ebsIops":                     llx.IntData(ebsIops),
			"ebsThroughput":               llx.IntData(ebsThroughput),
			"vpcId":                       llx.StringData(vpcId),
			"enforceHTTPS":                llx.BoolData(enforceHTTPS),
			"tlsSecurityPolicy":           llx.StringData(tlsSecurityPolicy),
			"samlEnabled":                 llx.BoolData(samlEnabled),
			"anonymousAuthEnabled":        llx.BoolData(anonymousAuthEnabled),
			"internalUserDatabaseEnabled": llx.BoolData(internalUserDatabaseEnabled),
			"advancedSecurityEnabled":     llx.BoolData(advancedSecurityEnabled),
			"processing":                  llx.BoolDataPtr(domain.Processing),
			"upgradeProcessing":           llx.BoolDataPtr(domain.UpgradeProcessing),
			"createdAt":                   createdAt,
			"autoTuneState":               llx.StringData(autoTuneState),
			"auditLogEnabled":             llx.BoolData(auditLogEnabled),
			"ipAddressType":               llx.StringData(string(domain.IPAddressType)),
			"serviceSoftwareNewVersion":   llx.StringData(serviceSoftwareNewVersion),
		})
	if err != nil {
		return nil, err
	}

	mqlDomain := resource.(*mqlAwsOpensearchDomain)
	mqlDomain.region = region
	mqlDomain.subnetIds = subnetIds
	mqlDomain.setSecurityGroupArns(sgArns)
	return mqlDomain, nil
}

func (a *mqlAwsOpensearchDomain) securityGroups() ([]any, error) {
	return a.newSecurityGroupResources(a.MqlRuntime)
}

func (a *mqlAwsOpensearchDomain) subnets() ([]any, error) {
	if len(a.subnetIds) == 0 {
		return []any{}, nil
	}

	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []any{}
	for _, subnetId := range a.subnetIds {
		subnetArn := fmt.Sprintf(subnetArnPattern, a.region, conn.AccountId(), subnetId)
		sub, err := NewResource(a.MqlRuntime, ResourceAwsVpcSubnet,
			map[string]*llx.RawData{"arn": llx.StringData(subnetArn)})
		if err != nil {
			return nil, err
		}
		res = append(res, sub)
	}
	return res, nil
}

func (a *mqlAwsOpensearchDomain) tags() (map[string]interface{}, error) {
	arnVal := a.Arn.Data
	if arnVal == "" {
		return map[string]interface{}{}, nil
	}

	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	svc := conn.OpenSearch(a.region)
	ctx := context.Background()

	resp, err := svc.ListTags(ctx, &opensearch.ListTagsInput{ARN: &arnVal})
	if err != nil {
		return nil, err
	}

	tags := make(map[string]interface{})
	for _, t := range resp.TagList {
		if t.Key != nil && t.Value != nil {
			tags[*t.Key] = *t.Value
		}
	}
	return tags, nil
}

// parseAuditLogEnabled extracts whether audit logging is enabled from OpenSearch
// LogPublishingOptions. Returns false if the map is nil, missing the AUDIT_LOGS
// key, or if the Enabled field is nil/false.
func parseAuditLogEnabled(opts map[string]opensearch_types.LogPublishingOption) bool {
	if opts == nil {
		return false
	}
	if auditLog, ok := opts["AUDIT_LOGS"]; ok {
		return convert.ToValue(auditLog.Enabled)
	}
	return false
}
