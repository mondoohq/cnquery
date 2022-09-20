package core

import (
	"strings"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/motor/platform"
	"go.mondoo.com/cnquery/motor/providers/network"
	"go.mondoo.com/cnquery/resources"
)

func (s *mqlPlatform) id() (string, error) {
	return "platform", nil
}

func (s *mqlPlatform) init(args *resources.Args) (*resources.Args, Platform, error) {
	var platform *platform.Platform
	var err error
	platform = s.MotorRuntime.Motor.GetAsset().GetPlatform()
	if platform == nil {
		// TODO(jaym): I don't know why we would need to do this if
		// the resolved asset is already on the motor. Maybe lazy
		// evaluation?
		platform, err = s.MotorRuntime.Motor.Platform()
	}
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
