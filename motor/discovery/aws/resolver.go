package aws

import (
	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/motor/asset"
	"go.mondoo.io/mondoo/motor/platform"
	"go.mondoo.io/mondoo/motor/transports"
	aws_transport "go.mondoo.io/mondoo/motor/transports/aws"
)

type Ec2Config struct {
	User    string
	Region  string
	Profile string
}

type Resolver struct{}

func (r *Resolver) Name() string {
	return "AWS EC2 Resolver"
}

func (r *Resolver) ParseConnectionURL(url string, opts ...transports.TransportConfigOption) (*transports.TransportConfig, error) {
	tc := &transports.TransportConfig{
		Backend: transports.TransportBackend_CONNECTION_AWS,
	}

	for i := range opts {
		opts[i](tc)
	}

	return tc, nil
}

func (r *Resolver) Resolve(t *transports.TransportConfig, opts map[string]string) ([]*asset.Asset, error) {
	resolved := []*asset.Asset{}

	// copy opts into transport config options, to ensure motor transport resolver will get the information as well
	for k, v := range opts {
		t.Options[k] = v
	}

	// add aws api as asset
	trans, err := aws_transport.New(t, aws_transport.TransportOptions(opts)...)
	// trans, err := aws_transport.New(t, transportOpts...)
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
		PlatformIDs: []string{identifier},
		Name:        "AWS Account " + info.ID,
		Platform:    pf,
		Connections: []*transports.TransportConfig{t}, // pass-in the current config
	})

	// discover ec2 instances
	if _, ok := opts["instances"]; ok {
		r, err := NewEc2Discovery(trans.Config())
		if err != nil {
			return nil, errors.Wrap(err, "could not initialize aws ec2 discovery")
		}

		// we may want to pass a specific user, otherwise it will fallback to ssh config
		ec2User, ok := t.Options["ec2user"]
		if ok {
			r.InstanceSSHUsername = ec2User
		}
		r.Insecure = t.Insecure

		assetList, err := r.List()
		if err != nil {
			return nil, errors.Wrap(err, "could not fetch ec2 instances")
		}
		log.Debug().Int("instances", len(assetList)).Bool("insecure", r.Insecure).Msg("completed instance search")
		for i := range assetList {
			log.Debug().Str("name", assetList[i].Name).Msg("resolved ec2 instance")
			if assetList[i].State != asset.State_STATE_RUNNING {
				log.Warn().Str("name", assetList[i].Name).Msg("skip instance that is not running")
				continue
			}
			resolved = append(resolved, assetList[i])
		}
	}
	return resolved, nil
}
