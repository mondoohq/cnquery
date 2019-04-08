package php

import (
	"encoding/json"
	"io"
	"io/ioutil"

	"go.mondoo.io/mondoo/vadvisor/api"
)

type ComposerPackage struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type ComposerLock struct {
	Readme      []string          `json:"_readme"`
	Hash        string            `json:"content-hash"`
	Packages    []ComposerPackage `json:"packages"`
	DevPackages []ComposerPackage `json:"packages-dev"`
}

func ParseComposerLock(r io.Reader) ([]*api.Package, error) {
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	var composerLock ComposerLock
	err = json.Unmarshal(data, &composerLock)
	if err != nil {
		return nil, err
	}

	entries := []*api.Package{}

	// add dependencies
	for i := range composerLock.Packages {
		pkg := composerLock.Packages[i]
		entries = append(entries, &api.Package{
			Name:      pkg.Name,
			Version:   pkg.Version,
			Format:    "php",
			Namespace: "php",
		})
	}

	return entries, nil
}
