package tfstate

import (
	"path/filepath"

	"go.mondoo.io/mondoo/motor/asset"
	"go.mondoo.io/mondoo/motor/discovery/common"
	"go.mondoo.io/mondoo/motor/discovery/credentials"
	"go.mondoo.io/mondoo/motor/motorid"
	"go.mondoo.io/mondoo/motor/providers"
	"go.mondoo.io/mondoo/motor/providers/resolver"
)

type Resolver struct{}

func (r *Resolver) Name() string {
	return "Terraform State Resolver"
}

func (r *Resolver) AvailableDiscoveryTargets() []string {
	return []string{}
}

func (r *Resolver) Resolve(root *asset.Asset, pCfg *providers.Config, cfn credentials.CredentialFn, sfn credentials.QuerySecretFn, userIdDetectors ...providers.PlatformIdDetector) ([]*asset.Asset, error) {
	name := ""
	if pCfg.Options["path"] != "" {
		// manifest parent directory name
		name = common.ProjectNameFromPath(pCfg.Options["path"])
	}

	assetObj := &asset.Asset{
		Name:        root.Name,
		Connections: []*providers.Config{pCfg},
		State:       asset.State_STATE_ONLINE,
		Labels:      map[string]string{},
	}

	if assetObj.Name == "" {
		assetObj.Name = "Terraform State Analysis " + name
	}

	path, ok := pCfg.Options["path"]
	if ok {
		absPath, _ := filepath.Abs(path)
		assetObj.Labels["path"] = absPath
	}

	m, err := resolver.NewMotorConnection(pCfg, cfn)
	if err != nil {
		return nil, err
	}
	defer m.Close()

	// determine platform information
	p, err := m.Platform()
	if err == nil {
		assetObj.Platform = p
	}

	fingerprint, err := motorid.IdentifyPlatform(m.Provider, p, userIdDetectors)
	if err != nil {
		return nil, err
	}
	assetObj.PlatformIds = fingerprint.PlatformIDs
	if fingerprint.Name != "" {
		assetObj.Name = fingerprint.Name
	}

	return []*asset.Asset{assetObj}, nil
}
