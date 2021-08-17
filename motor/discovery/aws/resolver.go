package aws

import (
	"strings"

	"go.mondoo.io/mondoo/motor/discovery/common"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/motor/asset"
	"go.mondoo.io/mondoo/motor/platform"
	"go.mondoo.io/mondoo/motor/transports"
	aws_transport "go.mondoo.io/mondoo/motor/transports/aws"
)

const (
	DiscoveryAll       = "all"
	DiscoveryInstances = "instances"
	DiscoverySSM       = "ssm"
)

type Resolver struct{}

func (r *Resolver) Name() string {
	return "AWS EC2 Resolver"
}

func (r *Resolver) AvailableDiscoveryTargets() []string {
	return []string{DiscoveryAll, DiscoveryInstances, DiscoverySSM}
}

func (r *Resolver) Resolve(tc *transports.TransportConfig, cfn common.CredentialFn, sfn common.QuerySecretFn) ([]*asset.Asset, error) {
	resolved := []*asset.Asset{}

	// add aws api as asset
	trans, err := aws_transport.New(tc, aws_transport.TransportOptions(tc.Options)...)
	if err != nil {
		return nil, err
	}

	identifier, err := trans.Identifier()
	if err != nil {
		return nil, err
	}

	// detect platform info for the asset
	detector := platform.NewDetector(trans)
	pf, err := detector.Platform()
	if err != nil {
		return nil, err
	}

	// add asset for the api itself
	info, err := trans.Account()
	if err != nil {
		return nil, err
	}

	resolved = append(resolved, &asset.Asset{
		PlatformIds: []string{identifier},
		Name:        "AWS Account " + info.ID,
		Platform:    pf,
		Connections: []*transports.TransportConfig{tc}, // pass-in the current config
	})

	// filter assets
	discoverFilter := map[string]string{}
	if tc.Discover != nil {
		discoverFilter = tc.Discover.Filter
	}

	instancesPlatformIdsMap := map[string]*asset.Asset{}
	// discover ssm instances
	if tc.IncludesDiscoveryTarget(DiscoveryAll) || tc.IncludesDiscoveryTarget(DiscoverySSM) {
		if val := discoverFilter["ssm"]; val == "true" {
			// create a map to track the platform ids of the ssm instances, to avoid duplication of assets
			s, err := NewSSMManagedInstancesDiscovery(trans.Config())
			if err != nil {
				return nil, errors.Wrap(err, "could not initialize aws ec2 ssm discovery")
			}
			s.FilterOptions = AssembleEc2InstancesFilters(discoverFilter)
			assetList, err := s.List()
			if err != nil {
				return nil, errors.Wrap(err, "could not fetch ec2 ssm instances")
			}
			log.Debug().Int("instances", len(assetList)).Msg("completed ssm instance search")
			for i := range assetList {
				log.Debug().Str("name", assetList[i].Name).Msg("resolved ssm instance")
				instancesPlatformIdsMap[assetList[i].PlatformIds[0]] = assetList[i]
			}
		}
	}
	// discover ec2 instances
	if tc.IncludesDiscoveryTarget(DiscoveryAll) || tc.IncludesDiscoveryTarget(DiscoveryInstances) {
		r, err := NewEc2Discovery(trans.Config())
		if err != nil {
			return nil, errors.Wrap(err, "could not initialize aws ec2 discovery")
		}

		r.Insecure = tc.Insecure
		r.FilterOptions = AssembleEc2InstancesFilters(discoverFilter)

		assetList, err := r.List()
		if err != nil {
			return nil, errors.Wrap(err, "could not fetch ec2 instances")
		}
		log.Debug().Int("instances", len(assetList)).Bool("insecure", r.Insecure).Msg("completed instance search")
		for i := range assetList {
			a := assetList[i]
			log.Debug().Str("name", a.Name).Msg("resolved ec2 instance")
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

	// add all the detected ssm instanced and ec2 instances to the list
	for k := range instancesPlatformIdsMap {
		a := instancesPlatformIdsMap[k]

		// find the secret reference for the asset
		common.EnrichAssetWithSecrets(a, sfn)
		resolved = append(resolved, a)
	}

	return resolved, nil
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
				// to use tag filters with aws, we have to specify tag:KEY for the key, and then put the value as the values
				key := "tag:" + tag[0]
				ec2InstancesFilters.Tags[key] = tag[1]
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

type Ec2InstancesFilters struct {
	InstanceIds []string
	Tags        map[string]string
	Regions     []string
}
