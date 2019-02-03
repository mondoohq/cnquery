package parser

import (
	"bufio"
	"errors"
	"io"
	"io/ioutil"
	"regexp"
	"strings"

	"github.com/rs/zerolog/log"
	plist "howett.net/plist"
)

type Package struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Arch        string `json:"arch"`
	Status      string `json:"status,omitempty"`
	Description string `json:"description"`

	// this may be the source package or an origin
	// e.g. on alpine it is used for parent  packages
	// o 	Package Origin - https://wiki.alpinelinux.org/wiki/Apk_spec
	Origin string `json:"origin"`
}

var (
	DPKG_REGEX   = regexp.MustCompile(`^(.+):\s(.+)$`)
	RPM_REGEX    = regexp.MustCompile(`^([\w-+]*)\s(\d*):([\w\d-+.:]+)\s([\w\d]*|\(none\))\s(.*)$`)
	PACMAN_REGEX = regexp.MustCompile(`^([\w-]*)\s([\w\d-+.:]+)$`)
	APK_REGEX    = regexp.MustCompile(`^([A-Za-z]):(.*)$`)
)

// parse the dpkg database content located in /var/lib/dpkg/status
func ParseDpkgPackages(input io.Reader) ([]Package, error) {
	const STATE_RESET = 0
	const STATE_DESC = 1
	pkgs := []Package{}

	add := func(pkg Package) {
		// do sanitization checks to ensure we have minimal information
		if pkg.Name != "" && pkg.Version != "" {
			pkgs = append(pkgs, pkg)
		} else {
			log.Debug().Msg("ignored apk package since information is missing")
		}
	}

	scanner := bufio.NewScanner(input)
	pkg := Package{}
	state := STATE_RESET
	var key string
	for scanner.Scan() {
		line := scanner.Text()

		// reset package definition once we reach a newline
		if len(line) == 0 {
			add(pkg)
			pkg = Package{}
		}

		m := DPKG_REGEX.FindStringSubmatch(line)
		key = ""
		if m != nil {
			key = m[1]
			state = STATE_RESET
		}
		switch {
		case key == "Package":
			pkg.Name = strings.TrimSpace(m[2])
		case key == "Version":
			pkg.Version = strings.TrimSpace(m[2])
		case key == "Architecture":
			pkg.Arch = strings.TrimSpace(m[2])
		case key == "Status":
			pkg.Status = strings.TrimSpace(m[2])
		case key == "Source":
			pkg.Origin = strings.TrimSpace(m[2])
		// description supports multi-line statements, start desc
		case key == "Description":
			pkg.Description = strings.TrimSpace(m[2])
			state = STATE_DESC
		// next desc line, append to previous one
		case state == STATE_DESC:
			pkg.Description += "\n" + strings.TrimSpace(line)
		}
	}

	// if the last line is not an empty line we have things in flight, lets check it
	add(pkg)

	return pkgs, nil
}

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
			if m[2] == "0" {
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

func ParsePacmanPackages(input io.Reader) []Package {
	pkgs := []Package{}
	scanner := bufio.NewScanner(input)
	for scanner.Scan() {
		line := scanner.Text()
		m := PACMAN_REGEX.FindStringSubmatch(line)
		if m != nil {
			pkgs = append(pkgs, Package{Name: m[1], Version: m[2]})
		}
	}
	return pkgs
}

// ParseApkDbPackages parses the database of the apk package manager located in
// `/lib/apk/db/installed`
// Apk spec: https://wiki.alpinelinux.org/wiki/Apk_spec
func ParseApkDbPackages(input io.Reader) []Package {
	pkgs := []Package{}

	var pkgVersion string
	var pkgEpoch string

	add := func(pkg Package) {
		// merge version and epoch
		if pkgEpoch == "0" || pkgEpoch == "" {
			pkg.Version = pkgVersion
		} else {
			pkg.Version = pkgEpoch + ":" + pkgVersion
		}

		// do sanitization checks to ensure we have minimal information
		if pkg.Name != "" && pkg.Version != "" {
			pkgs = append(pkgs, pkg)
		} else {
			log.Debug().Msg("ignored apk package since information is missing")
		}
	}

	scanner := bufio.NewScanner(input)
	pkg := Package{}
	var key string
	for scanner.Scan() {
		line := scanner.Text()

		// reset package definition once we reach a newline
		if len(line) == 0 {
			add(pkg)
			// reset values
			pkgEpoch = ""
			pkgVersion = ""
			pkg = Package{}
		}

		m := APK_REGEX.FindStringSubmatch(line)
		key = ""
		if m != nil {
			key = m[1]
		}

		// if we short line, we ignore it since this is not a valid line
		if len(line) < 2 {
			continue
		}

		// Parse the package name or version.
		switch key {
		case "P":
			pkg.Name = m[2] // package name
		case "V":
			pkgVersion = m[2] // package version
		case "A":
			pkg.Arch = m[2] // architecture
		case "t":
			pkgEpoch = m[2] // epoch
		case "o":
			pkg.Origin = m[2] // origin
		case "T":
			pkg.Description = m[2] // description
		}
	}

	// if the last line is not an empty line we have things in flight, lets check it
	add(pkg)
	return pkgs
}

// parse macos system version property list
func ParseMacOSPackages(input io.Reader) ([]Package, error) {
	var r io.ReadSeeker
	r, ok := input.(io.ReadSeeker)

	// if the read seaker is not implemented lets cache stdout in-memory
	if !ok {
		packageList, err := ioutil.ReadAll(input)
		if err != nil {
			return nil, err
		}
		r = strings.NewReader(string(packageList))
	}

	type sysProfilerItems struct {
		Name    string `plist:"_name"`
		Version string `plist:"version"`
	}

	type sysProfiler struct {
		Items []sysProfilerItems `plist:"_items"`
	}

	var data []sysProfiler
	decoder := plist.NewDecoder(r)
	err := decoder.Decode(&data)
	if err != nil {
		return nil, err
	}

	if len(data) != 1 {
		return nil, errors.New("format not supported")
	}

	pkgs := make([]Package, len(data[0].Items))
	for i, entry := range data[0].Items {
		pkgs[i].Name = entry.Name
		pkgs[i].Version = entry.Version
	}

	return pkgs, nil
}
