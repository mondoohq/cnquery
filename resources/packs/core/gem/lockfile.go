package gem

import (
	"bufio"
	"errors"
	"go.mondoo.io/mondoo/vadvisor"
	"io"
	"regexp"
	"strings"

	"github.com/rs/zerolog/log"
)

var (
	Specline     = regexp.MustCompile(`^\s*(.*)\s\((.*)\)\s*$`)
	GIT          = "GIT"
	PATH         = "PATH"
	GEM          = "GEM"
	DEPENDENCIES = "DEPENDENCIES"
)

func ParseGemfileLock(r io.Reader) ([]*vadvisor.Package, error) {
	pkgs := []*vadvisor.Package{}
	state := "INIT"

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()

		// check if we get a state change
		newState := strings.TrimSpace(line)
		switch newState {
		case GIT:
			state = newState
		case PATH:
			state = newState
		case GEM:
			log.Debug().Msg("GEM state")
			state = newState
		case DEPENDENCIES:
			state = newState
		}

		var err error
		var pkg *vadvisor.Package
		switch state {
		case GIT:
			fallthrough
		case PATH:
			fallthrough
		case GEM:
			pkg, err = parseSpecLine(line)
			if err != nil {
				log.Error().Err(err).Str("line", line).Msg("cannot parse gem package name")
				continue
			}
			if pkg != nil {
				pkgs = append(pkgs, pkg)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return pkgs, nil
}

func parseSpecLine(line string) (*vadvisor.Package, error) {
	// ignore everthing with 2 leading spaces, we do not need that info
	whitespace := LeadingSpaces(line)
	// We do not need to scan whitespace = 6 since those are just dependencies
	// of the package. At this point, we do not need the package graph
	if whitespace == 4 {
		name, version, err := ParsePackagename(line)
		if err != nil {
			return nil, err
		}

		return &vadvisor.Package{
			Name:      name,
			Version:   version,
			Format:    "gem",
			Namespace: "gem",
		}, nil
	}
	return nil, nil
}

func ParsePackagename(line string) (string, string, error) {
	m := Specline.FindStringSubmatch(line)
	if len(m) == 3 {
		return m[1], m[2], nil
	} else {
		return "", "", errors.New("cannot parse " + line)
	}
}

func LeadingSpaces(line string) int {
	i := 0
	for _, runeValue := range line {
		if runeValue == ' ' {
			i++
		} else {
			break
		}
	}
	return i
}
