package core

import (
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/motor/platform"
	"go.mondoo.com/cnquery/motor/providers/network"
	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/resources/packs/core/vadvisor"
	"strings"
)

// convertPlatform2VulnPlatform converts the motor platform.Platform to the
// platform object we use for vulnerability data
// TODO: we need to harmonize the platform objects
func convertPlatform2VulnPlatform(pf *platform.Platform) *vadvisor.Platform {
	if pf == nil {
		return nil
	}
	return &vadvisor.Platform{
		Name:    pf.Name,
		Release: pf.Version,
		Build:   pf.Build,
		Arch:    pf.Arch,
		Title:   pf.Title,
		Labels:  pf.Labels,
	}
}

func (s *mqlPlatform) init(args *resources.Args) (*resources.Args, Platform, error) {
	platform, err := s.MotorRuntime.Motor.Platform()
	if err == nil {
		labels := map[string]interface{}{}
		for k := range platform.Labels {
			labels[k] = platform.Labels[k]
		}

		(*args)["name"] = platform.Name
		(*args)["title"] = platform.PrettyTitle()
		(*args)["arch"] = platform.Arch
		// FIXME: remove in v8
		(*args)["release"] = platform.Release
		(*args)["version"] = platform.Version
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

		if transport, ok := s.MotorRuntime.Motor.Provider.(*network.Provider); ok {
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
