// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package packagejson

import (
	"encoding/json"
	"io"

	"go.mondoo.com/cnquery/v11/providers/os/resources/languages"
	"go.mondoo.com/cnquery/v11/providers/os/resources/languages/javascript"
)

var (
	_ languages.Extractor = (*Extractor)(nil)
	_ languages.Bom       = (*packageJson)(nil)
)

type Extractor struct{}

func (p *Extractor) Name() string {
	return "packagejson"
}

func (p *Extractor) Parse(r io.Reader, filename string) (languages.Bom, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	var packageJson packageJson
	err = json.Unmarshal(data, &packageJson)
	if err != nil {
		return nil, err
	}

	if filename != "" {
		packageJson.evidence = append(packageJson.evidence, filename)
	}

	return &packageJson, nil
}

func (p *packageJson) Root() *languages.Package {
	// root package
	root := &languages.Package{
		Name:         p.Name,
		Version:      p.Version,
		Purl:         javascript.NewPackageUrl(p.Name, p.Version),
		Cpes:         javascript.NewCpes(p.Name, p.Version),
		EvidenceList: javascript.NewEvidenceList(p.evidence),
	}

	return root
}

func (p *packageJson) Direct() languages.Packages {
	return nil
}

func (p *packageJson) Transitive() languages.Packages {
	// transitive dependencies, includes the root package
	var transitive languages.Packages

	r := p.Root()
	if r != nil {
		transitive = append(transitive, r)
	}

	for k, v := range p.Dependencies {
		transitive = append(transitive, &languages.Package{
			Name:         k,
			Version:      v,
			Purl:         javascript.NewPackageUrl(k, v),
			Cpes:         javascript.NewCpes(k, v),
			EvidenceList: javascript.NewEvidenceList(p.evidence),
		})
	}

	return transitive
}
