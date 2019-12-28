package packages

import (
	"bufio"
	"io"

	"github.com/rs/zerolog/log"
	"regexp"
)

var (
	APK_REGEX = regexp.MustCompile(`^([A-Za-z]):(.*)$`)
)

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
