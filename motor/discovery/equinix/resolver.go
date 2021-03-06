package equinix

import (
	"go.mondoo.io/mondoo/motor/asset"
	"go.mondoo.io/mondoo/motor/platform"
	"go.mondoo.io/mondoo/motor/transports"
	equinix_transport "go.mondoo.io/mondoo/motor/transports/equinix"
	"strings"
)

type EquinixConfig struct {
	ProjectID string
}

func ParseEquinixContext(gcpUrl string) EquinixConfig {
	var config EquinixConfig

	gcpUrl = strings.TrimPrefix(gcpUrl, "equinix://")

	keyValues := strings.Split(gcpUrl, "/")
	for i := 0; i < len(keyValues); {
		if keyValues[i] == "projects" {
			if i+1 < len(keyValues) {
				config.ProjectID = keyValues[i+1]
			}
		}
		i = i + 2
	}

	return config
}

type Resolver struct{}

func (r *Resolver) Name() string {
	return "Equinix Metal Resolver"
}

func (r *Resolver) ParseConnectionURL(url string, opts ...transports.TransportConfigOption) (*transports.TransportConfig, error) {
	// parse context from url
	config := ParseEquinixContext(url)

	tc := &transports.TransportConfig{
		Backend: transports.TransportBackend_CONNECTION_EQUINIX_METAL,
		Options: map[string]string{
			"projectID": config.ProjectID,
		},
	}

	for i := range opts {
		opts[i](tc)
	}

	return tc, nil
}

func (r *Resolver) Resolve(t *transports.TransportConfig, opts map[string]string) ([]*asset.Asset, error) {
	resolved := []*asset.Asset{}

	// add aws api as asset
	trans, err := equinix_transport.New(t)
	// trans, err := aws_transport.New(t, transportOpts...)
	if err != nil {
		return nil, err
	}

	identifier, err := trans.Identifier() // TODO: this identifier is not unique
	if err != nil {
		return nil, err
	}

	// detect platform info for the asset
	detector := platform.NewDetector(trans)
	pf, err := detector.Platform()
	if err != nil {
		return nil, err
	}

	resolved = append(resolved, &asset.Asset{
		PlatformIDs: []string{identifier},
		Name:        "Equinix Account", // TODO: we need to relate this to something
		Platform:    pf,
		Connections: []*transports.TransportConfig{t}, // pass-in the current config
	})

	return resolved, nil
}
