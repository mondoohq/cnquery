package detector

import (
	"errors"
	"runtime"

	"go.mondoo.io/mondoo/motor/platform"
	"go.mondoo.io/mondoo/motor/providers"
	"go.mondoo.io/mondoo/motor/providers/arista"
	"go.mondoo.io/mondoo/motor/providers/aws"
	"go.mondoo.io/mondoo/motor/providers/azure"
	"go.mondoo.io/mondoo/motor/providers/equinix"
	"go.mondoo.io/mondoo/motor/providers/gcp"
	"go.mondoo.io/mondoo/motor/providers/github"
	"go.mondoo.io/mondoo/motor/providers/gitlab"
	ipmi "go.mondoo.io/mondoo/motor/providers/ipmi"
	k8s_transport "go.mondoo.io/mondoo/motor/providers/k8s"
	"go.mondoo.io/mondoo/motor/providers/local"
	"go.mondoo.io/mondoo/motor/providers/ms365"
	"go.mondoo.io/mondoo/motor/providers/network"
	"go.mondoo.io/mondoo/motor/providers/terraform"
	"go.mondoo.io/mondoo/motor/providers/vsphere"
)

func New(t providers.Transport) *Detector {
	return &Detector{
		transport: t,
	}
}

type Detector struct {
	transport providers.Transport
	cache     *platform.Platform
}

func (d *Detector) resolveOS() (*platform.Platform, bool) {
	// NOTE: on windows, powershell calls are expensive therefore we want to shortcut the detection mechanism
	_, ok := d.transport.(*local.Provider)
	if ok && runtime.GOOS == "windows" {
		return platform.WindowsFamily.Resolve(d.transport)
	} else {
		return platform.OperatingSystems.Resolve(d.transport)
	}
}

func (d *Detector) Platform() (*platform.Platform, error) {
	if d.transport == nil {
		return nil, errors.New("cannot detect platform without a transport")
	}

	// check if platform is in cache
	if d.cache != nil {
		return d.cache, nil
	}

	var pi *platform.Platform
	switch pt := d.transport.(type) {
	case *vsphere.Provider:
		identifier, err := pt.Identifier()
		if err != nil {
			return nil, err
		}
		return platform.VspherePlatform(pt, identifier)
	case *arista.Provider:
		v, err := pt.GetVersion()
		if err != nil {
			return nil, errors.New("cannot determine arista version")
		}

		return &platform.Platform{
			Name:    "arista-eos",
			Title:   "Arista EOS",
			Arch:    v.Architecture,
			Release: v.Version,
			Version: v.Version,
			Kind:    pt.Kind(),
			Runtime: pt.Runtime(),
		}, nil
	case *aws.Provider:
		return &platform.Platform{
			Name:    "aws",
			Title:   "Amazon Web Services",
			Kind:    providers.Kind_KIND_API,
			Runtime: providers.RUNTIME_AWS,
		}, nil
	case *gcp.Provider:
		return &platform.Platform{
			Name:    "gcp",
			Title:   "Google Cloud Platform",
			Kind:    providers.Kind_KIND_API,
			Runtime: providers.RUNTIME_GCP,
		}, nil
	case *azure.Provider:
		return &platform.Platform{
			Name:    "azure",
			Title:   "Microsoft Azure",
			Kind:    providers.Kind_KIND_API,
			Runtime: providers.RUNTIME_AZ,
		}, nil
	case *ms365.Provider:
		return &platform.Platform{
			Name:    "microsoft365",
			Title:   "Microsoft 365",
			Kind:    providers.Kind_KIND_API,
			Runtime: providers.RUNTIME_MICROSOFT_GRAPH,
		}, nil
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
		return &platform.Platform{
			Name:    "gitlab",
			Title:   "GitLab",
			Kind:    providers.Kind_KIND_API,
			Runtime: providers.RUNTIME_GITLAB,
		}, nil
	case *terraform.Provider:
		return &platform.Platform{
			Name:    "terraform",
			Title:   "Terraform",
			Kind:    providers.Kind_KIND_API,
			Runtime: "",
		}, nil
	case *network.Provider:
		return &platform.Platform{
			Name:    pt.Scheme,
			Title:   "Network API",
			Kind:    pt.Kind(),
			Family:  pt.Family,
			Runtime: pt.Runtime(), // Not sure what we want to set here?
		}, nil
	default:
		var resolved bool
		pi, resolved = d.resolveOS()
		if !resolved {
			return nil, errors.New("could not determine operating system")
		}
		pi.Kind = d.transport.Kind()
		pi.Runtime = d.transport.Runtime()
	}

	// cache value
	d.cache = pi
	return pi, nil
}
