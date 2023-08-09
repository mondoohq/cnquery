package packages

import (
	"bufio"
	"encoding/json"
	"encoding/xml"
	"io"
)

func ParseRpmUpdates(input io.Reader) (map[string]PackageUpdate, error) {
	pkgs := map[string]PackageUpdate{}
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
		pkgs[pkg.Name] = pkg
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
// parses the output of `zypper -n --xmlout list-updates`
func ParseZypperUpdates(input io.Reader) (map[string]PackageUpdate, error) {
	pkgs := map[string]PackageUpdate{}
	zypper, err := ParseZypper(input)
	if err != nil {
		return nil, err
	}

	for _, u := range zypper.Updates {
		// filter for kind package
		if u.Kind != "package" {
			continue
		}
		pkgs[u.Name] = PackageUpdate{
			Name:      u.Name,
			Version:   u.OldEdition,
			Arch:      u.Arch,
			Available: u.Edition,
		}
	}
	return pkgs, nil
}

func ParseZypper(input io.Reader) (*zypper, error) {
	content, err := io.ReadAll(input)
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
