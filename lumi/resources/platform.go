package resources

import (
	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/lumi"
)

func (s *lumiPlatform) init(args *lumi.Args) (*lumi.Args, Platform, error) {
	platform, err := s.Runtime.Motor.Platform()
	if err == nil {
		(*args)["name"] = platform.Name
		(*args)["title"] = platform.Title
		(*args)["arch"] = platform.Arch
		(*args)["release"] = platform.Release

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
