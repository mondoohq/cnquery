package packages

import (
	"bufio"
	"io"
	"regexp"
	"strings"
)

var (
	RPM_REGEX = regexp.MustCompile(`^([\w-+]*)\s(\d*|\(none\)):([\w\d-+.:]+)\s([\w\d]*|\(none\))\s(.*)$`)
)

// ParseRpmPackages parses output from:
// rpm -qa --queryformat '%{NAME} %{EPOCHNUM}:%{VERSION}-%{RELEASE} %{ARCH} %{SUMMARY}\n'
func ParseRpmPackages(input io.Reader) []Package {
	pkgs := []Package{}
	scanner := bufio.NewScanner(input)
	for scanner.Scan() {
		line := scanner.Text()
		m := RPM_REGEX.FindStringSubmatch(line)
		if m != nil {
			var version string
			// only append the epoch if we have a non-zero value
			if m[2] == "0" || strings.TrimSpace(m[2]) == "(none)" {
				version = m[3]
			} else {
				version = m[2] + ":" + m[3]
			}

			arch := m[4]
			// if no arch provided, remove it completely
			if arch == "(none)" {
				arch = ""
			}

			pkgs = append(pkgs, Package{Name: m[1], Version: version, Arch: arch, Description: m[5]})
		}
	}
	return pkgs
}
