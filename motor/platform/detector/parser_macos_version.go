package detector

import (
	"encoding/xml"
	"regexp"
)

var DARWIN_RELEASE_REGEX = regexp.MustCompile(`(?m)^\s*(.+?)\s*:\s*(.+?)\s*$`)

// ParseDarwinRelease will parse the output of `/usr/bin/sw_vers`
func ParseDarwinRelease(content string) (map[string]string, error) {
	return parseKeyValue(content, DARWIN_RELEASE_REGEX), nil
}

// parse macos system version property list
type PropertyListDict struct {
	Keys   []string `xml:"key"`
	Values []string `xml:"string"`
}

type PropertyList struct {
	XMLName xml.Name         `xml:"plist"`
	Version string           `xml:"version,attr"`
	Dict    PropertyListDict `xml:"dict"`
}

// parseMacOSSystemVersion will parse the content of
// `/System/Library/CoreServices/SystemVersion.plist` and return the
// result as structured values
func ParseMacOSSystemVersion(content string) (map[string]string, error) {
	v := PropertyList{}
	err := xml.Unmarshal([]byte(content), &v)
	if err != nil {
		return nil, err
	}

	m := make(map[string]string)
	for i := range v.Dict.Keys {
		m[v.Dict.Keys[i]] = v.Dict.Values[i]
	}
	return m, nil
}
