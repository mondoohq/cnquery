// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package yarnlock

import (
	"bufio"
	"bytes"
	"io"
	"regexp"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/providers/os/resources/languages"
	"go.mondoo.com/mql/v13/providers/os/resources/languages/javascript"
	"sigs.k8s.io/yaml"
)

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

	// iterate and convert the format to yaml on the fly
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()

		reStr := regexp.MustCompile(`^(\s*.*)\s\"(.*)$`)
		repStr := "${1}: \"$2"
		line = reStr.ReplaceAllString(line, repStr)

		b.Write([]byte(line))
		b.Write([]byte("\n"))
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
		})
	}

	return transitive
}
