package mvd

import (
	"strings"
)

type BySeverity []*Advisory

func (s BySeverity) Len() int {
	return len(s)
}

func (s BySeverity) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s BySeverity) Less(i, j int) bool {
	return s[i].Score < s[j].Score
}

type ByPkgSeverity []*Package

func (s ByPkgSeverity) Len() int {
	return len(s)
}

func (s ByPkgSeverity) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

// sort first by score, then by reverse name
func (s ByPkgSeverity) Less(i, j int) bool {
	if s[i].Score == s[j].Score {
		return strings.ToLower(s[i].Name) > strings.ToLower(s[j].Name)
	}

	return s[i].Score < s[j].Score
}

func FilterByAffected(pkgs []*Package) []*Package {
	filtered := []*Package{}
	for i := range pkgs {
		if pkgs[i].Affected == true {
			filtered = append(filtered, pkgs[i])
		}
	}
	return filtered
}
