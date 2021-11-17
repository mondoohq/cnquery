package platform

import (
	"errors"
	"runtime"

	"go.mondoo.io/mondoo/motor/transports/terraform"

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
	"go.mondoo.io/mondoo/motor/transports/vsphere"
)

func NewDetector(t transports.Transport) *Detector {
	return &Detector{
		transport: t,
	}
}

type Detector struct {
	transport transports.Transport
	cache     *Platform
}

func (d *Detector) resolveOS() (*Platform, bool) {
	// NOTE: on windows, powershell calls are expensive therefore we want to shortcut the detection mechanism
	_, ok := d.transport.(*local.LocalTransport)
	if ok && runtime.GOOS == "windows" {
		return windowsFamily.Resolve(d.transport)
	} else {
		return operatingSystems.Resolve(d.transport)
	}
}

func (d *Detector) Platform() (*Platform, error) {
	if d.transport == nil {
		return nil, errors.New("cannot detect platform without a transport")
	}

	// check if platform is in cache
	if d.cache != nil {
		return d.cache, nil
	}

	var pi *Platform
	switch pt := d.transport.(type) {
	case *vsphere.Transport:
		identifier, err := pt.Identifier()
		if err != nil {
			return nil, err
		}
		return VspherePlatform(pt, identifier)
	case *arista.Transport:
		v, err := pt.GetVersion()
		if err != nil {
			return nil, errors.New("cannot determine arista version")
		}

		return &Platform{
			Name:    "arista-eos",
			Title:   "Arista EOS",
			Arch:    v.Architecture,
			Release: v.Version,
			Kind:    d.transport.Kind(),
			Runtime: d.transport.Runtime(),
		}, nil
	case *aws.Transport:
		return &Platform{
			Name:    "aws",
			Title:   "Amazon Web Services",
			Kind:    transports.Kind_KIND_API,
			Runtime: transports.RUNTIME_AWS,
		}, nil
	case *gcp.Transport:
		return &Platform{
			Name:    "gcp",
			Title:   "Google Cloud Platform",
			Kind:    transports.Kind_KIND_API,
			Runtime: transports.RUNTIME_GCP,
		}, nil
	case *azure.Transport:
		return &Platform{
			Name:    "azure",
			Title:   "Microsoft Azure",
			Kind:    transports.Kind_KIND_API,
			Runtime: transports.RUNTIME_AZ,
		}, nil
	case *ms365.Transport:
		return &Platform{
			Name:    "microsoft365",
			Title:   "Microsoft 365",
			Kind:    transports.Kind_KIND_API,
			Runtime: transports.RUNTIME_MICROSOFT_GRAPH,
		}, nil
	case *ipmi.Transport:
		return &Platform{
			Name:    "ipmi",
			Title:   "Ipmi",
			Kind:    d.transport.Kind(),
			Runtime: d.transport.Runtime(),
		}, nil
	case *equinix.Transport:
		return &Platform{
			Name:    "equinix",
			Title:   "Equinix Metal",
			Kind:    transports.Kind_KIND_API,
			Runtime: transports.RUNTIME_EQUINIX_METAL,
		}, nil
	case *k8s_transport.Transport:
		release := ""
		build := ""
		arch := ""
		sv := pt.ServerVersion()
		if sv != nil {
			release = sv.GitVersion
			build = sv.BuildDate
			arch = sv.Platform
		}

		return &Platform{
			Name:    "kubernetes",
			Title:   "Kubernetes Cluster",
			Release: release,
			Build:   build,
			Arch:    arch,
			Kind:    transports.Kind_KIND_API,
			Runtime: transports.RUNTIME_KUBERNETES,
		}, nil
	case *github.Transport:
		return &Platform{
			Name:    "github",
			Title:   "Github",
			Kind:    transports.Kind_KIND_API,
			Runtime: transports.RUNTIME_GITHUB,
		}, nil
	case *gitlab.Transport:
		return &Platform{
			Name:    "gitlab",
			Title:   "Gitlab",
			Kind:    transports.Kind_KIND_API,
			Runtime: transports.RUNTIME_GITLAB,
		}, nil
	case *terraform.Transport:
		return &Platform{
			Name:    "terraform",
			Title:   "Terraform",
			Kind:    transports.Kind_KIND_API,
			Runtime: "",
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
