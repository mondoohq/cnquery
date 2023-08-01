package detector

import (
	"errors"
	"regexp"
	"strings"
)

var (
	OS_RELEASE_REGEX    = regexp.MustCompile(`(?m)^\s*(.+?)\s*=\s*['"]?([^'"\n]*)['"]?\s*$`)
	LSB_RELEASE_REGEX   = regexp.MustCompile(`(?m)^\s*(.+?)\s*=["']?(.+?)["']?$`)
	RHEL_PLATFORM_REGEX = regexp.MustCompile(`^(.+)\srelease`)
	RHEL_RELEASE_REGEX  = regexp.MustCompile(`release ([\d\.]+)`)
)

func ParseOsRelease(content string) (map[string]string, error) {
	return parseKeyValue(content, OS_RELEASE_REGEX), nil
}

func ParseLsbRelease(content string) (map[string]string, error) {
	return parseKeyValue(content, LSB_RELEASE_REGEX), nil
}

func parseKeyValue(content string, regex *regexp.Regexp) map[string]string {
	res := regex.FindAllStringSubmatch(content, -1)
	m := make(map[string]string)
	for _, value := range res {
		m[value[1]] = value[2]
	}
	return m
}

func ParseRhelVersion(releaseDescription string) (string, string, error) {
	// extract platform name
	m := RHEL_PLATFORM_REGEX.FindStringSubmatch(releaseDescription)
	if len(m) < 1 {
		return "", "", errors.New("could not parse rhel version")
	}
	name := m[1]

	// extract release
	n := RHEL_RELEASE_REGEX.FindStringSubmatch(releaseDescription)
	if len(n) < 2 {
		return "", "", errors.New("could not parse rhel version")
	}
	release := n[1]

	return name, release, nil
}

var (
	suseTitle        = regexp.MustCompile(`^([\w\d\s\.]+)\(([\w\d\s\.]+)\)`)
	suseVersion      = regexp.MustCompile(`VERSION\s*=\s*"?([\d\.]+)"?`)
	susePatchVersion = regexp.MustCompile(`VERSION = (\d+)\nPATCHLEVEL = (\d+)`)
)

func ParseSuseTitle(v string) (string, string) {
	m := suseTitle.FindStringSubmatch(v)
	if len(m) == 0 {
		return "", ""
	}

	return strings.TrimSpace(m[1]), strings.TrimSpace(m[2])
}

func ParseSuseVersion(v string) string {
	m := susePatchVersion.FindStringSubmatch(v)
	if len(m) == 3 {
		return m[1] + "." + m[2]
	}

	m = suseVersion.FindStringSubmatch(v)
	if len(m) == 0 {
		return ""
	}

	return m[1]
}
