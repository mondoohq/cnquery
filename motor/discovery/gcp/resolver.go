package gcp

import (
	"strings"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/motor/asset"
	"go.mondoo.io/mondoo/motor/platform"
	"go.mondoo.io/mondoo/motor/transports"
	gcp_transport "go.mondoo.io/mondoo/motor/transports/gcp"
)

const (
	DiscoveryAll       = "all"
	DiscoveryInstances = "instances"
)

type GcpConfig struct {
	User    string
	Project string
}

func ParseGcpInstanceContext(gcpUrl string) GcpConfig {
	var config GcpConfig

	gcpUrl = strings.TrimPrefix(gcpUrl, "gcp://")

	keyValues := strings.Split(gcpUrl, "/")
	for i := 0; i < len(keyValues); {
		if keyValues[i] == "user" {
			if i+1 < len(keyValues) {
				config.User = keyValues[i+1]
			}
		}
		if keyValues[i] == "project" {
			if i+1 < len(keyValues) {
				config.Project = keyValues[i+1]
			}
		}
		i = i + 2
	}

	return config
}

type GcrResolver struct{}

func (r *GcrResolver) Name() string {
	return "GCP Container Registry Resolver"
}

func (r *GcrResolver) AvailableDiscoveryTargets() []string {
	return []string{}
}

func (r *GcrResolver) ParseConnectionURL(url string, opts ...transports.TransportConfigOption) (*transports.TransportConfig, error) {
	repository := strings.TrimPrefix(url, "gcr://")
	tc := &transports.TransportConfig{
		Backend: transports.TransportBackend_CONNECTION_CONTAINER_REGISTRY,
		Host:    repository,
	}

	for i := range opts {
		opts[i](tc)
	}

	return tc, nil
}

func (r *GcrResolver) Resolve(t *transports.TransportConfig) ([]*asset.Asset, error) {
	resolved := []*asset.Asset{}
	repository := t.Host

	log.Debug().Str("registry", repository).Msg("fetch meta information from gcr registry")
	gcrImages := NewGCRImages()
	assetList, err := gcrImages.ListRepository(repository, true)
	if err != nil {
		log.Error().Err(err).Msg("could not fetch k8s images")
		return nil, err
	}

	for i := range assetList {
		log.Debug().Str("name", assetList[i].Name).Str("image", assetList[i].Connections[0].Host+assetList[i].Connections[0].Path).Msg("resolved image")
		resolved = append(resolved, assetList[i])
	}

	return resolved, nil
}

type GcpResolver struct{}

func (k *GcpResolver) Name() string {
	return "GCP Compute Resolver"
}

func (r *GcpResolver) AvailableDiscoveryTargets() []string {
	return []string{DiscoveryAll, DiscoveryInstances}
}

func (r *GcpResolver) ParseConnectionURL(url string, opts ...transports.TransportConfigOption) (*transports.TransportConfig, error) {
	// parse context from url
	config := ParseGcpInstanceContext(url)

	// check if we got a project or try to determine it
	if len(config.Project) == 0 {
		// try to determine current project
		projectid, err := gcp_transport.GetCurrentProject()

		if err != nil || len(projectid) == 0 {
			return nil, errors.New("gcp: no project id provided")
		}
		config.Project = projectid
	}

	// add gcp api as asset
	t := &transports.TransportConfig{
		Backend: transports.TransportBackend_CONNECTION_GCP,
		User:    config.User,
		Options: map[string]string{
			// TODO: support organization scanning as well
			"project": config.Project,
		},
	}

	return t, nil
}

func (r *GcpResolver) Resolve(tc *transports.TransportConfig) ([]*asset.Asset, error) {
	resolved := []*asset.Asset{}

	trans, err := gcp_transport.New(tc)
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

	project := tc.Options["project"]

	resolved = append(resolved, &asset.Asset{
		PlatformIds: []string{identifier},
		Name:        "GCP project " + project,
		Platform:    pf,
		Connections: []*transports.TransportConfig{tc}, // pass-in the current config
	})

	// discover compute instances
	if tc.IncludesDiscoveryTarget(DiscoveryAll) || tc.IncludesDiscoveryTarget(DiscoveryInstances) {
		compute := NewCompute()

		// we may want to pass a specific user, otherwise it will fallback to ssh config
		if len(tc.User) > 0 {
			compute.InstanceSSHUsername = tc.User
		}

		assetList, err := compute.ListInstancesInProject(project)
		if err != nil {
			return nil, errors.Wrap(err, "could not fetch gcp compute instances")
		}
		log.Debug().Int("instances", len(assetList)).Msg("completed instance search")

		for i := range assetList {
			log.Debug().Str("name", assetList[i].Name).Msg("resolved gcp compute instance")
			resolved = append(resolved, assetList[i])
		}
	}

	return resolved, nil
}
