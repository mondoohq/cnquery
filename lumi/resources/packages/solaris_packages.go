package packages

import (
	"bufio"
	"fmt"
	"io"
	"regexp"

	motor "go.mondoo.io/mondoo/motor/motoros"
)

var (
	SOLARIS_PKG_REGEX  = regexp.MustCompile(`^(.*)\s+([\w\-]+)$`)
	SOLARIS_FMRI_REGEX = regexp.MustCompile(`^pkg:\/\/([\w]+)/(.*)@(.*),(.*):(.*)$`)
)

type SolarisPackage struct {
	Publisher string
	Name      string
	Version   string
	Branch    string
	Timestamp string
}

// parses a FMRI (Fault Managed Resource Indicator) like:
// pkg://solaris/diagnostic/wireshark@1.4.2,5.11-0.174:20110128T0635Z
// Publisher: solaris
// Package name: diagnostic/wireshark
// Component version: 1.4.2
// Build version: 5.11
// Branch version: 0.174
// Package timestamp: 20110128T0635Z
func ParseSolarisFmri(frmi string) (*SolarisPackage, error) {

	m := SOLARIS_FMRI_REGEX.FindStringSubmatch(frmi)
	if len(m) != 6 {
		return nil, fmt.Errorf("could not parse solaris package name: %s", frmi)
	}

	return &SolarisPackage{
		Publisher: m[1],
		Name:      m[2],
		Version:   m[3],
		Branch:    m[4],
		Timestamp: m[5],
	}, nil

}

// parse solaris package list
// see https://docs.oracle.com/cd/E23824_01/html/E21802/gkoic.html
// see https://www.oracle.com/technetwork/server-storage/solaris11/documentation/ips-one-liners-032011-337775.pdf
func ParseSolarisPackages(input io.Reader) []Package {
	pkgs := []Package{}
	scanner := bufio.NewScanner(input)
	for scanner.Scan() {
		line := scanner.Text()
		m := SOLARIS_PKG_REGEX.FindStringSubmatch(line)
		if m != nil && len(m) >= 1 {
			// TODO: check that it has the i flag
			spkg, err := ParseSolarisFmri(m[0])
			if err == nil {
				pkgs = append(pkgs, Package{
					Name:    spkg.Name,
					Version: spkg.Version,
					Format:  "ips",
				})
			}
		}
	}
	return pkgs
}

type SolarisPkgManager struct {
	motor *motor.Motor
}

func (s *SolarisPkgManager) Name() string {
	return "Solaris Package Manager"
}

func (script *SolarisPkgManager) Format() string {
	return "ips"
}

func (s *SolarisPkgManager) List() ([]Package, error) {
	cmd, err := s.motor.Transport.RunCommand("pkg list -v")
	if err != nil {
		return nil, fmt.Errorf("could not read solaris package list")
	}

	return ParseSolarisPackages(cmd.Stdout), nil
}

func (s *SolarisPkgManager) Available() (map[string]PackageUpdate, error) {
	return map[string]PackageUpdate{}, nil
}
