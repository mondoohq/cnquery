package parser

import (
	"bufio"
	"encoding/json"
	"encoding/xml"
	"io"
	"io/ioutil"
	"regexp"
)

// extends Package to store available version
type PackageUpdate struct {
	Package
	Available string `json:"available"`
	Repo      string `json:"repo"`
}

type OperatingSystemUpdate struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Severity    string `json:"severity"`
	Category    string `json:"category"`
	Restart     bool   `json:"restart"`
}

var (
	APK_UPDATE_REGEX  = regexp.MustCompile(`^([a-zA-Z0-9._]+)-([a-zA-Z0-9.\-\+]+)\s+<\s([a-zA-Z0-9.\-\+]+)\s*$`)
	DPKG_UPDATE_REGEX = regexp.MustCompile(`^Inst\s([a-zA-Z0-9.\-_]+)\s\[([a-zA-Z0-9.\-\+]+)\]\s\(([a-zA-Z0-9.\-\+]+)\s*(.*)\)(.*)$`)
)

func ParseApkUpdates(input io.Reader) ([]PackageUpdate, error) {
	var pkgs []PackageUpdate
	scanner := bufio.NewScanner(input)
	for scanner.Scan() {
		line := scanner.Text()
		m := APK_UPDATE_REGEX.FindStringSubmatch(line)
		if m != nil {
			pkgs = append(pkgs, PackageUpdate{
				Package:   Package{Name: m[1], Version: m[2]},
				Available: m[3],
			})
		}
	}
	return pkgs, nil
}

func ParseDpkgUpdates(input io.Reader) ([]PackageUpdate, error) {
	var pkgs []PackageUpdate
	scanner := bufio.NewScanner(input)
	for scanner.Scan() {
		line := scanner.Text()
		m := DPKG_UPDATE_REGEX.FindStringSubmatch(line)
		if m != nil {
			pkgs = append(pkgs, PackageUpdate{
				Package:   Package{Name: m[1], Version: m[2]},
				Available: m[3],
			})
		}
	}
	return pkgs, nil
}

func ParseRpmUpdates(input io.Reader) ([]PackageUpdate, error) {
	var pkgs []PackageUpdate
	scanner := bufio.NewScanner(input)
	for scanner.Scan() {
		line := scanner.Bytes()

		// we try to parse the content into the struct
		var pkg PackageUpdate
		err := json.Unmarshal(line, &pkg)
		if err != nil {
			// there are string lines that cannot be parsed
			continue
		}
		pkgs = append(pkgs, pkg)
	}
	return pkgs, nil
}

type zypperUpdate struct {
	Name        string `xml:"name,attr"`
	Kind        string `xml:"kind,attr"`
	Arch        string `xml:"arch,attr"`
	Edition     string `xml:"edition,attr"`
	OldEdition  string `xml:"edition-old,attr"`
	Status      string `xml:"status,attr"`
	Category    string `xml:"category,attr"`
	Severity    string `xml:"severity,attr"`
	PkgManager  string `xml:"pkgmanager,attr"`
	Restart     string `xml:"restart,attr"`
	Interactive string `xml:"interactive,attr"`

	Summary     string `xml:"summary"`
	Description string `xml:"description"`
}

type zypper struct {
	XMLNode xml.Name       `xml:"stream"`
	Updates []zypperUpdate `xml:"update-status>update-list>update"`
	Blocked []zypperUpdate `xml:"update-status>blocked-update-list>update"`
}

// for Suse, updates are package updates
func ParseZypperUpdates(input io.Reader) ([]PackageUpdate, error) {
	zypper, err := parseZypper(input)
	if err != nil {
		return nil, err
	}

	var pkgs []PackageUpdate
	for _, u := range zypper.Updates {
		// filter for kind package
		if u.Kind != "package" {
			continue
		}

		pkgs = append(pkgs, PackageUpdate{
			Package:   Package{Name: u.Name, Version: u.OldEdition, Arch: u.Arch},
			Available: u.Edition,
		})
	}
	return pkgs, nil
}

// for Suse, patches are operating system patches that are composed of multiple package updates
func ParseZypperPatches(input io.Reader) ([]OperatingSystemUpdate, error) {
	zypper, err := parseZypper(input)
	if err != nil {
		return nil, err
	}

	var updates []OperatingSystemUpdate
	// filter for kind patch
	for _, u := range zypper.Updates {
		if u.Kind != "patch" {
			continue
		}

		restart := false
		if u.Restart == "true" {
			restart = true
		}

		updates = append(updates, OperatingSystemUpdate{
			Name:        u.Name,
			Severity:    u.Severity,
			Restart:     restart,
			Category:    u.Category,
			Description: u.Description,
		})
	}

	return updates, nil
}

func parseZypper(input io.Reader) (*zypper, error) {
	content, err := ioutil.ReadAll(input)
	if err != nil {
		return nil, err
	}
	var patches zypper
	err = xml.Unmarshal(content, &patches)
	if err != nil {
		return nil, err
	}
	return &patches, nil
}
