package resources

import (
	"errors"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/llx"
	"go.mondoo.io/mondoo/lumi"
	"go.mondoo.io/mondoo/motor/platform"
	"go.mondoo.io/mondoo/motor/transports/network"
	"go.mondoo.io/mondoo/vadvisor"
	"go.mondoo.io/mondoo/vadvisor/sources/eol"
)

// convertLumiPlatform2ApiPlatform converts the lumi.Platform to
// a *vadvisor.Platform object for API communication
func convertLumiPlatform2ApiPlatform(pf Platform) *platform.Platform {
	if pf == nil {
		return nil
	}

	name, _ := pf.Name()
	release, _ := pf.Release()
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
		Build:   build,
		Arch:    arch,
		Title:   title,
		Labels:  pfLabels,
	}
}

// convertPlatform2VulnPlatform converts the motor platform.Platform to the
// platform object we use for vulnerability data
// TODO: we need to harmonize the platform objects
func convertPlatform2VulnPlatform(pf *platform.Platform) *vadvisor.Platform {
	if pf == nil {
		return nil
	}
	return &vadvisor.Platform{
		Name:    pf.Name,
		Release: pf.Release,
		Build:   pf.Build,
		Arch:    pf.Arch,
		Title:   pf.Title,
		Labels:  pf.Labels,
	}
}

func (s *lumiPlatform) init(args *lumi.Args) (*lumi.Args, Platform, error) {
	platform, err := s.MotorRuntime.Motor.Platform()
	if err == nil {
		labels := map[string]interface{}{}
		for k := range platform.Labels {
			labels[k] = platform.Labels[k]
		}

		(*args)["name"] = platform.Name
		(*args)["title"] = platform.Title
		(*args)["arch"] = platform.Arch
		(*args)["release"] = platform.Release
		(*args)["build"] = platform.Build
		(*args)["kind"] = platform.Kind.Name()
		// FIXME: remove in v8
		(*args)["runtimeEnv"] = platform.Runtime
		// FIXME: With the introduction of v8, we need to go through all the runtime
		// fields coming out of motor and make sure that they are written as
		// lowercase with dashes (-) separating them. This is to unify the way
		// the naming works across everything. See:
		// https://www.notion.so/mondoo/Asset-Type-e24efab340b54924b068e2d355fe3f31
		(*args)["runtime"] = strings.ReplaceAll(platform.Runtime, " ", "-")
		(*args)["labels"] = labels

		if transport, ok := s.MotorRuntime.Motor.Transport.(*network.Transport); ok {
			(*args)["fqdn"] = transport.FQDN
		} else {
			(*args)["fqdn"] = ""
		}

		families := []interface{}{}
		for _, f := range platform.Family {
			families = append(families, f)
		}
		(*args)["family"] = families

	} else {
		log.Error().Err(err).Msg("could not determine platform")
	}
	return args, nil, nil
}

func (s *lumiPlatform) id() (string, error) {
	return "platform", nil
}

func (s *lumiPlatformEol) id() (string, error) {
	return "platform.eol", nil
}

func (p *lumiPlatformEol) init(args *lumi.Args) (*lumi.Args, PlatformEol, error) {
	obj, err := p.MotorRuntime.CreateResource("platform")
	if err != nil {
		return nil, nil, err
	}

	// gather system information
	pf := obj.(Platform)
	eolPlatform := convertPlatform2VulnPlatform(convertLumiPlatform2ApiPlatform(pf))
	platformEolInfo := eol.EolInfo(eolPlatform)

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
