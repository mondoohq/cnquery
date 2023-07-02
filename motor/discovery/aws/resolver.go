package aws

import (
	"context"
	"fmt"
	"strings"

	"errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/motor/asset"
	"go.mondoo.com/cnquery/motor/discovery/common"
	"go.mondoo.com/cnquery/motor/platform/detector"
	"go.mondoo.com/cnquery/motor/providers"
	aws_provider "go.mondoo.com/cnquery/motor/providers/aws"
	"go.mondoo.com/cnquery/motor/vault"
)

const (
	DiscoveryInstances    = "instances"     // todo: convert to using mql under the hood
	DiscoverySSM          = "ssm"           // deprecated: use DiscoverySSMInstances instead
	DiscoverySSMInstances = "ssm-instances" // todo: convert to using mql under the hood
	DiscoveryECR          = "ecr"           // todo: convert to using mql under the hood
	DiscoveryECS          = "ecs"           // todo: convert to using mql under the hood, add ecs-exec
	// api scan
	DiscoveryAccounts                   = "accounts"
	DiscoveryResources                  = "resources"          // all the resources
	DiscoveryECSContainersAPI           = "ecs-containers-api" // need dedup story
	DiscoveryECRImageAPI                = "ecr-image-api"      // need policy + dedup story
	DiscoveryEC2InstanceAPI             = "ec2-instances-api"  // need policy + dedup story
	DiscoverySSMInstanceAPI             = "ssm-instances-api"  // need policy + dedup story
	DiscoveryS3Buckets                  = "s3-buckets"
	DiscoveryCloudtrailTrails           = "cloudtrail-trails"
	DiscoveryRdsDbInstances             = "rds-dbinstances"
	DiscoveryVPCs                       = "vpcs"
	DiscoverySecurityGroups             = "security-groups"
	DiscoveryIAMUsers                   = "iam-users"
	DiscoveryIAMGroups                  = "iam-groups"
	DiscoveryCloudwatchLoggroups        = "cloudwatch-loggroups"
	DiscoveryLambdaFunctions            = "lambda-functions"
	DiscoveryDynamoDBTables             = "dynamodb-tables"
	DiscoveryRedshiftClusters           = "redshift-clusters"
	DiscoveryVolumes                    = "ec2-volumes"
	DiscoverySnapshots                  = "ec2-snapshots"
	DiscoveryEFSFilesystems             = "efs-filesystems"
	DiscoveryAPIGatewayRestAPIs         = "gateway-restapis"
	DiscoveryELBLoadBalancers           = "elb-loadbalancers"
	DiscoveryESDomains                  = "es-domains"
	DiscoveryKMSKeys                    = "kms-keys"
	DiscoverySagemakerNotebookInstances = "sagemaker-notebookinstances"
)

var ResourceDiscoveryTargets = []string{
	DiscoveryResources, DiscoveryECSContainersAPI, DiscoveryEC2InstanceAPI, DiscoverySSMInstanceAPI, DiscoveryECRImageAPI,
	DiscoveryS3Buckets, DiscoveryCloudtrailTrails, DiscoveryRdsDbInstances,
	DiscoveryVPCs, DiscoverySecurityGroups, DiscoveryIAMUsers, DiscoveryIAMGroups,
	DiscoveryCloudwatchLoggroups, DiscoveryLambdaFunctions, DiscoveryDynamoDBTables, DiscoveryRedshiftClusters,
	DiscoveryVolumes, DiscoverySnapshots, DiscoveryEFSFilesystems,
	DiscoveryAPIGatewayRestAPIs, DiscoveryELBLoadBalancers, DiscoveryESDomains, DiscoveryKMSKeys, DiscoverySagemakerNotebookInstances,
}

type Resolver struct{}

func (r *Resolver) Name() string {
	return "AWS Resolver"
}

func (r *Resolver) AvailableDiscoveryTargets() []string {
	discovery := []string{
		common.DiscoveryAuto, common.DiscoveryAll, DiscoveryAccounts,
		DiscoveryInstances, DiscoverySSM, DiscoverySSMInstances,
		DiscoveryECR, DiscoveryECS,
	}
	discovery = append(discovery, ResourceDiscoveryTargets...)
	return discovery
}

func (r *Resolver) Resolve(ctx context.Context, root *asset.Asset, tc *providers.Config, credsResolver vault.Resolver, sfn common.QuerySecretFn, userIdDetectors ...providers.PlatformIdDetector) ([]*asset.Asset, error) {
	resolved := []*asset.Asset{}

	// add aws api as asset
	provider, err := aws_provider.New(tc, aws_provider.TransportOptions(tc.Options)...)
	if err != nil {
		return nil, err
	}

	identifier, err := provider.Identifier()
	if err != nil {
		return nil, err
	}

	// detect platform info for the asset
	detector := detector.New(provider)
	pf, err := detector.Platform()
	if err != nil {
		return nil, err
	}

	// add asset for the api itself
	info, err := provider.Account()
	if err != nil {
		return nil, err
	}

	alias := ""
	if len(info.Aliases) > 0 {
		// there can only be one alias
		alias = info.Aliases[0]
	}

	var resolvedRoot *asset.Asset
	if tc.IncludesOneOfDiscoveryTarget(common.DiscoveryAll, common.DiscoveryAuto, DiscoveryAccounts) {
		name := root.Name
		if name == "" {
			name = AssembleIntegrationName(alias, info.ID)
		}

		resolvedRoot = &asset.Asset{
			PlatformIds: []string{identifier},
			Name:        name,
			Platform:    pf,
			Connections: []*providers.Config{tc}, // pass-in the current config
			State:       asset.State_STATE_ONLINE,
		}
		resolved = append(resolved, resolvedRoot)
	}

	// filter assets
	discoverFilter := map[string]string{}
	if tc.Discover != nil {
		discoverFilter = tc.Discover.Filter
	}

	mqldiscovery, err := NewMQLAssetsDiscovery(provider)
	if err != nil {
		return nil, err
	}

	// resources as assets
	if tc.IncludesOneOfDiscoveryTarget(ResourceDiscoveryTargets...) { // todo: add auto, all when policies are ready
		assetList, err := GatherMQLObjects(mqldiscovery, tc.Clone(), info.ID)
		if err != nil {
			return nil, err
		}
		for i := range assetList {
			a := assetList[i]
			if resolvedRoot != nil {
				a.RelatedAssets = append(a.RelatedAssets, resolvedRoot)
			}
			resolved = append(resolved, a)
		}
	}

	instancesPlatformIdsMap := map[string]*asset.Asset{}
	// discover ssm instances
	if tc.IncludesOneOfDiscoveryTarget(common.DiscoveryAll, DiscoverySSM, DiscoverySSMInstances) {
		// create a map to track the platform ids of the ssm instances, to avoid duplication of assets
		s, err := NewSSMManagedInstancesDiscovery(mqldiscovery, tc.Clone(), info.ID)
		if err != nil {
			return nil, errors.Join(err, errors.New("could not initialize aws ec2 ssm discovery"))
		}
		s.FilterOptions = AssembleEc2InstancesFilters(discoverFilter)
		s.profile = tc.Options["profile"]
		s.PassInLabels = root.Labels
		assetList, err := s.List()
		if err != nil {
			return nil, errors.Join(err, errors.New("could not fetch ec2 ssm instances"))
		}
		log.Debug().Int("instances", len(assetList)).Msg("completed ssm instance search")
		for i := range assetList {
			a := assetList[i]
			if resolvedRoot != nil {
				a.RelatedAssets = append(a.RelatedAssets, resolvedRoot)
			}
			log.Debug().Str("name", a.Name).Str("region", a.Labels[RegionLabel]).Str("state", strings.ToLower(a.State.String())).Msg("resolved ssm instance")
			instancesPlatformIdsMap[a.PlatformIds[0]] = a
		}
	}
	// discover ec2 instances
	if tc.IncludesOneOfDiscoveryTarget(common.DiscoveryAll, DiscoveryInstances) {
		e, err := NewEc2Discovery(mqldiscovery, tc.Clone(), info.ID)
		if err != nil {
			return nil, errors.Join(err, errors.New("could not initialize aws ec2 discovery"))
		}

		e.Insecure = tc.Insecure
		e.FilterOptions = AssembleEc2InstancesFilters(discoverFilter)
		e.profile = tc.Options["profile"]
		e.PassInLabels = root.Labels
		assetList, err := e.List()
		if err != nil {
			return nil, errors.Join(err, errors.New("could not fetch ec2 instances"))
		}
		log.Debug().Int("instances", len(assetList)).Bool("insecure", e.Insecure).Msg("completed instance search")
		for i := range assetList {
			a := assetList[i]
			if resolvedRoot != nil {
				a.RelatedAssets = append(a.RelatedAssets, resolvedRoot)
			}
			log.Debug().Str("name", a.Name).Str("region", a.Labels[RegionLabel]).Str("state", strings.ToLower(a.State.String())).Msg("resolved ec2 instance")
			id := a.PlatformIds[0]
			existing, ok := instancesPlatformIdsMap[id]
			if ok {
				// NOTE: we do not merge connections here, since ssm is available
				// merge labels
				for k, v := range a.Labels {
					existing.Labels[k] = v
				}
			} else {
				instancesPlatformIdsMap[id] = a
			}
		}
	}

	// add all the detected ssm instance and ec2 instances to the list
	for k := range instancesPlatformIdsMap {
		resolved = append(resolved, instancesPlatformIdsMap[k])
	}

	if tc.IncludesOneOfDiscoveryTarget(common.DiscoveryAll, DiscoveryECR) {
		i, err := NewEcrDiscovery(mqldiscovery, tc.Clone(), info.ID)
		if err != nil {
			return nil, errors.Join(err, errors.New("could not initialize aws ecr discovery"))
		}

		i.profile = tc.Options["profile"]
		i.PassInLabels = root.Labels
		assetList, err := i.List()
		if err != nil {
			return nil, errors.Join(err, errors.New("could not fetch ecr repositories information"))
		}
		log.Debug().Int("images", len(assetList)).Msg("completed ecr search")
		for i := range assetList {
			a := assetList[i]
			if resolvedRoot != nil {
				a.RelatedAssets = append(a.RelatedAssets, resolvedRoot)
			}
			resolved = append(resolved, a)
		}
	}

	if tc.IncludesOneOfDiscoveryTarget(common.DiscoveryAll, DiscoveryECS) {
		c, err := NewECSContainersDiscovery(mqldiscovery, tc.Clone(), info.ID)
		if err != nil {
			return nil, errors.Join(err, errors.New("could not initialize aws ecs discovery"))
		}
		c.PassInLabels = root.Labels
		assetList, err := c.List()
		if err != nil {
			return nil, errors.Join(err, errors.New("could not fetch ecs clusters information"))
		}
		log.Debug().Int("assets", len(assetList)).Msg("completed ecs search")
		for i := range assetList {
			a := assetList[i]
			if a == nil {
				continue
			}
			if resolvedRoot != nil {
				a.RelatedAssets = append(a.RelatedAssets, resolvedRoot)
			}
			resolved = append(resolved, a)
		}
	}

	assetMap := make(map[string]*asset.Asset)
	// ensure we don't return the same asset twice
	for i := range resolved {
		assetMap[resolved[i].PlatformIds[0]] = resolved[i]
	}
	new := make([]*asset.Asset, 0, len(assetMap))
	for _, v := range assetMap {
		new = append(new, v)
	}

	return new, nil
}

func AssembleEc2InstancesFilters(opts map[string]string) Ec2InstancesFilters {
	var ec2InstancesFilters Ec2InstancesFilters
	if _, ok := opts["instance-ids"]; ok {
		instanceIds := strings.Split(opts["instance-ids"], ",")
		ec2InstancesFilters.InstanceIds = instanceIds
	}
	if _, ok := opts["tags"]; ok {
		tags := strings.Split(opts["tags"], ",")
		ec2InstancesFilters.Tags = make(map[string]string, len(tags))
		for _, tagkv := range tags {
			tag := strings.Split(tagkv, "=")
			if len(tag) == 2 {
				ec2InstancesFilters.Tags[tag[0]] = tag[1]
			} else if len(tag) == 1 {
				// this means no value was included, so we search for just the tag key
				ec2InstancesFilters.Tags["tag-key"] = tag[0]
			}
		}
	}
	if _, ok := opts["regions"]; ok {
		regions := strings.Split(opts["regions"], ",")
		ec2InstancesFilters.Regions = regions
	}
	return ec2InstancesFilters
}

func AssembleIntegrationName(alias string, id string) string {
	if alias == "" {
		return fmt.Sprintf("AWS Account %s", id)
	}
	return fmt.Sprintf("AWS Account %s (%s)", alias, id)
}
