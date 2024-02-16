// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package npm

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"regexp"
	"strings"

	"github.com/rs/zerolog/log"
	"sigs.k8s.io/yaml"
)

var (
	_ Parser = (*YarnLockParser)(nil)
)

type yarnLock map[string]yarnLockEntry

type yarnLockEntry struct {
	Version      string
	Resolved     string
	Dependencies map[string]string
}

type YarnLockParser struct{}

func (p *YarnLockParser) Parse(r io.Reader, filename string) (NpmPackageInfo, error) {
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

func (p *yarnLock) Root() *Package {
	// we don't have a root package in yarn.lock
	return nil
}

func (p *yarnLock) Direct() []*Package {
	return nil
}

func (p *yarnLock) Transitive() []*Package {
	transitive := []*Package{}

	// add all dependencies
	for k, v := range *p {
		name, _, err := parseYarnPackageName(k)
		if err != nil {
			log.Error().Str("name", name).Msg("cannot parse yarn package name")
			continue
		}
		transitive = append(transitive, &Package{
			Name:    name,
			Version: v.Version,
			Purl:    NewPackageUrl(name, v.Version),
			Cpes:    NewCpes(name, v.Version),
		})
	}

	return transitive
}

func parseYarnPackageName(name string) (string, string, error) {
	// a yarn package line may include may items
	pkgNames := strings.Split(name, ",")

	if len(pkgNames) == 0 {
		// something wrong
		return "", "", errors.New("cannot parse yarn package name")
	}

	parse := regexp.MustCompile(`^(.*)@(.*)$`)
	m := parse.FindStringSubmatch(strings.TrimSpace(pkgNames[0]))
	return m[1], m[2], nil
}
