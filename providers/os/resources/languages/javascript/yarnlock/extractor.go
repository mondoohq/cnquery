// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package yarnlock

import (
	"bufio"
	"bytes"
	"io"
	"regexp"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v11/providers/os/resources/languages"
	"go.mondoo.com/cnquery/v11/providers/os/resources/languages/javascript"
	"go.mondoo.com/cnquery/v11/sbom"
	"sigs.k8s.io/yaml"
)

var (
	_ languages.Extractor = (*Extractor)(nil)
	_ languages.Bom       = (*yarnLock)(nil)
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

	var yarnLock yarnLock

	err := yaml.Unmarshal(b.Bytes(), &yarnLock)
	if err != nil {
		return nil, err
	}

	return &yarnLock, nil
}

func (p *yarnLock) Root() *sbom.Package {
	// we don't have a root package in yarn.lock
	return nil
}

func (p *yarnLock) Direct() languages.Packages {
	return nil
}

func (p *yarnLock) Transitive() languages.Packages {
	var transitive languages.Packages

	// add all dependencies
	for k, v := range *p {
		name, _, err := parseYarnPackageName(k)
		if err != nil {
			log.Error().Str("name", name).Msg("cannot parse yarn package name")
			continue
		}
		transitive = append(transitive, &sbom.Package{
			Name:    name,
			Version: v.Version,
			Purl:    javascript.NewPackageUrl(name, v.Version),
			Cpes:    javascript.NewCpes(name, v.Version),
		})
	}

	return transitive
}
