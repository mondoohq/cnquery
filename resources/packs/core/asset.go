package core

import (
	"errors"
	"strings"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/motor/platform"
	"go.mondoo.com/cnquery/motor/providers/network"
	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/upstream/mvd"
)

// convertLumiPlatform2ApiPlatform converts the lumi.Platform to
// a *vadvisor.Platform object for API communication
func convertMqlAsset2ApiPlatform(a Asset) *platform.Platform {
	if a == nil {
		return nil
	}

	name, _ := a.Platform()
	release, _ := a.Version()
	version, _ := a.Version()
	build, _ := a.Build()
	arch, _ := a.Arch()
	title, _ := a.Title()
	labels, _ := a.Labels()

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
func convertAssetPlatform2VulnPlatform(pf *platform.Platform) *mvd.Platform {
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

func (a *mqlAsset) init(args *resources.Args) (*resources.Args, Asset, error) {
	platform, err := a.MotorRuntime.Motor.Platform()
	if err == nil {
		labels := map[string]interface{}{}
		for k := range platform.Labels {
			labels[k] = platform.Labels[k]
		}

		platformIDs, err := a.GetIds()
		if err != nil {
			return nil, nil, err
		}

		(*args)["name"] = a.MotorRuntime.Motor.GetAsset().Name
		(*args)["platform"] = platform.Name
		(*args)["ids"] = platformIDs
		(*args)["title"] = platform.PrettyTitle()
		(*args)["arch"] = platform.Arch
		(*args)["version"] = platform.Version
		(*args)["build"] = platform.Build
		(*args)["kind"] = platform.Kind.Name()
		// FIXME: With the introduction of v8, we need to go through all the runtime
		// fields coming out of motor and make sure that they are written as
		// lowercase with dashes (-) separating them. This is to unify the way
		// the naming works across everything. See:
		// https://www.notion.so/mondoo/Asset-Type-e24efab340b54924b068e2d355fe3f31
		(*args)["runtime"] = strings.ReplaceAll(platform.Runtime, " ", "-")
		(*args)["labels"] = labels

		if transport, ok := a.MotorRuntime.Motor.Provider.(*network.Provider); ok {
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
		log.Error().Err(err).Msg("could not determine asset")
	}
	return args, nil, nil
}

func (a *mqlAsset) id() (string, error) {
	return "asset", nil
}

func (a *mqlAsset) GetIds() ([]interface{}, error) {
	asset := a.MotorRuntime.Motor.GetAsset()
	if asset == nil {
		return nil, errors.New("unimplemented")
	}
	return StrSliceToInterface(asset.PlatformIds), nil
}
