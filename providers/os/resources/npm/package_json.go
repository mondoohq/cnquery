// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package npm

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"

	"go.mondoo.com/cnquery/v10/providers-sdk/v1/upstream/mvd"
)

// packageJson allows parsing the package json file
type packageJson struct {
	Name            string                `json:"name"`
	Description     string                `json:"description"`
	Version         string                `json:"version"`
	Private         bool                  `json:"private"`
	Homepage        string                `json:"homepage"`
	License         *packageJsonLicense   `json:"license"`
	Author          *packageJsonPeople    `json:"author"`
	Contributors    []packageJsonPeople   `json:"contributors"`
	Dependencies    map[string]string     `jsonn:"dependencies"`
	DevDependencies map[string]string     `jsonn:"devDependencies"`
	Repository      packageJsonRepository `json:"repository"`
	Engines         map[string]string     `jsonn:"engines"`
	CPU             []string              `json:"cpu"`
	OS              []string              `json:"os"`
}

// packageJsonPeople represents the author of the package
// https://docs.npmjs.com/cli/v10/configuring-npm/package-json#people-fields-author-contributors
type packageJsonPeople struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	URL   string `json:"url"`
}

// authorPattern parses the single-string author string:
//
//	^: Asserts the start of the string.
//	([^<]+): Captures one or more characters that are not <. This is for the author's name.
//	\s+: Matches one or more whitespace characters.
//	<([^>]+)>: Captures the email address within angle brackets.
//	\s+: Matches one or more whitespace characters again.
//	\(([^)]+)\): Captures the URL within parentheses.
//	$: Asserts the end of the string.
var authorPattern = regexp.MustCompile(`^([^<]+)\s+<([^>]+)>(?:\s+\(([^)]+)\))?$`)

// UnmarshalJSON implements the json.Unmarshaler interface
// package.json author can be a string or a structured object
func (a *packageJsonPeople) UnmarshalJSON(b []byte) error {
	var authorStr string
	type authorStruct packageJsonPeople
	author := authorStruct{}

	// try to unmarshal as structured object
	err := json.Unmarshal(b, &author)
	if err == nil {
		a.Name = author.Name
		a.Email = author.Email
		a.URL = author.URL
		return nil
	}

	// try to unmarshal as string
	err = json.Unmarshal(b, &authorStr)
	if err == nil {
		matches := authorPattern.FindStringSubmatch(authorStr)
		if len(matches) == 4 {
			a.Name = matches[1]
			a.Email = matches[2]
			a.URL = matches[3]
			return nil
		}
		// if the pattern does not match, we assume the string is the name
		a.Name = authorStr
		return nil
	}

	return errors.New("could not unmarshal author: " + string(b))
}

func (a *packageJsonPeople) String() string {
	b := strings.Builder{}
	b.WriteString(a.Name)
	if a.Email != "" {
		b.WriteString(fmt.Sprintf(" <%s>", a.Email))
	}
	if a.URL != "" {
		b.WriteString(fmt.Sprintf(" (%s)", a.URL))
	}
	return b.String()
}

type packageJsonRepository struct {
	Type      string `json:"type"`
	URL       string `json:"url"`
	Directory string `json:"directory"`
}

func (r *packageJsonRepository) UnmarshalJSON(b []byte) error {
	var repositoryStr string
	type repositoryStruct packageJsonRepository
	repo := repositoryStruct{}

	// try to unmarshal as structured object
	err := json.Unmarshal(b, &repo)
	if err == nil {
		r.Type = repo.Type
		r.URL = repo.URL
		r.Directory = repo.Directory
		return nil
	}

	// try to unmarshal as string
	err = json.Unmarshal(b, &repositoryStr)
	if err == nil {
		// handle case where the type is provided as prefix like `bitbucket:`
		parts := strings.SplitN(repositoryStr, ":", 2)
		if len(parts) == 2 {
			r.Type = parts[0]
			r.URL = parts[1]
			return nil
		} else {
			r.Type = "github"
			r.URL = repositoryStr
			return nil
		}
	}

	return errors.New("could not unmarshal repository: " + string(b))
}

type packageJsonLicense struct {
	Value string `json:"value"`
}

// UnmarshalJSON implements the json.Unmarshaler interface
// package.json license can be a string or a structured object
// For now we only support the plain string format and ignore the structured object
func (a *packageJsonLicense) UnmarshalJSON(b []byte) error {
	var licenseStr string

	// try to unmarshal as string
	err := json.Unmarshal(b, &licenseStr)
	if err == nil {
		a.Value = licenseStr
	}

	// we intentionally ignore the structured object
	return nil
}

type PackageJsonParser struct{}

func (p *PackageJsonParser) Parse(r io.Reader) ([]*mvd.Package, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	var packageJson packageJson
	err = json.Unmarshal(data, &packageJson)
	if err != nil {
		return nil, err
	}

	entries := []*mvd.Package{}

	// add own package
	entries = append(entries, &mvd.Package{
		Name:      packageJson.Name,
		Version:   packageJson.Version,
		Format:    "npm",
		Namespace: "nodejs",
	})

	// add all dependencies

	for k, v := range packageJson.Dependencies {
		entries = append(entries, &mvd.Package{
			Name:      k,
			Version:   v,
			Format:    "npm",
			Namespace: "nodejs",
		})
	}

	return entries, nil
}
