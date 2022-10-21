package core

import (
	"context"
	"errors"
	"time"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/motor/platform"
	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/upstream/mvd"
	"go.mondoo.com/ranger-rpc"
)

func (s *mqlPlatformEol) id() (string, error) {
	return "platform.eol", nil
}

// convertMqlPlatform2ApiPlatform converts the resources.Platform to
// a *vadvisor.Platform object for API communication
func convertMqlPlatform2ApiPlatform(pf Platform) *platform.Platform {
	if pf == nil {
		return nil
	}

	name, _ := pf.Name()
	release, _ := pf.Release()
	version, _ := pf.Version()
	build, _ := pf.Build()
	arch, _ := pf.Arch()
	title, _ := pf.Title()
	labels, _ := pf.Labels()

	pfLabels := map[string]string{}
	for k := range labels {
		v := labels[k]
		val, ok := v.(string)
		if ok {
			pfLabels[k] = val
		}
	}

	return &platform.Platform{
		Name:    name,
		Release: release,
		Version: version,
		Build:   build,
		Arch:    arch,
		Title:   title,
		Labels:  pfLabels,
	}
}

// convertPlatform2VulnPlatform converts the motor platform.Platform to the
// platform object we use for vulnerability data
// TODO: we need to harmonize the platform objects
func convertPlatform2VulnPlatform(pf *platform.Platform) *mvd.Platform {
	if pf == nil {
		return nil
	}
	return &mvd.Platform{
		Name:    pf.Name,
		Release: pf.Version,
		Build:   pf.Build,
		Arch:    pf.Arch,
		Title:   pf.Title,
		Labels:  pf.Labels,
	}
}

func (p *mqlPlatformEol) init(args *resources.Args) (*resources.Args, PlatformEol, error) {
	obj, err := p.MotorRuntime.CreateResource("platform")
	if err != nil {
		return nil, nil, err
	}

	// gather system information
	pf := obj.(Platform)
	eolPlatform := convertPlatform2VulnPlatform(convertMqlPlatform2ApiPlatform(pf))

	r := p.MotorRuntime
	mcc := r.UpstreamConfig
	if mcc == nil {
		return nil, nil, errors.New("mondoo upstream configuration is missing")
	}

	// get new advisory report
	// start scanner client
	scannerClient, err := newAdvisoryScannerHttpClient(mcc.ApiEndpoint, mcc.Plugins, ranger.DefaultHttpClient())
	if err != nil {
		return nil, nil, err
	}

	platformEolInfo, err := scannerClient.IsEol(context.Background(), eolPlatform)
	if err != nil {
		return nil, nil, err
	}

	log.Debug().Str("name", eolPlatform.Name).Str("release", eolPlatform.Release).Str("title", eolPlatform.Title).Msg("search for eol information")
	if platformEolInfo == nil {
		return nil, nil, errors.New("no platform eol information available")
	}

	var eolDate *time.Time
	if platformEolInfo.EolDate != "" {
		parsedEolDate, err := time.Parse(time.RFC3339, platformEolInfo.EolDate)
		if err != nil {
			return nil, nil, errors.New("could not parse eol date: " + platformEolInfo.EolDate)
		}
		eolDate = &parsedEolDate
	} else {
		eolDate = &llx.NeverFutureTime
	}

	// if the package cannot be found, we init it as an empty package
	(*args)["docsUrl"] = platformEolInfo.DocsUrl
	(*args)["productUrl"] = platformEolInfo.ProductUrl
	(*args)["date"] = eolDate

	return args, nil, nil
}
