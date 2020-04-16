package resources

import (
	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/lumi"
	"go.mondoo.io/mondoo/lumi/resources/shadow"
)

const defaultShadowConfig = "/etc/shadow"

func (s *lumiShadow) id() (string, error) {
	return defaultShadowConfig, nil
}

func (se *lumiShadow) init(args *lumi.Args) (*lumi.Args, error) {
	return args, nil
}

func (s *lumiShadow) GetList() ([]interface{}, error) {
	// TODO: we may want to create a real mondoo file resource
	f, err := s.Runtime.Motor.Transport.File(defaultShadowConfig)
	if err != nil {
		return nil, err
	}

	entries, err := shadow.ParseShadow(f)
	if err != nil {
		return nil, err
	}

	shadowEntryResources := make([]interface{}, len(entries))
	for i := range entries {
		entry := entries[i]

		// set init arguments for the lumi shadow entries resource
		args := make(lumi.Args)
		args["user"] = entry.User
		args["password"] = entry.Password
		args["lastchanges"] = entry.LastChanges
		args["mindays"] = entry.MinDays
		args["maxdays"] = entry.MaxDays
		args["warndays"] = entry.WarnDays
		args["inactivedays"] = entry.InactiveDays
		args["expirydates"] = entry.ExpiryDates
		args["reserved"] = entry.Reserved

		e, err := newShadow_entry(s.Runtime, &args)
		if err != nil {
			log.Error().Err(err).Str("shadow_entry", entry.User).Msg("lumi[shadow_entry]> could not create shadow entry resource")
			continue
		}
		shadowEntryResources[i] = e.(Shadow_entry)
	}

	return shadowEntryResources, nil
}

func (se *lumiShadow_entry) id() (string, error) {
	id, _ := se.User()
	return id, nil
}

func (se *lumiShadow_entry) init(args *lumi.Args) (*lumi.Args, error) {
	return args, nil
}
