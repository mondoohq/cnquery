// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package gemspec

import (
	"bufio"
	"io"
	"regexp"
	"strings"

	"go.mondoo.com/mql/v13/providers/os/resources/languages"
	"go.mondoo.com/mql/v13/providers/os/resources/languages/ruby"
)

var (
	_ languages.Extractor = (*Extractor)(nil)
	_ languages.Bom       = (*gemSpec)(nil)
)

// Patterns for extracting gemspec metadata via static analysis.
// These match common patterns without executing Ruby.
var (
	// spec.name = "foo" or spec.name = 'foo'
	namePattern = regexp.MustCompile(`\.name\s*=\s*["']([^"']+)["']`)
	// spec.version = "1.0.0" or spec.version = '1.0.0'
	versionPattern = regexp.MustCompile(`\.version\s*=\s*["']([^"']+)["']`)
	// spec.add_dependency "foo", "~> 1.0"
	// spec.add_runtime_dependency "foo", "~> 1.0"
	// spec.add_development_dependency "foo", "~> 1.0"
	depPattern = regexp.MustCompile(`\.(add_dependency|add_runtime_dependency|add_development_dependency)\s+["']([^"']+)["'](?:,\s*["']([^"']+)["'])?`)
)

// Extractor parses .gemspec files to extract gem metadata via static analysis.
type Extractor struct{}

func (e *Extractor) Name() string {
	return "gemspec"
}

func (e *Extractor) Parse(r io.Reader, filename string) (languages.Bom, error) {
	spec, err := parseGemspec(r)
	if err != nil {
		return nil, err
	}

	if filename != "" {
		spec.evidence = append(spec.evidence, filename)
	}

	return spec, nil
}

// parseGemspec reads a .gemspec file and extracts metadata via regex matching.
// This is static analysis — it does not execute Ruby code.
func parseGemspec(r io.Reader) (*gemSpec, error) {
	spec := &gemSpec{}

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()

		if m := namePattern.FindStringSubmatch(line); len(m) > 1 {
			spec.Name = m[1]
		}

		if m := versionPattern.FindStringSubmatch(line); len(m) > 1 {
			spec.Version = m[1]
		}

		if m := depPattern.FindStringSubmatch(line); len(m) > 2 {
			depType := m[1]
			depName := m[2]
			constraint := ""
			if len(m) > 3 {
				constraint = m[3]
			}

			spec.Dependencies = append(spec.Dependencies, gemDep{
				Name:       depName,
				Constraint: constraint,
				IsDev:      depType == "add_development_dependency",
			})
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return spec, nil
}

// Root returns the gem itself as the root package.
func (s *gemSpec) Root() *languages.Package {
	if s.Name == "" {
		return nil
	}

	return &languages.Package{
		Name:         s.Name,
		Version:      s.Version,
		Purl:         ruby.NewPackageUrl(s.Name, s.Version),
		Cpes:         ruby.NewCpes(s.Name, s.Version),
		EvidenceList: ruby.NewEvidenceList(s.evidence),
	}
}

// Direct returns non-dev dependencies.
func (s *gemSpec) Direct() languages.Packages {
	var direct languages.Packages
	for _, dep := range s.Dependencies {
		if dep.IsDev {
			continue
		}
		direct = append(direct, makeDepPackage(dep, s.evidence))
	}
	return direct
}

// Transitive returns all declared dependencies (gemspec only has direct deps).
func (s *gemSpec) Transitive() languages.Packages {
	var all languages.Packages
	for _, dep := range s.Dependencies {
		all = append(all, makeDepPackage(dep, s.evidence))
	}
	return all
}

func makeDepPackage(dep gemDep, evidence []string) *languages.Package {
	// Always generate a PURL — version is optional per the PURL spec.
	// Exact pins ("= 1.2.3") get a versioned PURL; constraints get versionless.
	version := dep.Constraint
	purlVersion := ""
	if !isVersionConstraint(dep.Constraint) {
		purlVersion = extractPinnedVersion(dep.Constraint)
	}

	return &languages.Package{
		Name:         dep.Name,
		Version:      version,
		Purl:         ruby.NewPackageUrl(dep.Name, purlVersion),
		Cpes:         ruby.NewCpes(dep.Name, purlVersion),
		EvidenceList: ruby.NewEvidenceList(evidence),
	}
}

// isVersionConstraint returns true if the string is a version constraint
// rather than an exact version. Exact pins like "= 1.2.3" are NOT constraints
// — they resolve to a specific version.
func isVersionConstraint(v string) bool {
	if v == "" {
		return true
	}
	// Exact pin: "= 1.2.3" (single = followed by space and version)
	if strings.HasPrefix(v, "= ") {
		return false
	}
	return strings.ContainsAny(v, "~><=! ")
}

// extractPinnedVersion extracts the version from an exact pin constraint "= X.Y.Z".
func extractPinnedVersion(v string) string {
	if strings.HasPrefix(v, "= ") {
		return strings.TrimPrefix(v, "= ")
	}
	return v
}
