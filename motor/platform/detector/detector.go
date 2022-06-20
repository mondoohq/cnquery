package detector

import (
	"errors"
	"runtime"

	"go.mondoo.io/mondoo/motor/platform"
	"go.mondoo.io/mondoo/motor/transports"
	"go.mondoo.io/mondoo/motor/transports/arista"
	"go.mondoo.io/mondoo/motor/transports/aws"
	"go.mondoo.io/mondoo/motor/transports/azure"
	"go.mondoo.io/mondoo/motor/transports/equinix"
	"go.mondoo.io/mondoo/motor/transports/gcp"
	"go.mondoo.io/mondoo/motor/transports/github"
	"go.mondoo.io/mondoo/motor/transports/gitlab"
	ipmi "go.mondoo.io/mondoo/motor/transports/ipmi"
	k8s_transport "go.mondoo.io/mondoo/motor/transports/k8s"
	"go.mondoo.io/mondoo/motor/transports/local"
	"go.mondoo.io/mondoo/motor/transports/ms365"
	"go.mondoo.io/mondoo/motor/transports/network"
	"go.mondoo.io/mondoo/motor/transports/terraform"
	"go.mondoo.io/mondoo/motor/transports/vsphere"
)

func New(t transports.Transport) *Detector {
	return &Detector{
		transport: t,
	}
}

type Detector struct {
	transport transports.Transport
	cache     *platform.Platform
}

func (d *Detector) resolveOS() (*platform.Platform, bool) {
	// NOTE: on windows, powershell calls are expensive therefore we want to shortcut the detection mechanism
	_, ok := d.transport.(*local.LocalTransport)
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
	case *vsphere.Transport:
		identifier, err := pt.Identifier()
		if err != nil {
			return nil, err
		}
		return platform.VspherePlatform(pt, identifier)
	case *arista.Transport:
		v, err := pt.GetVersion()
		if err != nil {
			return nil, errors.New("cannot determine arista version")
		}

		return &platform.Platform{
			Name:    "arista-eos",
			Title:   "Arista EOS",
			Arch:    v.Architecture,
			Release: v.Version,
			Kind:    pt.Kind(),
			Runtime: pt.Runtime(),
		}, nil
	case *aws.Transport:
		return &platform.Platform{
			Name:    "aws",
			Title:   "Amazon Web Services",
			Kind:    transports.Kind_KIND_API,
			Runtime: transports.RUNTIME_AWS,
		}, nil
	case *gcp.Transport:
		return &platform.Platform{
			Name:    "gcp",
			Title:   "Google Cloud Platform",
			Kind:    transports.Kind_KIND_API,
			Runtime: transports.RUNTIME_GCP,
		}, nil
	case *azure.Transport:
		return &platform.Platform{
			Name:    "azure",
			Title:   "Microsoft Azure",
			Kind:    transports.Kind_KIND_API,
			Runtime: transports.RUNTIME_AZ,
		}, nil
	case *ms365.Transport:
		return &platform.Platform{
			Name:    "microsoft365",
			Title:   "Microsoft 365",
			Kind:    transports.Kind_KIND_API,
			Runtime: transports.RUNTIME_MICROSOFT_GRAPH,
		}, nil
	case *ipmi.Transport:
		return &platform.Platform{
			Name:    "ipmi",
			Title:   "IPMI",
			Kind:    pt.Kind(),
			Runtime: pt.Runtime(),
		}, nil
	case *equinix.Transport:
		return &platform.Platform{
			Name:    "equinix",
			Title:   "Equinix Metal",
			Kind:    transports.Kind_KIND_API,
			Runtime: transports.RUNTIME_EQUINIX_METAL,
		}, nil
	case k8s_transport.Transport:
		return pt.PlatformInfo(), nil
	case *github.Transport:
		return &platform.Platform{
			Name:    "github",
			Title:   "GitHub",
			Kind:    transports.Kind_KIND_API,
			Runtime: transports.RUNTIME_GITHUB,
		}, nil
	case *gitlab.Transport:
		return &platform.Platform{
			Name:    "gitlab",
			Title:   "GitLab",
			Kind:    transports.Kind_KIND_API,
			Runtime: transports.RUNTIME_GITLAB,
		}, nil
	case *terraform.Transport:
		return &platform.Platform{
			Name:    "terraform",
			Title:   "Terraform",
			Kind:    transports.Kind_KIND_API,
			Runtime: "",
		}, nil
	case *network.Transport:
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
