// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package yarnlock

import (
	"bufio"
	"bytes"
	"io"
	"regexp"
	"strings"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/providers/os/resources/languages"
	"go.mondoo.com/mql/v13/providers/os/resources/languages/javascript"
	"sigs.k8s.io/yaml"
)

// Compiled once: matched against every line of a yarn.lock file.
var kv = regexp.MustCompile(`^(\s+)(\S+)\s+(.+?)\s*$`)

// yarnLockBom wraps a parsed yarnLock with file evidence.
type yarnLockBom struct {
	packages yarnLock
	evidence []string
}

var (
	_ languages.Extractor = (*Extractor)(nil)
	_ languages.Bom       = (*yarnLockBom)(nil)
)

type Extractor struct{}

func (p *Extractor) Name() string {
	return "yarnlock"
}

func (p *Extractor) Parse(r io.Reader, filename string) (languages.Bom, error) {
	var b bytes.Buffer

	// Convert the yarn.lock v1 (classic) pseudo-YAML to real YAML on the fly. The
	// classic format writes indented entries as `key value` (space-separated), not
	// `key: value` — and the value may be quoted (`version "1.3.8"`, `resolved
	// "url"`) OR unquoted (`integrity sha512-…`). Rewrite both forms to
	// `key: "value"`; pass through blank lines, comments, and mapping headers
	// (spec lines / `dependencies:` that already end in ":") untouched.
	scanner := bufio.NewScanner(r)
	// yarn.lock integrity/resolved lines can be long; grow the scanner buffer.
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.HasSuffix(strings.TrimRight(line, " "), ":") {
			b.WriteString(line + "\n")
			continue
		}
		if m := kv.FindStringSubmatch(line); m != nil {
			val := m[3]
			if !strings.HasPrefix(val, `"`) || !strings.HasSuffix(val, `"`) {
				val = `"` + val + `"`
			}
			b.WriteString(m[1] + m[2] + ": " + val + "\n")
			continue
		}
		b.WriteString(line + "\n")
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	var lock yarnLock

	err := yaml.Unmarshal(b.Bytes(), &lock)
	if err != nil {
		return nil, err
	}

	var result yarnLockBom
	result.packages = lock
	if filename != "" {
		result.evidence = append(result.evidence, filename)
	}

	return &result, nil
}

func (p *yarnLockBom) Root() *languages.Package {
	// we don't have a root package in yarn.lock
	return nil
}

func (p *yarnLockBom) Direct() languages.Packages {
	return nil
}

func (p *yarnLockBom) Transitive() languages.Packages {
	var transitive languages.Packages
	idx := p.packages.specIndex()

	// add all dependencies
	for k, v := range p.packages {
		name, _, err := parseYarnPackageName(k)
		if err != nil {
			log.Error().Str("name", name).Msg("cannot parse yarn package name")
			continue
		}
		transitive = append(transitive, &languages.Package{
			Name:         name,
			Version:      v.Version,
			Purl:         javascript.NewPackageUrl(name, v.Version),
			Cpes:         javascript.NewCpes(name, v.Version),
			EvidenceList: javascript.NewEvidenceList(p.evidence),
			DependsOn:    dependsOnRefs(idx, v.Dependencies),
			Hashes:       javascript.NewHashes(v.Integrity),
		})
	}

	return transitive
}
