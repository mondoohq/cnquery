package terraform

import (
	"context"
	"path/filepath"

	"go.mondoo.com/cnquery/motor/asset"
	"go.mondoo.com/cnquery/motor/discovery/common"
	"go.mondoo.com/cnquery/motor/motorid"
	"go.mondoo.com/cnquery/motor/providers"
	"go.mondoo.com/cnquery/motor/providers/resolver"
)

type Resolver struct{}

func (r *Resolver) Name() string {
	return "Terraform Static Analysis Resolver"
}

func (r *Resolver) AvailableDiscoveryTargets() []string {
	return []string{}
}

func (r *Resolver) Resolve(ctx context.Context, root *asset.Asset, tc *providers.Config, cfn common.CredentialFn, sfn common.QuerySecretFn, userIdDetectors ...providers.PlatformIdDetector) ([]*asset.Asset, error) {
	name := ""
	if tc.Options["path"] != "" {
		// manifest parent directory name
		name = common.ProjectNameFromPath(tc.Options["path"])
	}

	assetObj := &asset.Asset{
		Name:        root.Name,
		Connections: []*providers.Config{tc},
		State:       asset.State_STATE_ONLINE,
		Labels:      map[string]string{},
	}

	if assetObj.Name == "" {
		assetObj.Name = "Terraform Static Analysis " + name
	}

	// we have 3 different asset types for terraform: hcl, plan and state
	// platform name will differ: terraform, terraform-plan, terraform-state
	// platform family will be terraform

	path, ok := tc.Options["path"]
	if ok {
		absPath, _ := filepath.Abs(path)
		assetObj.Labels["path"] = absPath
	}

	m, err := resolver.NewMotorConnection(ctx, tc, cfn)
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
