package packages

import (
	"bufio"
	"io"
	"regexp"
)

var (
	PACMAN_REGEX = regexp.MustCompile(`^([\w-]*)\s([\w\d-+.:]+)$`)
)

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
