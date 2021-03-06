package platform

import (
	"errors"
	"go.mondoo.io/mondoo/motor/transports/equinix"

	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/motor/transports"
	"go.mondoo.io/mondoo/motor/transports/arista"
	"go.mondoo.io/mondoo/motor/transports/aws"
	"go.mondoo.io/mondoo/motor/transports/azure"
	"go.mondoo.io/mondoo/motor/transports/gcp"
	ipmi "go.mondoo.io/mondoo/motor/transports/ipmi"
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
	log.Debug().Msg("detector> start resolving the platfrom")
	return operatingSystems.Resolve(d.transport)
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
			Runtime: transports.RUNTIME_AWS,
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
