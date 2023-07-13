package resourceclient

import (
	"encoding/json"
	"regexp"
	"unicode"
	"unicode/utf8"
)

// camelCase conversion adapted from https://gist.github.com/piersy/b9934790a8892db1a603820c0c23e4a7
// Regexp definitions
var (
	keyMatchRegex    = regexp.MustCompile(`\"(\w+)\":`)
	wordBarrierRegex = regexp.MustCompile(`(\w)([A-Z])`)
)

type camelCaseMarshaller struct {
	Value interface{}
}

func (c camelCaseMarshaller) MarshalJSON() ([]byte, error) {
	marshalled, err := json.Marshal(c.Value)

	converted := keyMatchRegex.ReplaceAllFunc(
		marshalled,
		func(match []byte) []byte {
			// empty keys are valid JSON, only lowercase if we do not have an empty key.
			if len(match) > 2 {
				// decode first rune after the double quotes
				r, width := utf8.DecodeRune(match[1:])
				r = unicode.ToLower(r)
				utf8.EncodeRune(match[1:width+1], r)
			}
			return match
		},
	)
	return converted, err
}

// PropertiesToDict converts an interface to a lowerCase json
// This enables us to avoid reencoding the xml and mo tag for vmware structs
func PropertiesToDict(value interface{}) (map[string]interface{}, error) {
	// config to dict
	configDict := map[string]interface{}{}
	configData, err := json.Marshal(camelCaseMarshaller{value})
	if err != nil {
		return nil, err
	}

	// iterate over keys and lowerCase them
	err = json.Unmarshal(configData, &configDict)
	if err != nil {
		return nil, err
	}

	return configDict, nil
}
