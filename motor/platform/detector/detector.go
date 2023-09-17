package detector

import (
	"errors"
	"runtime"

	"go.mondoo.com/cnquery/motor/providers/opcua"

	"go.mondoo.com/cnquery/motor/platform"
	"go.mondoo.com/cnquery/motor/providers"
	"go.mondoo.com/cnquery/motor/providers/arista"
	"go.mondoo.com/cnquery/motor/providers/aws"
	"go.mondoo.com/cnquery/motor/providers/equinix"
	"go.mondoo.com/cnquery/motor/providers/github"
	"go.mondoo.com/cnquery/motor/providers/gitlab"
	"go.mondoo.com/cnquery/motor/providers/google"
	ipmi "go.mondoo.com/cnquery/motor/providers/ipmi"
	k8s_transport "go.mondoo.com/cnquery/motor/providers/k8s"
	"go.mondoo.com/cnquery/motor/providers/local"
	"go.mondoo.com/cnquery/motor/providers/microsoft"
	"go.mondoo.com/cnquery/motor/providers/network"
	"go.mondoo.com/cnquery/motor/providers/oci"
	"go.mondoo.com/cnquery/motor/providers/okta"
	"go.mondoo.com/cnquery/motor/providers/os"
	"go.mondoo.com/cnquery/motor/providers/slack"
	"go.mondoo.com/cnquery/motor/providers/terraform"
	"go.mondoo.com/cnquery/motor/providers/vcd"
	"go.mondoo.com/cnquery/motor/providers/vsphere"
)

func New(p providers.Instance) *Detector {
	return &Detector{
		provider: p,
	}
}

type Detector struct {
	provider providers.Instance
	cache    *platform.Platform
}

func (d *Detector) resolveOS(p os.OperatingSystemProvider) (*platform.Platform, bool) {
	// NOTE: on windows, powershell calls are expensive therefore we want to shortcut the detection mechanism
	local, ok := p.(*local.Provider)
	if ok && runtime.GOOS == "windows" {
		return WindowsFamily.Resolve(local)
	} else {
		return OperatingSystems.Resolve(p)
	}
}

func (d *Detector) Platform() (*platform.Platform, error) {
	if d.provider == nil {
		return nil, errors.New("cannot detect platform without a transport")
	}

	// check if platform is in cache
	if d.cache != nil {
		return d.cache, nil
	}
	var pi *platform.Platform
	switch pt := d.provider.(type) {
	case *vsphere.Provider:
		identifier, err := pt.Identifier()
		if err != nil {
			return nil, err
		}
		return VspherePlatform(pt, identifier)
	case *arista.Provider:
		v, err := pt.GetVersion()
		if err != nil {
			return nil, errors.New("cannot determine arista version")
		}

		return &platform.Platform{
			Name:    "arista-eos",
			Title:   "Arista EOS",
			Arch:    v.Architecture,
			Version: v.Version,
			Kind:    pt.Kind(),
			Runtime: pt.Runtime(),
		}, nil
	case *aws.Provider:
		return pt.PlatformInfo(), nil
	case *google.Provider:
		return pt.PlatformInfo()
	case *microsoft.Provider:
		return pt.PlatformInfo()
	case *ipmi.Provider:
		return &platform.Platform{
			Name:    "ipmi",
			Title:   "IPMI",
			Kind:    pt.Kind(),
			Runtime: pt.Runtime(),
		}, nil
	case *equinix.Provider:
		return &platform.Platform{
			Name:    "equinix",
			Title:   "Equinix Metal",
			Kind:    providers.Kind_KIND_API,
			Runtime: providers.RUNTIME_EQUINIX_METAL,
		}, nil
	case k8s_transport.KubernetesProvider:
		return pt.PlatformInfo(), nil
	case *github.Provider:
		return pt.PlatformInfo()
	case *gitlab.Provider:
		return pt.PlatformInfo()
	case *terraform.Provider:
		return pt.PlatformInfo(), nil
	case *network.Provider:
		return &platform.Platform{
			Name:    pt.Scheme,
			Title:   "Network API",
			Kind:    pt.Kind(),
			Family:  pt.Family,
			Runtime: pt.Runtime(), // Not sure what we want to set here?
		}, nil
	case *okta.Provider:
		return &platform.Platform{
			Name:    "okta-org",
			Title:   "Okta Organization",
			Kind:    providers.Kind_KIND_API,
			Family:  []string{"okta"},
			Runtime: pt.Runtime(), // TODO
		}, nil
	case *slack.Provider:
		return pt.PlatformInfo()
	case *vcd.Provider:
		return pt.PlatformInfo()
	case *oci.Provider:
		return pt.PlatformInfo()
	case *opcua.Provider:
		return pt.PlatformInfo()
	case os.OperatingSystemProvider:
		var resolved bool
		pi, resolved = d.resolveOS(pt)
		if !resolved {
			return nil, errors.New("could not determine operating system")
		}
		pi.Kind = d.provider.Kind()
		pi.Runtime = d.provider.Runtime()
	default:
		return nil, errors.New("could not determine platform")
	}

	// cache value
	d.cache = pi
	return pi, nil
}
