package resources

import "go.mondoo.io/mondoo/lumi/resources/macos"

func (m *lumiMacos) id() (string, error) {
	return "macos", nil
}

func (m *lumiMacos) GetUserPreferences() (map[string]interface{}, error) {
	res := map[string]interface{}{}
	preferences, err := macos.NewPreferences(m.Runtime.Motor.Transport).UserPreferences()
	if err != nil {
		return nil, err
	}

	for k := range preferences {
		res[k] = preferences[k]
	}
	return res, nil
}

func (m *lumiMacos) GetUserHostPreferences() (map[string]interface{}, error) {
	res := map[string]interface{}{}
	preferences, err := macos.NewPreferences(m.Runtime.Motor.Transport).UserHostPreferences()
	if err != nil {
		return nil, err
	}

	for k := range preferences {
		res[k] = preferences[k]
	}
	return res, nil
}
