package discovery

import (
	"regexp"

	"github.com/cockroachdb/errors"
	"go.mondoo.io/mondoo/apps/mondoo/cmd/options"
	"go.mondoo.io/mondoo/motor/asset"
)

var scheme = regexp.MustCompile(`^(.*?):\/\/(.*)$`)

type Resolver interface {
	Resolve(asset *options.VulnOptsAsset, opts *options.VulnOpts) ([]*asset.Asset, error)
}

var resolver map[string]Resolver

func init() {
	resolver = make(map[string]Resolver)
	resolver["local"] = &instanceResolver{}
	resolver["winrm"] = &instanceResolver{}
	resolver["ssh"] = &instanceResolver{}
	resolver["docker"] = &dockerResolver{}
	resolver["mock"] = &instanceResolver{}
	resolver["tar"] = &instanceResolver{}
	resolver["k8s"] = &k8sResolver{}
	resolver["gcr"] = &gcrResolver{}
	resolver["gcp"] = &gcpResolver{}
	resolver["cr"] = &containerRegistryResolver{}
	resolver["az"] = &azureResolver{}
	resolver["azure"] = &azureResolver{}
	resolver["aws"] = &awsResolver{}
	resolver["ec2"] = &awsResolver{}
	resolver["vagrant"] = &vagrantResolver{}
	resolver["mock"] = &mockResolver{}
	resolver["vsphere"] = &vsphereResolver{}
	resolver["vsphere+vm"] = &vmwareGuestResolver{}
	resolver["aristaeos"] = &instanceResolver{}
	resolver["ms365"] = &ms365Resolver{}
	resolver["ipmi"] = &ipmiResolver{}
}

func Assets(opts *options.VulnOpts) ([]*asset.Asset, error) {
	resolved := []*asset.Asset{}

	for i := range opts.Assets {
		asset := opts.Assets[i]

		m := scheme.FindStringSubmatch(asset.Connection)
		if len(m) < 3 {
			return nil, errors.New("unsupported connection string: " + asset.Connection)
		}

		resolverId := m[1]
		r, ok := resolver[resolverId]
		if !ok {
			return nil, errors.New("unsupported backend: " + resolverId)
		}

		resolverAssets, err := r.Resolve(asset, opts)
		if err != nil {
			return nil, err
		}
		resolved = append(resolved, resolverAssets...)
	}

	return resolved, nil
}
