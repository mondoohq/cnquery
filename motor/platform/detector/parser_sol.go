package detector

import (
	"fmt"
	"regexp"
	"strings"
)

type SolarisRelease struct {
	ID      string
	Title   string
	Release string
}

var solarisVersionRegex = regexp.MustCompile(`^\s+((?:[\w]\s*)*Solaris)\s([\w\d.]+)`)

func ParseSolarisRelease(content string) (*SolarisRelease, error) {
	m := solarisVersionRegex.FindStringSubmatch(content)
	if len(m) < 2 {
		return nil, fmt.Errorf("could not parse solaris version: %s", content)
	}

	id := strings.ToLower(m[1])
	id = strings.Replace(id, "oracle", "", 1)
	id = strings.ReplaceAll(id, " ", "")

	return &SolarisRelease{
		ID:      id,
		Title:   m[1],
		Release: m[2],
	}, nil
}
