package resources

import (
	"errors"
	"math"
	"strconv"

	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/lumi/resources/shadow"
)

const defaultShadowConfig = "/etc/shadow"

func (s *lumiShadow) id() (string, error) {
	return defaultShadowConfig, nil
}

func parseInt(s string, dflt int64, msg string) (int64, error) {
	if s == "" {
		return dflt, nil
	}

	res, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, errors.New("failed to parse " + msg + " as a number, it is '" + s + "'")
	}
	return res, nil
}

func (s *lumiShadow) GetList() ([]interface{}, error) {
	osProvider, err := osProvider(s.MotorRuntime.Motor)
	if err != nil {
		return nil, err
	}

	// TODO: we may want to create a real mondoo file resource
	f, err := osProvider.FS().Open(defaultShadowConfig)
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

		maxdays, err := parseInt(entry.MaxDays, math.MaxInt64, "MaxDays")
		if err != nil {
			return nil, err
		}

		mindays, err := parseInt(entry.MinDays, -1, "MinDays")
		if err != nil {
			return nil, err
		}

		warndays, err := parseInt(entry.WarnDays, -1, "WarnDays")
		if err != nil {
			return nil, err
		}

		inactivedays, err := parseInt(entry.InactiveDays, math.MaxInt64, "InactiveDays")
		if err != nil {
			return nil, err
		}

		shadowEntry, err := s.MotorRuntime.CreateResource("shadow.entry",
			"user", entry.User,
			"password", entry.Password,
			"lastchanged", entry.LastChanged,
			"mindays", mindays,
			"maxdays", maxdays,
			"warndays", warndays,
			"inactivedays", inactivedays,
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
