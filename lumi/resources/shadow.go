package resources

import (
	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/lumi/resources/shadow"
)

const defaultShadowConfig = "/etc/shadow"

func (s *lumiShadow) id() (string, error) {
	return defaultShadowConfig, nil
}

func (s *lumiShadow) GetList() ([]interface{}, error) {
	// TODO: we may want to create a real mondoo file resource
	f, err := s.Runtime.Motor.Transport.FS().Open(defaultShadowConfig)
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

		shadowEntry, err := s.Runtime.CreateResource("shadow.entry",
			"user", entry.User,
			"password", entry.Password,
			"lastchanged", entry.LastChanged,
			"mindays", entry.MinDays,
			"maxdays", entry.MaxDays,
			"warndays", entry.WarnDays,
			"inactivedays", entry.InactiveDays,
			"expirydates", entry.ExpiryDates,
			"reserved", entry.Reserved,
		)
		if err != nil {
			log.Error().Err(err).Str("shadow_entry", entry.User).Msg("lumi[shadow_entry]> could not create shadow entry resource")
			return nil, err
		}
		shadowEntryResources[i] = shadowEntry.(ShadowEntry)
	}

	return shadowEntryResources, nil
}

func (se *lumiShadowEntry) id() (string, error) {
	id, _ := se.User()
	return id, nil
}
