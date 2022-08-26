package php

import (
	"encoding/json"
	"go.mondoo.com/cnquery/vadvisor"
	"io"
	"io/ioutil"
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

func ParseComposerLock(r io.Reader) ([]*vadvisor.Package, error) {
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	var composerLock ComposerLock
	err = json.Unmarshal(data, &composerLock)
	if err != nil {
		return nil, err
	}

	entries := []*vadvisor.Package{}

	// add dependencies
	for i := range composerLock.Packages {
		pkg := composerLock.Packages[i]
		entries = append(entries, &vadvisor.Package{
			Name:      pkg.Name,
			Version:   pkg.Version,
			Format:    "php",
			Namespace: "php",
		})
	}

	return entries, nil
}
