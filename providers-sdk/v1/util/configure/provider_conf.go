// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

type ProvidersConf struct {
	Builtin []Builtin `json:"builtin"`

	_providers []string // Providers names
}

func (c ProvidersConf) Providers() []string {
	if len(c._providers) == 0 {
		for _, b := range c.Builtin {
			c._providers = append(c._providers, b.Name)
		}
	}
	return c._providers
}

type Builtin struct {
	Name      string
	Remote    string
	GoPackage string
}

func (b Builtin) Cwd() string {
	if b.Remote != "" {
		return b.Remote
	}

	return ""
}

func (b Builtin) Resource() string {
	if b.Remote != "" {
		return fmt.Sprintf("%s/resources/%s.resources.json", b.Remote, b.Name)
	}
	return "providers/" + b.Name + "/resources/" + b.Name + ".resources.json"
}

func (b Builtin) Dist() string {
	return "providers/" + b.Name + ".resources.json"
}

func (b Builtin) GoMod() string {
	if b.Remote != "" {
		return b.Remote + "/go.mod"
	}
	return "providers/" + b.Name + "/go.mod"
}

func (b *Builtin) UnmarshalJSON(data []byte) error {
	var name string
	if err := json.Unmarshal(data, &name); err == nil {
		b.Name = name
		b.GoPackage = "go.mondoo.com/cnquery/v11/providers/" + name

		return nil
	}

	var raw struct {
		Name      string `json:"name"`
		Remote    string `json:"remote"`
		GoPackage string `json:"goPackage"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	b.Name = raw.Name
	b.Remote = raw.Remote
	if strings.HasPrefix(b.Remote, "~/") {
		b.Remote = os.ExpandEnv(strings.Replace(b.Remote, "~/", "$HOME/", 1))
	}

	b.GoPackage = raw.GoPackage

	return nil
}

func (b *Builtin) MarshalJSON() ([]byte, error) {
	if b.Remote == "" && b.GoPackage == "" {
		return json.Marshal(b.Name)
	}

	return json.Marshal(struct {
		Name      string `json:"name"`
		Remote    string `json:"remote"`
		GoPackage string `json:"goPackage"`
	}{
		Name:      b.Name,
		Remote:    b.Remote,
		GoPackage: b.GoPackage,
	})
}
