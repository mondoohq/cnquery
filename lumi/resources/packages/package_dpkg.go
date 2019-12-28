package packages

import (
	"bufio"
	"io"
	"regexp"
	"strings"

	"github.com/rs/zerolog/log"
)

var (
	DPKG_REGEX        = regexp.MustCompile(`^(.+):\s(.+)$`)
	DPKG_ORIGIN_REGEX = regexp.MustCompile(`^\s*([^\(]*)(?:\((.*)\))?\s*$`)
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
			o := DPKG_ORIGIN_REGEX.FindStringSubmatch(m[2])
			if o != nil && len(o) >= 1 {
				pkg.Origin = strings.TrimSpace(o[1])
			} else {
				log.Error().Str("origin", m[2]).Msg("cannot parse dpkg origin")
			}
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
